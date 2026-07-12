package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"
)

const (
	bankStatementAIMLFlagEnv         = "ENABLE_AIML_BANK_STATEMENT_ASSIST"
	bankStatementAIMLTimeoutEnv      = "BANK_STATEMENT_AI_TIMEOUT_MS"
	bankStatementAIMLMaxTextCharsEnv = "BANK_STATEMENT_AI_MAX_TEXT_CHARS"
	defaultBankStatementAITimeoutMS  = 3500
	defaultBankStatementAITextLimit  = 24000
	bankStatementAIMLMaxTokens       = 2200
	bankStatementValidationTolerance = 0.051
)

type bankStatementParseOutcome struct {
	Statement      *parsedStatement
	ImportMethod   string
	OCRConfidence  float64
	ValidationNote []string
}

type bankStatementValidation struct {
	Blocking             bool
	BlockingIssues       []string
	Warnings             []string
	BalanceMismatchCount int
	AmbiguousAmountCount int
	BothSidesCount       int
	ZeroDateCount        int
}

type bankStatementAIMLResponse struct {
	AccountNumber  string                  `json:"account_number"`
	IBAN           string                  `json:"iban"`
	Currency       string                  `json:"currency"`
	PeriodStart    string                  `json:"period_start"`
	PeriodEnd      string                  `json:"period_end"`
	OpeningBalance float64                 `json:"opening_balance"`
	ClosingBalance float64                 `json:"closing_balance"`
	TotalDebits    float64                 `json:"total_debits"`
	TotalCredits   float64                 `json:"total_credits"`
	Lines          []bankStatementAIMLLine `json:"lines"`
}

type bankStatementAIMLLine struct {
	LineNumber  int     `json:"line_number"`
	Date        string  `json:"date"`
	ValueDate   string  `json:"value_date"`
	Reference   string  `json:"reference"`
	Description string  `json:"description"`
	Debit       float64 `json:"debit"`
	Credit      float64 `json:"credit"`
	Balance     float64 `json:"balance"`
}

func (a *App) parseImportedPDFBankStatement(extractedText string) (*bankStatementParseOutcome, error) {
	localParsed, source, localErr := parseBankStatementLocally(extractedText)
	if localErr == nil && localParsed != nil {
		validation := validateParsedStatement(localParsed)
		if !validation.Blocking && !shouldEscalateBankStatementToAIML(source, validation, len(localParsed.Lines)) {
			return &bankStatementParseOutcome{
				Statement:      localParsed,
				ImportMethod:   source,
				OCRConfidence:  0.84,
				ValidationNote: validation.Warnings,
			}, nil
		}

		if !isBankStatementAIMLAssistEnabled() {
			if validation.Blocking {
				return nil, fmt.Errorf("local parser produced invalid statement: %s", strings.Join(validation.BlockingIssues, "; "))
			}
			return &bankStatementParseOutcome{
				Statement:      localParsed,
				ImportMethod:   source,
				OCRConfidence:  0.78,
				ValidationNote: validation.Warnings,
			}, nil
		}

		aimlParsed, model, aiErr := parseBankStatementWithAIML(extractedText)
		if aiErr == nil && aimlParsed != nil {
			aiValidation := validateParsedStatement(aimlParsed)
			if !aiValidation.Blocking {
				log.Printf("✅ Bank statement AI assist recovered structured statement using %s", model)
				return &bankStatementParseOutcome{
					Statement:      aimlParsed,
					ImportMethod:   fmt.Sprintf("PDF_OCR_AIML:%s", model),
					OCRConfidence:  0.93,
					ValidationNote: aiValidation.Warnings,
				}, nil
			}
			log.Printf("⚠️ Bank statement AI assist returned invalid statement: %s", strings.Join(aiValidation.BlockingIssues, "; "))
		} else if aiErr != nil {
			log.Printf("⚠️ Bank statement AI assist unavailable, keeping local parse path: %v", aiErr)
		}

		if validation.Blocking {
			return nil, fmt.Errorf("statement failed validation after local and AI parsing: %s", strings.Join(validation.BlockingIssues, "; "))
		}
		return &bankStatementParseOutcome{
			Statement:      localParsed,
			ImportMethod:   source,
			OCRConfidence:  0.76,
			ValidationNote: validation.Warnings,
		}, nil
	}

	if !isBankStatementAIMLAssistEnabled() {
		return nil, localErr
	}

	aimlParsed, model, aiErr := parseBankStatementWithAIML(extractedText)
	if aiErr != nil {
		if localErr != nil {
			return nil, fmt.Errorf("%w; AI assist also failed: %v", localErr, aiErr)
		}
		return nil, aiErr
	}

	aiValidation := validateParsedStatement(aimlParsed)
	if aiValidation.Blocking {
		return nil, fmt.Errorf("AI-assisted parser produced invalid statement: %s", strings.Join(aiValidation.BlockingIssues, "; "))
	}

	return &bankStatementParseOutcome{
		Statement:      aimlParsed,
		ImportMethod:   fmt.Sprintf("PDF_OCR_AIML:%s", model),
		OCRConfidence:  0.92,
		ValidationNote: aiValidation.Warnings,
	}, nil
}

