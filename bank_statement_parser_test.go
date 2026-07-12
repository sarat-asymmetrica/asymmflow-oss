package main

import (
	"math"
	"testing"
)

func TestParseNBBFormatInfersPolarityFromRunningBalance(t *testing.T) {
	statementText := `National Bank of Bahrain BSC
Account Number 200000412340002
Statement 01/01/2026 To 31/01/2026
Opening Balance 70,058.251
02/01/2026 Fawri Ordinary Transfer NATIONAL PETROLEUM CO. 999990378 22,720.500 92,778.751
03/01/2026 DGC NBB TRF FROM 0012340001 TO 0099640704 TRF 999990371 3,658.600 89,120.151
04/01/2026 INWD PO1 8583790000 6001 Cheque 42 999990373 12,000.000 101,120.151
Closing Balance 101,120.151`

	parsed, err := parseNBBFormat(statementText)
	if err != nil {
		t.Fatalf("parseNBBFormat returned error: %v", err)
	}

	if got, want := len(parsed.Lines), 3; got != want {
		t.Fatalf("expected %d lines, got %d", want, got)
	}

	assertAmount := func(label string, got, want float64) {
		t.Helper()
		if math.Abs(got-want) > 0.001 {
			t.Fatalf("%s = %.3f, want %.3f", label, got, want)
		}
	}

	assertAmount("opening balance", parsed.OpeningBalance, 70058.251)
	assertAmount("closing balance", parsed.ClosingBalance, 101120.151)

	assertAmount("line1 credit", parsed.Lines[0].Credit, 22720.500)
	assertAmount("line1 debit", parsed.Lines[0].Debit, 0)
	assertAmount("line2 debit", parsed.Lines[1].Debit, 3658.600)
	assertAmount("line2 credit", parsed.Lines[1].Credit, 0)
	assertAmount("line3 credit", parsed.Lines[2].Credit, 12000.000)
	assertAmount("line3 debit", parsed.Lines[2].Debit, 0)
}

func TestParseNBBFormatKeepsContinuationLinesWithCreditWord(t *testing.T) {
	statementText := `National Bank of Bahrain BSC
Account Number 200000412340002
Statement 01/01/2026 To 31/01/2026
Opening Balance 100.000
02/01/2026 SWIFT CHARGES FOR
Credit Advice Ref 999990001 15.000 85.000
Closing Balance 85.000`

	parsed, err := parseNBBFormat(statementText)
	if err != nil {
		t.Fatalf("parseNBBFormat returned error: %v", err)
	}

	if got, want := len(parsed.Lines), 1; got != want {
		t.Fatalf("expected %d line, got %d", want, got)
	}

	if desc := parsed.Lines[0].Description; desc != "SWIFT CHARGES FOR Credit Advice Ref" {
		t.Fatalf("unexpected description: %q", desc)
	}
	if math.Abs(parsed.Lines[0].Debit-15.000) > 0.001 {
		t.Fatalf("expected debit 15.000, got %.3f", parsed.Lines[0].Debit)
	}
	if parsed.Lines[0].Credit != 0 {
		t.Fatalf("expected credit 0, got %.3f", parsed.Lines[0].Credit)
	}
}

func TestParseNBBFormatKeepsContinuationLinesWithBalanceWord(t *testing.T) {
	statementText := `National Bank of Bahrain BSC
Account Number 200000412340002
Statement 01/01/2026 To 31/01/2026
Opening Balance 100.000
02/01/2026 Receipt for
Balance Transfer Ref 999990002 25.000 125.000
Closing Balance 125.000`

	parsed, err := parseNBBFormat(statementText)
	if err != nil {
		t.Fatalf("parseNBBFormat returned error: %v", err)
	}

	if got, want := len(parsed.Lines), 1; got != want {
		t.Fatalf("expected %d line, got %d", want, got)
	}

	if desc := parsed.Lines[0].Description; desc != "Receipt for Balance Transfer Ref" {
		t.Fatalf("unexpected description: %q", desc)
	}
	if math.Abs(parsed.Lines[0].Credit-25.000) > 0.001 {
		t.Fatalf("expected credit 25.000, got %.3f", parsed.Lines[0].Credit)
	}
	if parsed.Lines[0].Debit != 0 {
		t.Fatalf("expected debit 0, got %.3f", parsed.Lines[0].Debit)
	}
}

