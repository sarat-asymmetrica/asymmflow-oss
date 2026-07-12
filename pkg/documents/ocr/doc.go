package ocr

type PageContent struct {
	PageNumber int    `json:"page_number"`
	Text       string `json:"text"`
}

type OCREngine interface {
	ExtractText(filePath string) (string, error)
	ExtractPages(filePath string) ([]PageContent, error)
	SupportedFormats() []string
}