func parseBankStatementLocally(text string) (*parsedStatement, string, error) {
	parsed, err := parseNBBFormat(text)
	if err == nil && parsed != nil {
		return parsed, "PDF_OCR_NBB", nil
	}

	parsed, kfhErr := parseKFHFormat(text)
	if kfhErr == nil && parsed != nil {
		return parsed, "PDF_OCR_KFH", nil
	}

	parsed, alSalamErr := parseAlSalamOCRFormat(text)
	if alSalamErr == nil && parsed != nil {
		return parsed, "PDF_OCR_AL_SALAM", nil
	}

	parsed, genericErr := parseGenericBankFormat(text)
	if genericErr == nil && parsed != nil {
		return parsed, "PDF_OCR_GENERIC", nil
	}

	if err != nil {
		return nil, "", fmt.Errorf("%w; kfh parser: %v; al salam parser: %v; generic parser: %v", err, kfhErr, alSalamErr, genericErr)
	}
	if kfhErr != nil || alSalamErr != nil {
		return nil, "", fmt.Errorf("kfh parser: %v; al salam parser: %v; generic parser: %v", kfhErr, alSalamErr, genericErr)
	}
	return nil, "", genericErr
}

func validateParsedStatement(parsed *parsedStatement) bankStatementValidation {
	validation := bankStatementValidation{}
	if parsed == nil {
		validation.Blocking = true
		validation.BlockingIssues = append(validation.BlockingIssues, "statement is empty")
		return validation
	}
	if len(parsed.Lines) == 0 {
		validation.Blocking = true
		validation.BlockingIssues = append(validation.BlockingIssues, "statement has no transaction lines")
		return validation
	}

	var computedDebits float64
	var computedCredits float64
	computedDebitCount := 0
	computedCreditCount := 0
	headerDebitCount := parsed.DebitCount
	headerCreditCount := parsed.CreditCount
	headerDebitTotal := parsed.TotalDebits
	headerCreditTotal := parsed.TotalCredits

	previousBalance := parsed.OpeningBalance
	previousBalanceKnown := previousBalance != 0
	if !previousBalanceKnown && len(parsed.Lines) > 0 {
		first := parsed.Lines[0]
		if first.Balance != 0 {
			previousBalance = first.Balance + first.Debit - first.Credit
			previousBalanceKnown = true
		}
	}

	for idx, line := range parsed.Lines {
		if line.Date.IsZero() {
			validation.ZeroDateCount++
		}
		if line.Debit > 0 && line.Credit > 0 {
			validation.BothSidesCount++
		}
		if line.Debit == 0 && line.Credit == 0 {
			validation.AmbiguousAmountCount++
		}
		computedDebits += line.Debit
		computedCredits += line.Credit
		if line.Debit > 0 {
			computedDebitCount++
		}
		if line.Credit > 0 {
			computedCreditCount++
		}

		if previousBalanceKnown && line.Balance != 0 {
			expectedBalance := previousBalance + line.Credit - line.Debit
			if absFloat(expectedBalance-line.Balance) > bankStatementValidationTolerance {
				validation.BalanceMismatchCount++
			}
		}
		if line.Balance != 0 || idx == len(parsed.Lines)-1 {
			previousBalance = line.Balance
			previousBalanceKnown = line.Balance != 0
		}
	}

	if validation.BothSidesCount > 0 {
		validation.Blocking = true
		validation.BlockingIssues = append(validation.BlockingIssues, fmt.Sprintf("%d lines contain both debit and credit amounts", validation.BothSidesCount))
	}
	if validation.ZeroDateCount > maxInt(1, len(parsed.Lines)/4) {
		validation.Blocking = true
		validation.BlockingIssues = append(validation.BlockingIssues, "too many lines are missing transaction dates")
	}
	if validation.BalanceMismatchCount > maxInt(1, len(parsed.Lines)/5) {
		validation.Blocking = true
		validation.BlockingIssues = append(validation.BlockingIssues, fmt.Sprintf("%d lines break running-balance continuity", validation.BalanceMismatchCount))
	}

	if parsed.OpeningBalance != 0 && len(parsed.Lines) > 0 {
		expectedClosing := parsed.OpeningBalance + computedCredits - computedDebits
		if parsed.ClosingBalance != 0 && absFloat(expectedClosing-parsed.ClosingBalance) > bankStatementValidationTolerance {
			validation.Blocking = true
			validation.BlockingIssues = append(validation.BlockingIssues, "opening balance, totals, and closing balance do not reconcile")
		}
	}
	if headerDebitTotal > 0 && absFloat(headerDebitTotal-computedDebits) > bankStatementValidationTolerance {
		validation.Blocking = true
		validation.BlockingIssues = append(validation.BlockingIssues, "parsed debits do not match the statement debit summary")
	}
	if headerCreditTotal > 0 && absFloat(headerCreditTotal-computedCredits) > bankStatementValidationTolerance {
		validation.Blocking = true
		validation.BlockingIssues = append(validation.BlockingIssues, "parsed credits do not match the statement credit summary")
	}
	if headerDebitCount > 0 && headerDebitCount != computedDebitCount {
		validation.Blocking = true
		validation.BlockingIssues = append(validation.BlockingIssues, "parsed debit count does not match the statement debit summary")
	}
	if headerCreditCount > 0 && headerCreditCount != computedCreditCount {
		validation.Blocking = true
		validation.BlockingIssues = append(validation.BlockingIssues, "parsed credit count does not match the statement credit summary")
	}

	if parsed.TotalDebits == 0 && computedDebits > 0 {
		validation.Warnings = append(validation.Warnings, "statement totals were reconstructed from line items")
		parsed.TotalDebits = computedDebits
	}
	if parsed.TotalCredits == 0 && computedCredits > 0 {
		validation.Warnings = append(validation.Warnings, "statement totals were reconstructed from line items")
		parsed.TotalCredits = computedCredits
	}
	parsed.DebitCount = 0
	parsed.CreditCount = 0
	for _, line := range parsed.Lines {
		if line.Debit > 0 {
			parsed.DebitCount++
		}
		if line.Credit > 0 {
			parsed.CreditCount++
		}
	}

	return validation
}