func TestParseNBBFormatMatchesRealJanuary2026Statement(t *testing.T) {
	statementText := `National Bank of Bahrain BSC
Building:120, Road/Street:383, Town: Manama, Block:316, Country: Bahrain
VAT account Number: 200000412340002

ACCOUNT STATEMENT - TAX INVOICE
M/S.ACME INSTRUMENTATION WLL Account Number 0000000012340001
CURRENT ACCOUNT-BHD
Bank BIC DMOABHBM
IBAN BH16DMOA00000012340001
Date Printed 01/02/2026
Currency BHD
Statement 01/01/2026 To 31/01/2026
Opening Balance 70,558.251
Closing Balance 87,980.027
Total Debits 42 For 43,569.974
Total Credits 9 For 60,991.750
Current Balance 83,680.027
Available Balance 83,680.027
Amount Type All

Date Reference Number Description Currency Debit Amount Credit Amount Balance
Opening Balance 70,558.251
01/01/2026 999990378 Fawri Ordinary Transfer NATIONAL PETROLEUM CO. CLOSED 126199 FTRF BHD 22,720.500 93,278.751
04/01/2026 999990373 INWD P01 85837900006001 Cheque 842 FCHK00000842 BHD 12,000.000 81,278.751
05/01/2026 999990371 DGC NBB TRF FROM 0012340001 TO 0099640740 FTRF BHD 3,658.600 77,620.151
08/01/2026 999990367 DGC-Fawri to BH90BBME00001490465001,Ref:DMA080126ODK9POL FTRF BHD 1,197.998 76,422.153
08/01/2026 999990360 EFTS Charges DMA080126ODK9POL FTRF BHD 0.100 76,422.053
08/01/2026 999990358 VAT 10.00% On Fee 0.100 for Ref:DMA080126ODK9POL FTRF BHD 0.010 76,422.043
11/01/2026 999990356 ONUS P01B82 33886330001001 Cheque 839 FCHK00000839 BHD 900.000 75,522.043
12/01/2026 999990351 INWD P01 85925600006001 Cheque 845 FCHK00000845 BHD 3,000.000 72,522.043
13/01/2026 999990349 Fawri Ordinary Transfer MEADOW DAIRY WLL 510001395926 FTRF BHD 20,359.900 92,881.943
13/01/2026 999990344 NBB CHQ NO.846 FCHK00000846 BHD 3,000.000 89,881.943
14/01/2026 999990339 Fawri Ordinary transfers RIVERSIDE POWER OPERATION AND MAINTENANCE FTRF BHD 6,893.700 96,775.643
14/01/2026 999990334 DGC CROSS CCY JAMIE WONG FX RATE: 0.37785 FTRF BHD 1,133.550 95,642.093
14/01/2026 999990326 SWIFT FEES DMA140126BSLITVK FTRF BHD 5.000 95,637.093
14/01/2026 999990324 VAT 10.00% On Fee 5.000 for Ref:DMA140126BSLITVK FTRF BHD 0.500 95,636.593
14/01/2026 999990322 Corr Bnk Chg-DMA140126BSLITVK FTRF BHD 9.068 95,627.525
15/01/2026 999990315 NBL-Fawateer toNBRBBHBM.205000543241 DMA150126BNLXMND FTRF BHD 10,944.330 84,683.195
15/01/2026 999990310 AmendBG 211330LTG FTRF BHD 25.000 84,658.195
Date and Time: 01/02/2026 9:34 AM Page 1 of 2
15/01/2026 999990308 10.00%VAT 2.50 BHD FTRF BHD 2.500 84,655.695
18/01/2026 999990306 Fawri Ordinary transfers COASTAL JV WLL /INV/INV-2025-0091 INV-2025-0092 FTRF BHD 1,887.050 86,542.745
20/01/2026 999990301 AmendBG 0000124BGG650707 FTRF BHD 25.000 86,517.745
20/01/2026 999990299 10.00%VAT 2.50 BHD FTRF BHD 2.500 86,515.245
21/01/2026 999990297 DGC-Fawri to BH95DMOB00016470214001,Ref:DMA210126ODNF7NW FTRF BHD 448.800 86,066.445
21/01/2026 999990290 EFTS Charges DMA210126ODNF7NW FTRF BHD 0.100 86,066.345
21/01/2026 999990288 VAT 10.00% On Fee 0.100 for Ref:DMA210126ODNF7NW FTRF BHD 0.010 86,066.335
22/01/2026 999990286 Fawri Ordinary Transfer MEADOW DAIRY WLL 510002195426 FTRF BHD 660.000 86,726.335
22/01/2026 999990281 AmendBG 0000124BGG651207 FTRF BHD 25.000 86,701.335
22/01/2026 999990279 10.00%VAT 2.50 BHD FTRF BHD 2.500 86,698.835
24/01/2026 999990277 DGC-Fawri to BH49DMOC00204922100100,Ref:DMA240126ODO3YBO FTRF BHD 1,009.710 85,689.125
24/01/2026 999990270 EFTS Charges DMA240126ODO3YBO FTRF BHD 0.100 85,689.025
24/01/2026 999990268 VAT 10.00% On Fee 0.100 for Ref:DMA240126ODO3YBO FTRF BHD 0.010 85,689.015
25/01/2026 999990266 NBB CHQ NO.847 FCHK00000847 BHD 3,000.000 82,689.015
27/01/2026 999990254 03INTL001329 INV-2024-0159-AQUAPURE WATER TECHNOLOGIES SO FTRF BHD 3,068.900 85,757.915
27/01/2026 999990249 Fawri Ordinary Transfer AQUAPURE WATER TECHNOLOGIES + SOLUTIO FTRF BHD 180.000 85,937.915
27/01/2026 999990244 Fawri Ordinary Transfer HARBOR DAIRY COMPANY W.L.L. PAYMENT FTRF BHD 2,470.600 88,408.515
28/01/2026 999990239 Fawri Ordinary transfers COASTAL JV WLL /INV/INV-2025-0104 FTRF BHD 2,751.100 91,159.615
28/01/2026 999990234 AmendBG 211367LTG FTRF BHD 123.516 91,036.099
28/01/2026 999990232 10.00%VAT 12.35 BHD FTRF BHD 12.352 91,023.747
28/01/2026 999990230 AmendBG 211548LTG FTRF BHD 25.000 90,998.747
28/01/2026 999990228 10.00%VAT 2.50 BHD FTRF BHD 2.500 90,996.247
28/01/2026 999990226 AmendBG 221586LTG FTRF BHD 25.000 90,971.247
28/01/2026 999990224 10.00%VAT 2.50 BHD FTRF BHD 2.500 90,968.747
29/01/2026 999990222 ONUS P01B89 34138930001001 Cheque 902 FCHK00000902 BHD 900.000 90,068.747
29/01/2026 999990217 ONUS P01B89 34139130001001 Cheque 849 FCHK00000849 BHD 1,300.000 88,768.747
29/01/2026 999990212 NBL-Fawri+ to BH28ABCO55056239101001,Ref:DMA290126ONPHDGU FTRF BHD 350.000 88,418.747
29/01/2026 999990207 EFTS Charges DMA290126ONPHDGU FTRF BHD 0.100 88,418.647
29/01/2026 999990205 VAT 10.00% On Fee 0.100 for Ref:DMA290126ONPHDGU FTRF BHD 0.010 88,418.637
29/01/2026 999990203 NBL-Fawri+ to BH30ABCO53824079101001,Ref:DMA290126ONPHDM6 FTRF BHD 400.000 88,018.637
29/01/2026 999990198 EFTS Charges DMA290126ONPHDM6 FTRF BHD 0.100 88,018.537
29/01/2026 999990196 VAT 10.00% On Fee 0.100 for Ref:DMA290126ONPHDM6 FTRF BHD 0.010 88,018.527
31/01/2026 999990194 AmendBG 0000125BGG650403 FTRF BHD 35.000 87,983.527
31/01/2026 999990192 10.00%VAT 3.50 BHD FTRF BHD 3.500 87,980.027
Closing Balance 87,980.027`

	parsed, err := parseNBBFormat(statementText)
	if err != nil {
		t.Fatalf("parseNBBFormat returned error: %v", err)
	}

	if got, want := parsed.AccountNumber, "0000000012340001"; got != want {
		t.Fatalf("account number = %q, want %q", got, want)
	}
	if got, want := parsed.IBAN, "BH16DMOA00000012340001"; got != want {
		t.Fatalf("IBAN = %q, want %q", got, want)
	}
	if got, want := len(parsed.Lines), 51; got != want {
		t.Fatalf("expected %d lines, got %d", want, got)
	}

	assertAmount := func(label string, got, want float64) {
		t.Helper()
		if math.Abs(got-want) > 0.001 {
			t.Fatalf("%s = %.3f, want %.3f", label, got, want)
		}
	}

	assertAmount("opening balance", parsed.OpeningBalance, 70558.251)
	assertAmount("closing balance", parsed.ClosingBalance, 87980.027)
	assertAmount("total debits", parsed.TotalDebits, 43569.974)
	assertAmount("total credits", parsed.TotalCredits, 60991.750)
	if got, want := parsed.DebitCount, 42; got != want {
		t.Fatalf("debit count = %d, want %d", got, want)
	}
	if got, want := parsed.CreditCount, 9; got != want {
		t.Fatalf("credit count = %d, want %d", got, want)
	}

	if got := parsed.Lines[0].Description; got != "Fawri Ordinary Transfer NATIONAL PETROLEUM CO. CLOSED 126199 FTRF" {
		t.Fatalf("unexpected first description: %q", got)
	}
	assertAmount("first line credit", parsed.Lines[0].Credit, 22720.500)
	assertAmount("last line debit", parsed.Lines[len(parsed.Lines)-1].Debit, 3.500)
	assertAmount("last line balance", parsed.Lines[len(parsed.Lines)-1].Balance, 87980.027)
}

