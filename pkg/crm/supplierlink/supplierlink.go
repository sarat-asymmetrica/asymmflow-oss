// Package supplierlink resolves the product→supplier relationship (PH
// convergence Band-2 rows 15-16, PH product_supplier_link_policy.go 55654d3/
// 7034098). The engine carries only the resolution MECHANISM — a four-tier
// fallback chain and a two-pass commercial-token search. The alias VOCABULARY
// (canonical code spellings, brand aliases) is company knowledge and is
// injected via AliasConfig so it can live in overlay configuration, never in
// engine code.
package supplierlink

import (
	"fmt"
	"strings"

	"gorm.io/gorm"

	"ph_holdings_app/pkg/crm"
)

// AliasConfig is the injectable commercial vocabulary.
type AliasConfig struct {
	// CanonicalCodes maps a known variant supplier-code spelling (uppercase)
	// to the canonical code stored on the supplier row, e.g. "SVX" -> "SRVX".
	CanonicalCodes map[string]string
	// BrandAliases maps an uppercased token to the search terms it implies,
	// e.g. a brand name to the supplier names that carry it.
	BrandAliases map[string][]string
}

// CanonicalCode returns the canonical spelling for a raw supplier code:
// uppercased, trimmed, and mapped through CanonicalCodes when known.
func (c AliasConfig) CanonicalCode(raw string) string {
	code := strings.ToUpper(strings.TrimSpace(raw))
	if canonical, ok := c.CanonicalCodes[code]; ok {
		return canonical
	}
	return code
}

// searchTerms expands one token into the ordered list of terms worth trying:
// the raw token, its canonical code, then any configured brand aliases.
func (c AliasConfig) searchTerms(token string) []string {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil
	}
	seen := map[string]struct{}{}
	terms := make([]string, 0, 4)
	add := func(t string) {
		t = strings.TrimSpace(t)
		if t == "" {
			return
		}
		key := strings.ToUpper(t)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		terms = append(terms, t)
	}
	add(token)
	add(c.CanonicalCode(token))
	for _, alias := range c.BrandAliases[strings.ToUpper(token)] {
		add(alias)
	}
	// Multi-word tokens (product names like "Oxan Analytics 1900 Gas
	// Analyzer") lead with the brand: also try the first two words and the
	// first word, expanding each through the alias table.
	if words := strings.Fields(token); len(words) > 1 {
		for _, lead := range []string{strings.Join(words[:2], " "), words[0]} {
			add(lead)
			add(c.CanonicalCode(lead))
			for _, alias := range c.BrandAliases[strings.ToUpper(lead)] {
				add(alias)
			}
		}
	}
	return terms
}

func escapeLikeWildcards(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}

// FindSupplierByCommercialToken finds a supplier from a free-text commercial
// token. Two passes per expanded term: exact match on code/name first, then a
// LIKE scan over name, brands handled, and product types.
func FindSupplierByCommercialToken(db *gorm.DB, token string, cfg AliasConfig) (*crm.SupplierMaster, error) {
	terms := cfg.searchTerms(token)
	if len(terms) == 0 {
		return nil, fmt.Errorf("empty supplier token")
	}

	for _, term := range terms {
		upper := strings.ToUpper(term)
		var supplier crm.SupplierMaster
		if err := db.
			Where("UPPER(supplier_code) = ? OR UPPER(supplier_name) = ?", upper, upper).
			First(&supplier).Error; err == nil {
			return &supplier, nil
		}
	}

	for _, term := range terms {
		pattern := "%" + escapeLikeWildcards(term) + "%"
		var supplier crm.SupplierMaster
		if err := db.
			Where(`supplier_name LIKE ? ESCAPE '\' OR brands_handled LIKE ? ESCAPE '\' OR product_types LIKE ? ESCAPE '\'`,
				pattern, pattern, pattern).
			First(&supplier).Error; err == nil {
			return &supplier, nil
		}
	}

	return nil, fmt.Errorf("no supplier matches token %q", token)
}

// ResolveSupplierForProduct resolves a product's supplier through the
// four-tier fallback chain: exact supplier ID, exact supplier code, canonical
// supplier code, then commercial-token search over the product's code, name,
// and description. It never writes; see NormalizeProductSupplierLink for the
// write side.
func ResolveSupplierForProduct(db *gorm.DB, product crm.ProductMaster, cfg AliasConfig) (*crm.SupplierMaster, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection not available")
	}

	if supplierID := strings.TrimSpace(product.SupplierID); supplierID != "" {
		var supplier crm.SupplierMaster
		if err := db.First(&supplier, "id = ?", supplierID).Error; err == nil {
			return &supplier, nil
		}
	}

	if supplierCode := strings.TrimSpace(product.SupplierCode); supplierCode != "" {
		var supplier crm.SupplierMaster
		if err := db.Where("supplier_code = ?", supplierCode).First(&supplier).Error; err == nil {
			return &supplier, nil
		}
		if canonical := cfg.CanonicalCode(supplierCode); canonical != supplierCode {
			if err := db.Where("supplier_code = ?", canonical).First(&supplier).Error; err == nil {
				return &supplier, nil
			}
		}
	}

	for _, token := range []string{product.SupplierCode, product.ProductName, product.Description, product.ProductCode} {
		if supplier, err := FindSupplierByCommercialToken(db, token, cfg); err == nil {
			return supplier, nil
		}
	}

	return nil, fmt.Errorf("supplier not found for product %s", product.ProductCode)
}

// NormalizeProductSupplierLink resolves and stamps the canonical supplier ID
// and code onto the product, returning the corrected copy. Callers decide
// what a resolution failure means (seed paths keep the product unlinked
// rather than fabricating a dangling foreign key).
func NormalizeProductSupplierLink(db *gorm.DB, product crm.ProductMaster, cfg AliasConfig) (crm.ProductMaster, error) {
	supplier, err := ResolveSupplierForProduct(db, product, cfg)
	if err != nil {
		return product, err
	}
	product.SupplierID = supplier.ID
	product.SupplierCode = supplier.SupplierCode
	return product, nil
}