func shouldEscalateBankStatementToAIML(source string, validation bankStatementValidation, lineCount int) bool {
	if !strings.Contains(strings.ToUpper(source), "GENERIC") {
		return false
	}
	if validation.Blocking {
		return true
	}
	if validation.BalanceMismatchCount > 0 {
		return true
	}
	return validation.AmbiguousAmountCount > maxInt(1, lineCount/6)
}

func isBankStatementAIMLAssistEnabled() bool {
	return getEnvBool(bankStatementAIMLFlagEnv, false)
}

func parseBankStatementWithAIML(text string) (*parsedStatement, string, error) {
	apiKey := strings.TrimSpace(getAIMLAPIKey())
	if apiKey == "" {
		return nil, "", fmt.Errorf("AIML API key is not configured")
	}

	cleanedText := strings.TrimSpace(text)
	if cleanedText == "" {
		return nil, "", fmt.Errorf("statement text is empty")
	}
	maxChars := getEnvInt(bankStatementAIMLMaxTextCharsEnv, defaultBankStatementAITextLimit)
	if maxChars > 0 && len(cleanedText) > maxChars {
		cleanedText = cleanedText[:maxChars]
	}

	systemPrompt := `You extract bank statement rows into strict JSON for ERP reconciliation.
Return only JSON.
Keep separate charges, VAT rows, and counterparty rows as separate transactions if the text suggests they are separate.
Correct debit vs credit polarity using running balance and banking semantics.
Do not invent rows that are not present in the text.`

	userPrompt := fmt.Sprintf(`Parse this OCR bank statement into JSON with this schema:
{
  "account_number": string,
  "iban": string,
  "currency": string,
  "period_start": "YYYY-MM-DD",
  "period_end": "YYYY-MM-DD",
  "opening_balance": number,
  "closing_balance": number,
  "total_debits": number,
  "total_credits": number,
  "lines": [
    {
      "line_number": number,
      "date": "YYYY-MM-DD",
      "value_date": "YYYY-MM-DD",
      "reference": string,
      "description": string,
      "debit": number,
      "credit": number,
      "balance": number
    }
  ]
}

OCR text:
%s`, cleanedText)

	response, model, err := callAIMLForBankStatement(apiKey, systemPrompt, userPrompt)
	if err != nil {
		return nil, "", err
	}

	jsonPayload := extractJSONObject(response)
	if jsonPayload == "" {
		return nil, model, fmt.Errorf("AI response did not contain JSON")
	}

	var payload bankStatementAIMLResponse
	if err := json.Unmarshal([]byte(jsonPayload), &payload); err != nil {
		return nil, model, fmt.Errorf("failed to parse AI JSON response: %w", err)
	}

	parsed, err := normalizeAIMLBankStatement(payload)
	if err != nil {
		return nil, model, err
	}
	return parsed, model, nil
}