func TestParseNBBFormatMatchesEuroCallStatement(t *testing.T) {
	statementText := `National Bank of Bahrain BSC
ACCOUNT STATEMENT - TAX INVOICE
M/S.ACME INSTRUMENTATION WLL Account Number 0000000012340002
CALL ACCOUNT -EUR RETAIL
Bank BIC DMOABHBM
IBAN BH93DMOA00000012340002
Date Printed 01/02/2026
Currency EUR
Statement 01/01/2026 To 31/01/2026
Opening Balance 71,351.58
Closing Balance 66,299.69
Total Debits 9 For 5,051.89
Total Credits 0 For 0.00
Date Reference Number Description Currency Debit Amount Credit Amount Balance
Opening Balance 71,351.58
13/01/2026 999998595 DGC Remittance To MERIDIAN FTRF EUR 4,950.00 66,401.58
13/01/2026 999998590 SWIFT FEES DMA130126BSL9CZV FTRF EUR 11.84 66,389.74
13/01/2026 999998588 VAT 10.00% On Fee 11.840 for Ref:DMA130126BSL9CZV FTRF EUR 1.18 66,388.56
14/01/2026 999998586 SWIFT CHARGES DMA181225BSF8QC5 DTD 181225 FTRF EUR 10.00 66,378.56
21/01/2026 999998579 Balance Confirmation FTRF EUR 47.15 66,331.41
21/01/2026 999998578 10.00%VAT 4.72 EUR FTRF EUR 4.72 66,326.69
29/01/2026 999998576 SWIFT CHARGES DMA130126BSL9CZV DTD 130126 FTRF EUR 7.00 66,319.69
29/01/2026 999998574 SWIFT CHARGES DMA130126BSL9CZV DTD 140126 FTRF EUR 10.00 66,309.69
29/01/2026 999998572 SWIFT CHARGES DMA311225BSIGI30 DTD 311225 FTRF EUR 10.00 66,299.69
Closing Balance 66,299.69`

	parsed, err := parseNBBFormat(statementText)
	if err != nil {
		t.Fatalf("parseNBBFormat returned error: %v", err)
	}

	if parsed.Currency != "EUR" {
		t.Fatalf("currency = %q, want EUR", parsed.Currency)
	}
	if parsed.AccountNumber != "0000000012340002" {
		t.Fatalf("account number = %q", parsed.AccountNumber)
	}
	if parsed.IBAN != "BH93DMOA00000012340002" {
		t.Fatalf("IBAN = %q", parsed.IBAN)
	}
	if got, want := len(parsed.Lines), 9; got != want {
		t.Fatalf("lines = %d, want %d", got, want)
	}
	if math.Abs(parsed.TotalDebits-5051.89) > 0.001 {
		t.Fatalf("total debits = %.3f, want 5051.890", parsed.TotalDebits)
	}
	if parsed.TotalCredits != 0 {
		t.Fatalf("total credits = %.3f, want 0", parsed.TotalCredits)
	}
	if parsed.DebitCount != 9 || parsed.CreditCount != 0 {
		t.Fatalf("unexpected debit/credit counts: %d/%d", parsed.DebitCount, parsed.CreditCount)
	}
	if math.Abs(parsed.Lines[0].Debit-4950.0) > 0.001 {
		t.Fatalf("first debit = %.3f, want 4950.000", parsed.Lines[0].Debit)
	}
	if parsed.Lines[0].Credit != 0 {
		t.Fatalf("first credit = %.3f, want 0", parsed.Lines[0].Credit)
	}
	if parsed.Lines[0].Description != "DGC Remittance To MERIDIAN FTRF" {
		t.Fatalf("unexpected first description: %q", parsed.Lines[0].Description)
	}
}

