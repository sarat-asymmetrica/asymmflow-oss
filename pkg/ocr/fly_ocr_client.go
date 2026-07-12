package ocr

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

type FlyOCRClient struct {
	APIUrl string
	APIKey string
}

func NewFlyOCRClient(apiUrl, apiKey string) *FlyOCRClient {
	return &FlyOCRClient{APIUrl: apiUrl, APIKey: apiKey}
}

func (c *FlyOCRClient) BatchPreprocess(ctx context.Context, filePath string) (*PreprocessResult, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	encoded := base64.StdEncoding.EncodeToString(data)
	req := map[string]any{"image": encoded}
	var result PreprocessResult
	return &result, c.callAPI(ctx, "/api/ocr/batch-preprocess", req, &result)
}

func (c *FlyOCRClient) QualityGate(ctx context.Context, text string) (*QualityResult, error) {
	req := map[string]any{"data": text, "resonance": true}
	var result QualityResult
	return &result, c.callAPI(ctx, "/api/ocr/quality-gate", req, &result)
}

func (c *FlyOCRClient) TableExtraction(ctx context.Context, text string, tallyData any) (*ArchaeologyResult, error) {
	prompt := fmt.Sprintf("Extract tables/entities from: %s", text)
	if tallyData != nil {
		prompt += ". Reconcile with Tally data provided."
	}
	req := map[string]any{"prompt": prompt, "model": "table_extraction"}
	var result ArchaeologyResult
	return &result, c.callAPI(ctx, "/api/kernel/table_extraction", req, &result)
}

func (c *FlyOCRClient) callAPI(ctx context.Context, endpoint string, req any, result any) error {
	jsonData, err := json.Marshal(req)
	if err != nil {
		return err
	}
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.APIUrl+endpoint, strings.NewReader(string(jsonData)))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)
	}
	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, result)
}

type PreprocessResult struct {
	Text string `json:"text"`
}

type QualityResult struct {
	Quality    string  `json:"quality"`
	Confidence float64 `json:"confidence"`
}

type ArchaeologyResult struct {
	Tables     []map[string]any `json:"tables"`
	Entities   []map[string]any `json:"entities"`
	Reconciled *map[string]any  `json:"reconciled,omitempty"`
}