func callAIMLForBankStatement(apiKey, systemPrompt, userPrompt string) (string, string, error) {
	type message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	type request struct {
		Model       string    `json:"model"`
		Messages    []message `json:"messages"`
		MaxTokens   int       `json:"max_tokens"`
		Temperature float64   `json:"temperature"`
	}

	timeout := time.Duration(getEnvInt(bankStatementAIMLTimeoutEnv, defaultBankStatementAITimeoutMS)) * time.Millisecond
	if timeout <= 0 {
		timeout = defaultBankStatementAITimeoutMS * time.Millisecond
	}

	models := dedupeStringSlice(append([]string{getAIMLModelID()}, fallbackAIMLModels...))
	var lastErr error
	for _, model := range models {
		body, err := json.Marshal(request{
			Model: model,
			Messages: []message{
				{Role: "system", Content: systemPrompt},
				{Role: "user", Content: userPrompt},
			},
			MaxTokens:   bankStatementAIMLMaxTokens,
			Temperature: 0.1,
		})
		if err != nil {
			return "", "", fmt.Errorf("marshal error: %w", err)
		}

		req, err := http.NewRequest("POST", aimlAPIURL, bytes.NewBuffer(body))
		if err != nil {
			return "", "", fmt.Errorf("request creation error: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)

		client := &http.Client{Timeout: timeout}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			log.Printf("⚠️ Bank statement AIML attempt failed (%s): %v", model, err)
			continue
		}

		respBody, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			log.Printf("⚠️ Bank statement AIML response read failed (%s): %v", model, readErr)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("AIML API error (status %d): %s", resp.StatusCode, string(respBody))
			log.Printf("⚠️ Bank statement AIML attempt failed (%s): %v", model, lastErr)
			continue
		}

		var parsed struct {
			Choices []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			} `json:"choices"`
		}
		if err := json.Unmarshal(respBody, &parsed); err != nil {
			lastErr = err
			log.Printf("⚠️ Bank statement AIML JSON decode failed (%s): %v", model, err)
			continue
		}
		if len(parsed.Choices) == 0 {
			lastErr = fmt.Errorf("no response choices from AIML API")
			log.Printf("⚠️ Bank statement AIML returned no choices (%s)", model)
			continue
		}

		return parsed.Choices[0].Message.Content, model, nil
	}

	return "", "", lastErr
}