func TestParseNBBFormatMatchesEuroCallColumnarOCRLayout(t *testing.T) {
	statementText := `National Bank of Bahrain BSC
ACCOUNT STATEMENT - TAX INVOICE
M/S.ACME INSTRUMENTATION WLL
BH93DMOA00000012340002
EUR
01/01/2026 To 31/01/2026
0000000012340002
01/02/2026
DMOABHBM
66,299.69
66,299.69
Account Number
Date Printed
Statement
Currency
Opening Balance
IBAN
Bank BIC
Closing Balance
Current Balance
Available Balance
Total Debits
Total Credits
9
0
All
5,051.89
0.00
Amount Type
71,351.58
Account Type
CALL A/C
66,299.69
CALL ACCOUNT -EUR RETAIL
For
For
Balance
Date
Description
Currency
Debit Amount
Credit Amount
Reference Number
71,351.58
Opening Balance
13/01/2026
999998595
DGC Remittance To MERIDIAN FTRF
66,401.58
EUR
4,950.00
13/01/2026
999998590
SWIFT FEES DMA130126BSL9CZV FTRF
66,389.74
EUR
11.84
13/01/2026
999998588
VAT 10.00% On Fee 11.840 for Ref:DMA130126BSL9CZV FTRF
66,388.56
EUR
1.18
14/01/2026
999998586
SWIFT CHARGES DMA181225BSF8QC5 DTD 181225 FTRF
66,378.56
EUR
10.00
21/01/2026
999998579
Balance Confirmation FTRF
66,331.41
EUR
47.15
21/01/2026
999998578
10.00%VAT 4.72 EUR FTRF
66,326.69
EUR
4.72
29/01/2026
999998576
SWIFT CHARGES DMA130126BSL9CZV DTD 130126 FTRF
66,319.69
EUR
7.00
29/01/2026
999998574
SWIFT CHARGES DMA130126BSL9CZV DTD 140126 FTRF
66,309.69
EUR
10.00
29/01/2026
999998572
SWIFT CHARGES DMA311225BSIGI30 DTD 311225 FTRF
66,299.69
EUR
10.00
Date and Time:
01/02/2026 9:37 AM
Page 1 of 2`

	parsed, err := parseNBBFormat(statementText)
	if err != nil {
		t.Fatalf("parseNBBFormat returned error: %v", err)
	}
	if got, want := len(parsed.Lines), 9; got != want {
		t.Fatalf("lines = %d, want %d", got, want)
	}
	if math.Abs(parsed.OpeningBalance-71351.58) > 0.001 {
		t.Fatalf("opening balance = %.2f", parsed.OpeningBalance)
	}
	if math.Abs(parsed.ClosingBalance-66299.69) > 0.001 {
		t.Fatalf("closing balance = %.2f", parsed.ClosingBalance)
	}
	if math.Abs(parsed.TotalDebits-5051.89) > 0.001 {
		t.Fatalf("total debits = %.2f", parsed.TotalDebits)
	}
	if parsed.TotalCredits != 0 {
		t.Fatalf("total credits = %.2f, want 0", parsed.TotalCredits)
	}
	if parsed.DebitCount != 9 || parsed.CreditCount != 0 {
		t.Fatalf("unexpected debit/credit counts: %d/%d", parsed.DebitCount, parsed.CreditCount)
	}
	if math.Abs(parsed.Lines[0].Debit-4950.0) > 0.001 {
		t.Fatalf("first debit = %.2f", parsed.Lines[0].Debit)
	}
	if parsed.Lines[0].Reference != "999998595" {
		t.Fatalf("first ref = %q", parsed.Lines[0].Reference)
	}
}

