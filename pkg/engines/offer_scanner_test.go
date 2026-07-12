package engines

import (
	"testing"
)

func TestOfferScanner_ParseFolderName(t *testing.T) {
	scanner := NewOfferScanner(".")

	tests := []struct {
		folderName string
		wantID     string
		wantCust   string
		wantProd   string
	}{
		{"101 VERTEX AIT", "101", "VERTEX", "AIT"},
		{"102 AQUAPURE FIT", "102", "AQUAPURE", "FIT"},
		{"103 NPC SP", "103", "NPC", "SP"},
		{"109 DELTA", "109", "DELTA", ""},
	}

	for _, tt := range tests {
		t.Run(tt.folderName, func(t *testing.T) {
			offerID, customer, productType := scanner.ParseFolderName(tt.folderName)
			if offerID != tt.wantID {
				t.Errorf("OfferID = %v, want %v", offerID, tt.wantID)
			}
			if customer != tt.wantCust {
				t.Errorf("CustomerName = %v, want %v", customer, tt.wantCust)
			}
			if productType != tt.wantProd {
				t.Errorf("ProductType = %v, want %v", productType, tt.wantProd)
			}
		})
	}
}
