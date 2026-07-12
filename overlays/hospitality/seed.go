package hospitality

import "fmt"

// SeedDemo loads the synthetic Wasela Café menu and floor plan (idempotent —
// existing rows are left alone). Everything here is fictional, consistent with
// the repo's synthetic canon: prices are NET of VAT in halalas, standard-rated
// at 15%.
func (s *Service) SeedDemo() error {
	items := []MenuItem{
		{Name: "Saudi Coffee (Dallah)", Category: "beverage", UnitPriceHalalas: 1200, TaxRate: 0.15, Active: true},
		{Name: "Karak Chai", Category: "beverage", UnitPriceHalalas: 848, TaxRate: 0.15, Active: true},
		{Name: "Mint Lemonade", Category: "beverage", UnitPriceHalalas: 1400, TaxRate: 0.15, Active: true},
		{Name: "Chicken Kabsa", Category: "food", UnitPriceHalalas: 4200, TaxRate: 0.15, Active: true},
		{Name: "Lamb Mandi", Category: "food", UnitPriceHalalas: 5600, TaxRate: 0.15, Active: true},
		{Name: "Falafel Plate", Category: "food", UnitPriceHalalas: 2400, TaxRate: 0.15, Active: true},
		{Name: "Kunafa", Category: "dessert", UnitPriceHalalas: 1900, TaxRate: 0.15, Active: true},
		{Name: "Luqaimat", Category: "dessert", UnitPriceHalalas: 1500, TaxRate: 0.15, Active: true},
	}
	for _, item := range items {
		if err := s.db.Where("name = ?", item.Name).FirstOrCreate(&item).Error; err != nil {
			return fmt.Errorf("hospitality: seed menu item %q: %w", item.Name, err)
		}
	}

	for i := 1; i <= 8; i++ {
		table := DiningTable{Code: fmt.Sprintf("T%d", i), Seats: 4}
		if err := s.db.Where("code = ?", table.Code).FirstOrCreate(&table).Error; err != nil {
			return fmt.Errorf("hospitality: seed table %q: %w", table.Code, err)
		}
	}
	return nil
}