func TestParseNBBColumnarLayoutContinuesAcrossPageMarker(t *testing.T) {
	lines := []string{
		"Reference Number",
		"01/01/2026",
		"999990001",
		"First page columnar credit",
		"1,100.000",
		"BHD",
		"100.000",
		"02/01/2026",
		"999990002",
		"First page columnar debit",
		"1,050.000",
		"BHD",
		"50.000",
		"Date and Time: 01/02/2026 9:34 AM",
		"Page 1 of 2",
		"--- PAGE 2 ---",
		"03/01/2026",
		"999990003",
		"Second page columnar debit",
		"1,025.000",
		"BHD",
		"25.000",
		"04/01/2026",
		"999990004",
		"Second page columnar credit",
		"1,225.000",
		"BHD",
		"200.000",
	}

	base := &parsedStatement{
		Currency:       "BHD",
		OpeningBalance: 1000.000,
	}

	parsed := parseNBBColumnarLayout(lines, base)
	if parsed == nil {
		t.Fatalf("parseNBBColumnarLayout returned nil")
	}
	if got, want := len(parsed.Lines), 4; got != want {
		t.Fatalf("lines = %d, want %d", got, want)
	}
	if parsed.Lines[2].Reference != "999990003" {
		t.Fatalf("third line ref = %q", parsed.Lines[2].Reference)
	}
	if parsed.Lines[3].Reference != "999990004" {
		t.Fatalf("fourth line ref = %q", parsed.Lines[3].Reference)
	}
	if math.Abs(parsed.Lines[3].Balance-1225.000) > 0.001 {
		t.Fatalf("fourth line balance = %.3f, want 1225.000", parsed.Lines[3].Balance)
	}
}

