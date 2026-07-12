package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// GraphClient for Microsoft Graph API operations
type GraphClient struct {
	accessToken string
	httpClient  *http.Client
}

// NewGraphClient creates authenticated Graph client
func NewGraphClient(accessToken string) *GraphClient {
	return &GraphClient{
		accessToken: accessToken,
		httpClient:  &http.Client{},
	}
}

// SendEmail via Outlook
type EmailRequest struct {
	ToRecipients []string `json:"toRecipients"`
	Subject      string   `json:"subject"`
	Body         string   `json:"body"`
	Attachments  []struct {
		Filename string `json:"filename"`
		Content  []byte `json:"content"`
	} `json:"attachments"`
}

func (g *GraphClient) SendEmail(ctx context.Context, req *EmailRequest) error {
	// Build message JSON
	msgBody := map[string]any{
		"message": map[string]any{
			"subject": req.Subject,
			"body": map[string]any{
				"contentType": "HTML",
				"content":     req.Body,
			},
			"toRecipients": toRecipientsJSON(req.ToRecipients),
		},
		"saveToSentItems": "true",
	}

	return g.graphRequest(ctx, "POST", "https://graph.microsoft.com/v1.0/me/sendMail", msgBody)
}

func toRecipientsJSON(emails []string) []map[string]any {
	recipients := make([]map[string]any, len(emails))
	for i, email := range emails {
		recipients[i] = map[string]any{
			"emailAddress": map[string]string{
				"address": email,
			},
		}
	}
	return recipients
}

// CreateCalendarEvent
type CalendarEventRequest struct {
	Subject     string
	Start       string // RFC3339 format
	End         string
	Description string
	Location    string
}

func (g *GraphClient) CreateCalendarEvent(ctx context.Context, req *CalendarEventRequest) error {
	event := map[string]any{
		"subject": req.Subject,
		"start": map[string]any{
			"dateTime": req.Start,
			"timeZone": "UTC",
		},
		"end": map[string]any{
			"dateTime": req.End,
			"timeZone": "UTC",
		},
		"bodyPreview": req.Description,
	}

	return g.graphRequest(ctx, "POST", "https://graph.microsoft.com/v1.0/me/events", event)
}

// UploadToOneDrive
func (g *GraphClient) UploadToOneDrive(ctx context.Context, fileName string, fileContent []byte) error {
	// Upload to root drive
	path := fmt.Sprintf("https://graph.microsoft.com/v1.0/me/drive/root:/%s:/content", url.PathEscape(fileName))

	req, err := http.NewRequestWithContext(ctx, "PUT", path, bytes.NewReader(fileContent))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+g.accessToken)
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("OneDrive upload failed: %s", string(body))
	}
	return nil
}

// GetContacts from address book
type Contact struct {
	ID    string
	Name  string
	Email string
	Phone string
}

func (g *GraphClient) GetContacts(ctx context.Context) ([]Contact, error) {
	path := "https://graph.microsoft.com/v1.0/me/contacts"

	req, err := http.NewRequestWithContext(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+g.accessToken)

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Value []struct {
			ID             string `json:"id"`
			DisplayName    string `json:"displayName"`
			EmailAddresses []struct {
				Address string `json:"address"`
			} `json:"emailAddresses"`
			MobilePhone string `json:"mobilePhone"`
		} `json:"value"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	contacts := make([]Contact, len(result.Value))
	for i, c := range result.Value {
		email := ""
		if len(c.EmailAddresses) > 0 {
			email = c.EmailAddresses[0].Address
		}
		contacts[i] = Contact{
			ID:    c.ID,
			Name:  c.DisplayName,
			Email: email,
			Phone: c.MobilePhone,
		}
	}
	return contacts, nil
}

// Helper: graphRequest makes authenticated Graph API call
func (g *GraphClient) graphRequest(ctx context.Context, method, urlStr string, body any) error {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, urlStr, reqBody)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+g.accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Graph API error (%d): %s", resp.StatusCode, string(bodyBytes))
	}
	return nil
}