func normalizeAIMLBankStatement(payload bankStatementAIMLResponse) (*parsedStatement, error) {
	if len(payload.Lines) == 0 {
		return nil, fmt.Errorf("AI response had no transaction lines")
	}

	parsed := &parsedStatement{
		AccountNumber:  strings.TrimSpace(payload.AccountNumber),
		IBAN:           strings.TrimSpace(payload.IBAN),
		Currency:       strings.TrimSpace(payload.Currency),
		OpeningBalance: payload.OpeningBalance,
		ClosingBalance: payload.ClosingBalance,
		TotalDebits:    payload.TotalDebits,
		TotalCredits:   payload.TotalCredits,
	}
	if parsed.Currency == "" {
		parsed.Currency = "BHD"
	}
	if parsed.PeriodStart = parseFlexibleDate(payload.PeriodStart); parsed.PeriodStart.IsZero() {
		parsed.PeriodStart = time.Time{}
	}
	if parsed.PeriodEnd = parseFlexibleDate(payload.PeriodEnd); parsed.PeriodEnd.IsZero() {
		parsed.PeriodEnd = time.Time{}
	}

	for idx, line := range payload.Lines {
		date := parseFlexibleDate(line.Date)
		valueDate := parseFlexibleDate(line.ValueDate)
		if valueDate.IsZero() {
			valueDate = date
		}
		parsed.Lines = append(parsed.Lines, parsedLine{
			LineNumber:  maxInt(1, chooseLineNumber(line.LineNumber, idx+1)),
			Date:        date,
			ValueDate:   valueDate,
			Reference:   strings.TrimSpace(line.Reference),
			Description: strings.TrimSpace(line.Description),
			Debit:       clampNonNegative(line.Debit),
			Credit:      clampNonNegative(line.Credit),
			Balance:     line.Balance,
		})
	}

	sort.SliceStable(parsed.Lines, func(i, j int) bool {
		if parsed.Lines[i].LineNumber == parsed.Lines[j].LineNumber {
			return parsed.Lines[i].Date.Before(parsed.Lines[j].Date)
		}
		return parsed.Lines[i].LineNumber < parsed.Lines[j].LineNumber
	})

	if parsed.PeriodStart.IsZero() && len(parsed.Lines) > 0 {
		parsed.PeriodStart = parsed.Lines[0].Date
	}
	if parsed.PeriodEnd.IsZero() && len(parsed.Lines) > 0 {
		parsed.PeriodEnd = parsed.Lines[len(parsed.Lines)-1].Date
	}
	if parsed.ClosingBalance == 0 && len(parsed.Lines) > 0 {
		parsed.ClosingBalance = parsed.Lines[len(parsed.Lines)-1].Balance
	}
	if parsed.OpeningBalance == 0 && len(parsed.Lines) > 0 {
		first := parsed.Lines[0]
		parsed.OpeningBalance = first.Balance + first.Debit - first.Credit
	}

	return parsed, nil
}

func extractJSONObject(raw string) string {
	content := strings.TrimSpace(raw)
	if strings.HasPrefix(content, "```") {
		content = strings.TrimSpace(strings.TrimPrefix(content, "```json"))
		content = strings.TrimSpace(strings.TrimPrefix(content, "```"))
		content = strings.TrimSpace(strings.TrimSuffix(content, "```"))
	}
	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")
	if start == -1 || end == -1 || end <= start {
		return ""
	}
	return content[start : end+1]
}

func absFloat(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func clampNonNegative(value float64) float64 {
	if value < 0 {
		return 0
	}
	return value
}

func chooseLineNumber(lineNumber int, fallback int) int {
	if lineNumber > 0 {
		return lineNumber
	}
	return fallback
}