func TestParseNBBFormatRejectsKFHStatement(t *testing.T) {
	statementText := `Account History
ACME INSTRUMENTATION SPC
ACME INSTRUMENTATION W.L.L
Account Number: 0010912340001
IBAN: BH04DMOB00010912340001
Currency: BHD
Opening Balance: 25,709.210
03/01/2026 03/01/2026 SUNDRY CREDIT(ADV) | IGT/DG/2025/1451 | CC 500.000 26,209.210`

	if _, err := parseNBBFormat(statementText); err == nil {
		t.Fatalf("expected KFH statement text to be rejected by parseNBBFormat")
	}
}

func TestParseKFHFormatMatchesJanuary2026Statement(t *testing.T) {
	statementText := `Account History
ACME INSTRUMENTATION SPC
ACME INSTRUMENTATION W.L.L
Account Number: 0010912340001 IBAN: BH04DMOB00010912340001
Currency: BHD
From - To: 01/01/2026 - 31/01/2026
Opening Balance: 25,709.210
Closing Balance: 56,753.716
Date Value Date Description Amount Balance
03/01/2026 03/01/2026 SUNDRY CREDIT(ADV) | IGT/DG/2025/1451 | CC 500.000 26,209.210
15/01/2026 15/01/2026 FAWRI | IPS102K2N0039VCU | NORTH GRID AUTHORITY | TBS2601142849344 | PAYMENT | FROM BH14DMOD00100000249790 31,072.006 57,281.216
21/01/2026 21/01/2026 DEBIT(TF) | IGT/DG/2026/0146 | CCM 527.500 56,753.716`

	parsed, err := parseKFHFormat(statementText)
	if err != nil {
		t.Fatalf("parseKFHFormat returned error: %v", err)
	}

	if parsed.AccountNumber != "0010912340001" {
		t.Fatalf("account number = %q", parsed.AccountNumber)
	}
	if parsed.IBAN != "BH04DMOB00010912340001" {
		t.Fatalf("IBAN = %q", parsed.IBAN)
	}
	if parsed.Currency != "BHD" {
		t.Fatalf("currency = %q", parsed.Currency)
	}
	if len(parsed.Lines) != 3 {
		t.Fatalf("lines = %d, want 3", len(parsed.Lines))
	}
	if math.Abs(parsed.TotalCredits-31572.006) > 0.001 {
		t.Fatalf("total credits = %.3f, want 31572.006", parsed.TotalCredits)
	}
	if math.Abs(parsed.TotalDebits-527.5) > 0.001 {
		t.Fatalf("total debits = %.3f, want 527.500", parsed.TotalDebits)
	}
	if parsed.CreditCount != 2 || parsed.DebitCount != 1 {
		t.Fatalf("unexpected debit/credit counts: %d/%d", parsed.DebitCount, parsed.CreditCount)
	}
	if math.Abs(parsed.Lines[1].Credit-31072.006) > 0.001 {
		t.Fatalf("second line credit = %.3f, want 31072.006", parsed.Lines[1].Credit)
	}
	if math.Abs(parsed.Lines[2].Debit-527.5) > 0.001 {
		t.Fatalf("third line debit = %.3f, want 527.500", parsed.Lines[2].Debit)
	}
}

func TestParseKFHFormatMatchesColumnarPDFExtractionLayout(t *testing.T) {
	statementText := `Account History
ACME INSTRUMENTATION SPC
ACME INSTRUMENTATION W.L.L

Account Number:
0010912340001
Currency:
BHD
From - To:
01/01/2026 31/01/2026

Opening Balance:
25,709.210

03/01/2026
03/01/2026
SUNDRY CREDIT(ADV) | IGT/DG/2025/1451 | CC

15/01/2026
15/01/2026
FAWRI | IPS102K2N0039VCU | NORTH GRID
AUTHORITY | TBS2601142849344 | PAYMENT | FROM
BH14DMOD00100000249790

21/01/2026
21/01/2026
DEBIT(TF) | IGT/DG/2026/0146 | CCM

Amount
Balance
500.000
26,209.210
31,072.006
57,281.216
527.500
56,753.716

Closing Balance:
56,753.716
Generated On: 01/02/2026 09:50 AM
Page 1 of 1`

	parsed, err := parseKFHFormat(statementText)
	if err != nil {
		t.Fatalf("parseKFHFormat returned error: %v", err)
	}

	if got, want := len(parsed.Lines), 3; got != want {
		t.Fatalf("lines = %d, want %d", got, want)
	}
	if math.Abs(parsed.OpeningBalance-25709.210) > 0.001 {
		t.Fatalf("opening balance = %.3f, want 25709.210", parsed.OpeningBalance)
	}
	if math.Abs(parsed.ClosingBalance-56753.716) > 0.001 {
		t.Fatalf("closing balance = %.3f, want 56753.716", parsed.ClosingBalance)
	}
	if math.Abs(parsed.Lines[0].Credit-500.000) > 0.001 {
		t.Fatalf("first line credit = %.3f, want 500.000", parsed.Lines[0].Credit)
	}
	if math.Abs(parsed.Lines[1].Credit-31072.006) > 0.001 {
		t.Fatalf("second line credit = %.3f, want 31072.006", parsed.Lines[1].Credit)
	}
	if math.Abs(parsed.Lines[2].Debit-527.500) > 0.001 {
		t.Fatalf("third line debit = %.3f, want 527.500", parsed.Lines[2].Debit)
	}
	if got, want := parsed.Lines[1].Reference, "IPS102K2N0039VCU"; got != want {
		t.Fatalf("second line reference = %q, want %q", got, want)
	}
}

func TestRepairParsedStatementPolarityRepairsSystematicInversionFromStatementTotals(t *testing.T) {
	parsed := &parsedStatement{
		Currency:       "BHD",
		ClosingBalance: 56753.716,
		TotalDebits:    527.500,
		TotalCredits:   31572.006,
		DebitCount:     1,
		CreditCount:    2,
		Lines: []parsedLine{
			{
				LineNumber:  1,
				Date:        parseFlexibleDate("03/01/2026"),
				ValueDate:   parseFlexibleDate("03/01/2026"),
				Description: "TXN 1451",
				Debit:       500.000,
				Balance:     26209.210,
			},
			{
				LineNumber:  2,
				Date:        parseFlexibleDate("15/01/2026"),
				ValueDate:   parseFlexibleDate("15/01/2026"),
				Description: "TXN 2849344",
				Debit:       31072.006,
				Balance:     57281.216,
			},
			{
				LineNumber:  3,
				Date:        parseFlexibleDate("21/01/2026"),
				ValueDate:   parseFlexibleDate("21/01/2026"),
				Description: "TXN 0146",
				Credit:      527.500,
				Balance:     56753.716,
			},
		},
	}

	repairParsedStatementPolarity(parsed)

	if math.Abs(parsed.Lines[0].Credit-500.000) > 0.001 || parsed.Lines[0].Debit != 0 {
		t.Fatalf("first line polarity not repaired: debit=%.3f credit=%.3f", parsed.Lines[0].Debit, parsed.Lines[0].Credit)
	}
	if math.Abs(parsed.Lines[1].Credit-31072.006) > 0.001 || parsed.Lines[1].Debit != 0 {
		t.Fatalf("second line polarity not repaired: debit=%.3f credit=%.3f", parsed.Lines[1].Debit, parsed.Lines[1].Credit)
	}
	if math.Abs(parsed.Lines[2].Debit-527.500) > 0.001 || parsed.Lines[2].Credit != 0 {
		t.Fatalf("third line polarity not repaired: debit=%.3f credit=%.3f", parsed.Lines[2].Debit, parsed.Lines[2].Credit)
	}
	if math.Abs(parsed.TotalDebits-527.500) > 0.001 {
		t.Fatalf("total debits = %.3f, want 527.500", parsed.TotalDebits)
	}
	if math.Abs(parsed.TotalCredits-31572.006) > 0.001 {
		t.Fatalf("total credits = %.3f, want 31572.006", parsed.TotalCredits)
	}
}

func TestParseAlSalamOCRFormatMatchesScannedLayout(t *testing.T) {
	statementText := `BEACON CONTROLS W.L.L
Statement of Account
From: 01 Jan 2026
To: 01 Feb 2026
SWIFT Code: DMOCBHBM
IBAN: BH42DMOC00216912340100
Account Number: 216912340100
Account Currency: BHD
Closing Balance: BHD 15,523.604
Total Credits: BHD 2,945.500
Total Debits: BHD 0.000
Date Description Value Date Amount Balance
07 Jan 2026 Opening Balance 07 Jan 2026 (945.500) 12,578.104
20 Jan 2026 TT2600200633 - Chque Cancelled 20 Jan 2026 2,945.500 15,523.604
Closing Balance 15,523.604`

	parsed, err := parseAlSalamOCRFormat(statementText)
	if err != nil {
		t.Fatalf("parseAlSalamOCRFormat returned error: %v", err)
	}

	if parsed.AccountNumber != "216912340100" {
		t.Fatalf("account number = %q", parsed.AccountNumber)
	}
	if parsed.IBAN != "BH42DMOC00216912340100" {
		t.Fatalf("IBAN = %q", parsed.IBAN)
	}
	if len(parsed.Lines) != 2 {
		t.Fatalf("lines = %d, want 2", len(parsed.Lines))
	}
	if math.Abs(parsed.Lines[0].Debit-945.5) > 0.001 {
		t.Fatalf("opening row debit = %.3f, want 945.500", parsed.Lines[0].Debit)
	}
	if math.Abs(parsed.Lines[1].Credit-2945.5) > 0.001 {
		t.Fatalf("credit row credit = %.3f, want 2945.500", parsed.Lines[1].Credit)
	}
}
