package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
)

type MasterDataCleanupAudit struct {
	GeneratedAt          time.Time                      `json:"generated_at"`
	CustomerCandidates   []MasterDataDuplicateCandidate `json:"customer_candidates"`
	SupplierCandidates   []MasterDataDuplicateCandidate `json:"supplier_candidates"`
	AutoMergeCustomerIDs []string                       `json:"auto_merge_customer_ids"`
	AutoMergeSupplierIDs []string                       `json:"auto_merge_supplier_ids"`
}

type MasterDataDuplicateCandidate struct {
	EntityType      string                      `json:"entity_type"`
	NormalizedName  string                      `json:"normalized_name"`
	PrimaryID       string                      `json:"primary_id"`
	PrimaryName     string                      `json:"primary_name"`
	AutoMergeSafe   bool                        `json:"auto_merge_safe"`
	AutoMergeReason string                      `json:"auto_merge_reason"`
	Members         []MasterDataDuplicateMember `json:"members"`
}

type MasterDataDuplicateMember struct {
	ID                   string    `json:"id"`
	Name                 string    `json:"name"`
	CreatedAt            time.Time `json:"created_at"`
	InvoiceCount         int       `json:"invoice_count"`
	OrderCount           int       `json:"order_count"`
	OfferCount           int       `json:"offer_count"`
	ContactCount         int       `json:"contact_count"`
	OpportunityCount     int       `json:"opportunity_count"`
	PurchaseOrderCount   int       `json:"purchase_order_count"`
	SupplierInvoiceCount int       `json:"supplier_invoice_count"`
	SupplierPaymentCount int       `json:"supplier_payment_count"`
	ProductCount         int       `json:"product_count"`
	ActivityScore        int       `json:"activity_score"`
}

type masterDataCleanupResult struct {
	CustomerGroupsMerged  int      `json:"customer_groups_merged"`
	SupplierGroupsMerged  int      `json:"supplier_groups_merged"`
	CustomerRecordsMerged int      `json:"customer_records_merged"`
	SupplierRecordsMerged int      `json:"supplier_records_merged"`
	Notes                 []string `json:"notes"`
}

func (a *App) GetMasterDataCleanupAudit() (*MasterDataCleanupAudit, error) {
	if err := a.requirePermission("settings:update"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	return buildMasterDataCleanupAudit(a.db)
}

func (a *App) WriteMasterDataCleanupReport(outputPath string) (string, error) {
	if err := a.requirePermission("settings:update"); err != nil {
		return "", err
	}
	if a.db == nil {
		return "", fmt.Errorf("database not initialized")
	}
	audit, err := buildMasterDataCleanupAudit(a.db)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(outputPath) == "" {
		outputPath = filepath.Join("docs", "MASTER_DATA_CLEANUP_REVIEW_"+time.Now().Format("2006_01_02")+".md")
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return "", err
	}
	if err := os.WriteFile(outputPath, []byte(renderMasterDataCleanupReport(audit)), 0644); err != nil {
		return "", err
	}
	return outputPath, nil
}

func buildMasterDataCleanupAudit(db *gorm.DB) (*MasterDataCleanupAudit, error) {
	audit := &MasterDataCleanupAudit{GeneratedAt: time.Now()}

	customerCandidates, err := buildCustomerDuplicateCandidates(db)
	if err != nil {
		return nil, err
	}
	supplierCandidates, err := buildSupplierDuplicateCandidates(db)
	if err != nil {
		return nil, err
	}

	audit.CustomerCandidates = customerCandidates
	audit.SupplierCandidates = supplierCandidates
	for _, candidate := range customerCandidates {
		if candidate.AutoMergeSafe {
			audit.AutoMergeCustomerIDs = append(audit.AutoMergeCustomerIDs, candidate.PrimaryID)
		}
	}
	for _, candidate := range supplierCandidates {
		if candidate.AutoMergeSafe {
			audit.AutoMergeSupplierIDs = append(audit.AutoMergeSupplierIDs, candidate.PrimaryID)
		}
	}
	return audit, nil
}

func buildCustomerDuplicateCandidates(db *gorm.DB) ([]MasterDataDuplicateCandidate, error) {
	var customers []CustomerMaster
	if err := db.Where("deleted_at IS NULL").Find(&customers).Error; err != nil {
		return nil, fmt.Errorf("load customers: %w", err)
	}

	groups := map[string][]CustomerMaster{}
	for _, customer := range customers {
		key := normalizeMasterDataName(customer.BusinessName)
		if key == "" {
			continue
		}
		groups[key] = append(groups[key], customer)
	}

	candidates := make([]MasterDataDuplicateCandidate, 0)
	for key, members := range groups {
		if len(members) < 2 {
			continue
		}
		candidate, err := buildCustomerCandidate(db, key, members)
		if err != nil {
			return nil, err
		}
		candidates = append(candidates, candidate)
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].AutoMergeSafe != candidates[j].AutoMergeSafe {
			return candidates[i].AutoMergeSafe && !candidates[j].AutoMergeSafe
		}
		return candidates[i].PrimaryName < candidates[j].PrimaryName
	})
	return candidates, nil
}

func buildCustomerCandidate(db *gorm.DB, key string, customers []CustomerMaster) (MasterDataDuplicateCandidate, error) {
	members := make([]MasterDataDuplicateMember, 0, len(customers))
	for _, customer := range customers {
		member, err := buildCustomerDuplicateMember(db, customer)
		if err != nil {
			return MasterDataDuplicateCandidate{}, err
		}
		members = append(members, member)
	}
	sortDuplicateMembers(members)

	candidate := MasterDataDuplicateCandidate{
		EntityType:     "customer",
		NormalizedName: key,
		PrimaryID:      members[0].ID,
		PrimaryName:    members[0].Name,
		Members:        members,
	}
	candidate.AutoMergeSafe, candidate.AutoMergeReason = evaluateCustomerAutoMerge(members)
	return candidate, nil
}

func buildCustomerDuplicateMember(db *gorm.DB, customer CustomerMaster) (MasterDataDuplicateMember, error) {
	member := MasterDataDuplicateMember{
		ID:        customer.ID,
		Name:      customer.BusinessName,
		CreatedAt: customer.CreatedAt,
	}
	counts := []struct {
		table string
		col   string
		dst   *int
	}{
		{"invoices", "customer_id", &member.InvoiceCount},
		{"orders", "customer_id", &member.OrderCount},
		{"offers", "customer_id", &member.OfferCount},
		{"customer_contacts", "customer_id", &member.ContactCount},
		{"opportunities", "customer_id", &member.OpportunityCount},
	}
	for _, count := range counts {
		total, err := countRowsByColumn(db, count.table, count.col, customer.ID)
		if err != nil {
			return member, err
		}
		*count.dst = total
	}
	member.ActivityScore = member.InvoiceCount*1000 + member.OrderCount*300 + member.OfferCount*120 + member.OpportunityCount*80 + member.ContactCount*15 + nameCompletenessScore(member.Name)
	return member, nil
}

func buildSupplierDuplicateCandidates(db *gorm.DB) ([]MasterDataDuplicateCandidate, error) {
	var suppliers []SupplierMaster
	if err := db.Where("deleted_at IS NULL").Find(&suppliers).Error; err != nil {
		return nil, fmt.Errorf("load suppliers: %w", err)
	}

	groups := map[string][]SupplierMaster{}
	for _, supplier := range suppliers {
		key := normalizeMasterDataName(supplier.SupplierName)
		if key == "" {
			continue
		}
		groups[key] = append(groups[key], supplier)
	}

	candidates := make([]MasterDataDuplicateCandidate, 0)
	for key, members := range groups {
		if len(members) < 2 {
			continue
		}
		candidate, err := buildSupplierCandidate(db, key, members)
		if err != nil {
			return nil, err
		}
		candidates = append(candidates, candidate)
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].AutoMergeSafe != candidates[j].AutoMergeSafe {
			return candidates[i].AutoMergeSafe && !candidates[j].AutoMergeSafe
		}
		return candidates[i].PrimaryName < candidates[j].PrimaryName
	})
	return candidates, nil
}

func buildSupplierCandidate(db *gorm.DB, key string, suppliers []SupplierMaster) (MasterDataDuplicateCandidate, error) {
	members := make([]MasterDataDuplicateMember, 0, len(suppliers))
	for _, supplier := range suppliers {
		member, err := buildSupplierDuplicateMember(db, supplier)
		if err != nil {
			return MasterDataDuplicateCandidate{}, err
		}
		members = append(members, member)
	}
	sortDuplicateMembers(members)

	candidate := MasterDataDuplicateCandidate{
		EntityType:     "supplier",
		NormalizedName: key,
		PrimaryID:      members[0].ID,
		PrimaryName:    members[0].Name,
		Members:        members,
	}
	candidate.AutoMergeSafe, candidate.AutoMergeReason = evaluateSupplierAutoMerge(members)
	return candidate, nil
}

func buildSupplierDuplicateMember(db *gorm.DB, supplier SupplierMaster) (MasterDataDuplicateMember, error) {
	member := MasterDataDuplicateMember{
		ID:        supplier.ID,
		Name:      supplier.SupplierName,
		CreatedAt: supplier.CreatedAt,
	}
	counts := []struct {
		table string
		col   string
		dst   *int
	}{
		{"purchase_orders", "supplier_id", &member.PurchaseOrderCount},
		{"supplier_invoices", "supplier_id", &member.SupplierInvoiceCount},
		{"supplier_payments", "supplier_id", &member.SupplierPaymentCount},
		{"supplier_contacts", "supplier_id", &member.ContactCount},
		{"products", "supplier_id", &member.ProductCount},
	}
	for _, count := range counts {
		total, err := countRowsByColumn(db, count.table, count.col, supplier.ID)
		if err != nil {
			return member, err
		}
		*count.dst = total
	}
	member.ActivityScore = member.PurchaseOrderCount*1000 + member.SupplierInvoiceCount*700 + member.SupplierPaymentCount*500 + member.ProductCount*120 + member.ContactCount*15 + nameCompletenessScore(member.Name)
	return member, nil
}

func ApplyLowRiskMasterDataCleanup(db *gorm.DB) (*masterDataCleanupResult, *MasterDataCleanupAudit, error) {
	audit, err := buildMasterDataCleanupAudit(db)
	if err != nil {
		return nil, nil, err
	}

	result := &masterDataCleanupResult{}
	err = db.Transaction(func(tx *gorm.DB) error {
		for _, candidate := range audit.CustomerCandidates {
			if !candidate.AutoMergeSafe || len(candidate.Members) < 2 {
				continue
			}
			if err := mergeCustomerCandidate(tx, candidate); err != nil {
				return err
			}
			result.CustomerGroupsMerged++
			result.CustomerRecordsMerged += len(candidate.Members) - 1
		}
		for _, candidate := range audit.SupplierCandidates {
			if !candidate.AutoMergeSafe || len(candidate.Members) < 2 {
				continue
			}
			if err := mergeSupplierCandidate(tx, candidate); err != nil {
				return err
			}
			result.SupplierGroupsMerged++
			result.SupplierRecordsMerged += len(candidate.Members) - 1
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return result, audit, nil
}

func mergeCustomerCandidate(tx *gorm.DB, candidate MasterDataDuplicateCandidate) error {
	var primary CustomerMaster
	if err := tx.Where("id = ?", candidate.PrimaryID).First(&primary).Error; err != nil {
		return fmt.Errorf("load primary customer %s: %w", candidate.PrimaryID, err)
	}

	for _, member := range candidate.Members[1:] {
		var duplicate CustomerMaster
		if err := tx.Where("id = ?", member.ID).First(&duplicate).Error; err != nil {
			return fmt.Errorf("load duplicate customer %s: %w", member.ID, err)
		}
		if err := mergeCustomerRecordData(tx, &primary, duplicate); err != nil {
			return err
		}
		if err := moveEntityReferences(tx, "customer_id", duplicate.ID, primary.ID, "customers"); err != nil {
			return err
		}
		if err := moveEntityReferences(tx, "matched_customer_id", duplicate.ID, primary.ID, "customers"); err != nil {
			return err
		}
		if err := moveTypedEntityReferences(tx, "customer", duplicate.ID, primary.ID); err != nil {
			return err
		}
		if err := tx.Model(&CustomerMaster{}).Where("id = ?", duplicate.ID).Updates(map[string]any{
			"deleted_at": time.Now(),
			"updated_at": time.Now(),
		}).Error; err != nil {
			return fmt.Errorf("soft-delete duplicate customer %s: %w", duplicate.ID, err)
		}
	}

	return tx.Model(&CustomerMaster{}).Where("id = ?", primary.ID).Updates(primaryCustomerUpdates(primary)).Error
}

func mergeSupplierCandidate(tx *gorm.DB, candidate MasterDataDuplicateCandidate) error {
	var primary SupplierMaster
	if err := tx.Where("id = ?", candidate.PrimaryID).First(&primary).Error; err != nil {
		return fmt.Errorf("load primary supplier %s: %w", candidate.PrimaryID, err)
	}

	for _, member := range candidate.Members[1:] {
		var duplicate SupplierMaster
		if err := tx.Where("id = ?", member.ID).First(&duplicate).Error; err != nil {
			return fmt.Errorf("load duplicate supplier %s: %w", member.ID, err)
		}
		if err := mergeSupplierRecordData(tx, &primary, duplicate); err != nil {
			return err
		}
		if err := moveEntityReferences(tx, "supplier_id", duplicate.ID, primary.ID, "suppliers"); err != nil {
			return err
		}
		if err := moveTypedEntityReferences(tx, "supplier", duplicate.ID, primary.ID); err != nil {
			return err
		}
		if err := tx.Model(&SupplierMaster{}).Where("id = ?", duplicate.ID).Updates(map[string]any{
			"deleted_at": time.Now(),
			"updated_at": time.Now(),
		}).Error; err != nil {
			return fmt.Errorf("soft-delete duplicate supplier %s: %w", duplicate.ID, err)
		}
	}

	return tx.Model(&SupplierMaster{}).Where("id = ?", primary.ID).Updates(primarySupplierUpdates(primary)).Error
}

func mergeCustomerRecordData(tx *gorm.DB, primary *CustomerMaster, duplicate CustomerMaster) error {
	primary.ShortCode = pickLongerString(primary.ShortCode, duplicate.ShortCode)
	primary.TradingName = pickLongerString(primary.TradingName, duplicate.TradingName)
	primary.CRNumber = masterDataCoalesceString(primary.CRNumber, duplicate.CRNumber)
	primary.Status = masterDataCoalesceString(primary.Status, duplicate.Status)
	primary.PrimaryPhone = masterDataCoalesceString(primary.PrimaryPhone, duplicate.PrimaryPhone)
	primary.PrimaryEmail = masterDataCoalesceString(primary.PrimaryEmail, duplicate.PrimaryEmail)
	primary.MobileNumber = masterDataCoalesceString(primary.MobileNumber, duplicate.MobileNumber)
	primary.Website = masterDataCoalesceString(primary.Website, duplicate.Website)
	primary.AddressLine1 = pickLongerString(primary.AddressLine1, duplicate.AddressLine1)
	primary.City = masterDataCoalesceString(primary.City, duplicate.City)
	primary.Country = masterDataCoalesceString(primary.Country, duplicate.Country)
	primary.Industry = masterDataCoalesceString(primary.Industry, duplicate.Industry)
	primary.TRN = masterDataCoalesceString(primary.TRN, duplicate.TRN)
	primary.UpdatedAt = time.Now()
	return nil
}

func mergeSupplierRecordData(tx *gorm.DB, primary *SupplierMaster, duplicate SupplierMaster) error {
	primary.Country = masterDataCoalesceString(primary.Country, duplicate.Country)
	primary.TaxID = masterDataCoalesceString(primary.TaxID, duplicate.TaxID)
	primary.SupplierType = masterDataCoalesceString(primary.SupplierType, duplicate.SupplierType)
	primary.BrandsHandled = mergeDelimitedText(primary.BrandsHandled, duplicate.BrandsHandled)
	primary.ProductTypes = mergeDelimitedText(primary.ProductTypes, duplicate.ProductTypes)
	primary.PrimaryContact = masterDataCoalesceString(primary.PrimaryContact, duplicate.PrimaryContact)
	primary.Email = masterDataCoalesceString(primary.Email, duplicate.Email)
	primary.Phone = masterDataCoalesceString(primary.Phone, duplicate.Phone)
	primary.Address = pickLongerString(primary.Address, duplicate.Address)
	primary.BankName = masterDataCoalesceString(primary.BankName, duplicate.BankName)
	primary.AccountNumber = masterDataCoalesceString(primary.AccountNumber, duplicate.AccountNumber)
	primary.IBAN = masterDataCoalesceString(primary.IBAN, duplicate.IBAN)
	primary.SwiftCode = masterDataCoalesceString(primary.SwiftCode, duplicate.SwiftCode)
	primary.PaymentTerms = masterDataCoalesceString(primary.PaymentTerms, duplicate.PaymentTerms)
	if primary.Rating == 0 && duplicate.Rating > 0 {
		primary.Rating = duplicate.Rating
	}
	primary.Notes = mergeDelimitedText(primary.Notes, duplicate.Notes)
	primary.UpdatedAt = time.Now()
	return nil
}

func moveEntityReferences(tx *gorm.DB, column, fromID, toID string, excludeTable string) error {
	tables, err := listTablesWithColumn(tx, column)
	if err != nil {
		return err
	}
	for _, table := range tables {
		if table == excludeTable {
			continue
		}
		if !isValidSQLIdentifier(table) || !isValidSQLIdentifier(column) {
			continue
		}
		sql := fmt.Sprintf("UPDATE %s SET %s = ? WHERE %s = ?", table, column, column)
		if err := tx.Exec(sql, toID, fromID).Error; err != nil {
			return fmt.Errorf("move %s refs in %s: %w", column, table, err)
		}
	}
	return nil
}

func moveTypedEntityReferences(tx *gorm.DB, entityType, fromID, toID string) error {
	tables, err := listTablesWithColumn(tx, "entity_id")
	if err != nil {
		return err
	}
	for _, table := range tables {
		if !isValidSQLIdentifier(table) {
			continue
		}
		hasEntityType, err := tableHasColumn(tx, table, "entity_type")
		if err != nil {
			return err
		}
		if !hasEntityType {
			continue
		}
		sql := fmt.Sprintf("UPDATE %s SET entity_id = ? WHERE entity_id = ? AND entity_type = ?", table)
		if err := tx.Exec(sql, toID, fromID, entityType).Error; err != nil {
			return fmt.Errorf("move typed refs in %s: %w", table, err)
		}
	}
	return nil
}

func listTablesWithColumn(tx *gorm.DB, column string) ([]string, error) {
	rows, err := tx.Raw("SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%'").Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			return nil, err
		}
		ok, err := tableHasColumn(tx, table, column)
		if err != nil {
			return nil, err
		}
		if ok {
			tables = append(tables, table)
		}
	}
	sort.Strings(tables)
	return tables, nil
}

func tableHasColumn(tx *gorm.DB, table, column string) (bool, error) {
	if !isValidSQLIdentifier(table) {
		return false, nil
	}
	rows, err := tx.Raw(fmt.Sprintf("PRAGMA table_info(%s)", table)).Rows()
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name string
		var colType string
		var notNull int
		var defaultValue sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &colType, &notNull, &defaultValue, &pk); err != nil {
			return false, err
		}
		if strings.EqualFold(name, column) {
			return true, nil
		}
	}
	return false, nil
}

func countRowsByColumn(db *gorm.DB, table, column, id string) (int, error) {
	if !isValidSQLIdentifier(table) || !isValidSQLIdentifier(column) {
		return 0, fmt.Errorf("unsafe identifier %s.%s", table, column)
	}
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s = ? AND deleted_at IS NULL", table, column)
	var count int
	if err := db.Raw(query, id).Scan(&count).Error; err != nil {
		return 0, fmt.Errorf("count %s.%s: %w", table, column, err)
	}
	return count, nil
}

func evaluateCustomerAutoMerge(members []MasterDataDuplicateMember) (bool, string) {
	if len(members) < 2 {
		return false, ""
	}
	primary := members[0]
	secondaryOperational := 0
	secondaryTotal := 0
	for _, member := range members[1:] {
		secondaryOperational += member.InvoiceCount + member.OrderCount + member.OfferCount + member.OpportunityCount
		secondaryTotal += member.ActivityScore
	}
	if secondaryOperational == 0 {
		return true, "secondary customer records have no operational history"
	}
	if primary.ActivityScore >= secondaryTotal*4 && secondaryOperational <= len(members)-1 {
		return true, "primary customer record overwhelmingly dominates live activity"
	}
	return false, "review required because duplicate customers still carry live activity"
}

func evaluateSupplierAutoMerge(members []MasterDataDuplicateMember) (bool, string) {
	if len(members) < 2 {
		return false, ""
	}
	primary := members[0]
	secondaryOperational := 0
	secondaryTotal := 0
	for _, member := range members[1:] {
		secondaryOperational += member.PurchaseOrderCount + member.SupplierInvoiceCount + member.SupplierPaymentCount + member.ProductCount
		secondaryTotal += member.ActivityScore
	}
	if secondaryOperational == 0 {
		return true, "secondary supplier records have no procurement history"
	}
	if primary.ActivityScore >= secondaryTotal*4 && secondaryOperational <= len(members)-1 {
		return true, "primary supplier record overwhelmingly dominates live activity"
	}
	return false, "review required because duplicate suppliers still carry live activity"
}

func sortDuplicateMembers(members []MasterDataDuplicateMember) {
	sort.SliceStable(members, func(i, j int) bool {
		if members[i].ActivityScore != members[j].ActivityScore {
			return members[i].ActivityScore > members[j].ActivityScore
		}
		if nameCompletenessScore(members[i].Name) != nameCompletenessScore(members[j].Name) {
			return nameCompletenessScore(members[i].Name) > nameCompletenessScore(members[j].Name)
		}
		if !members[i].CreatedAt.Equal(members[j].CreatedAt) {
			return members[i].CreatedAt.Before(members[j].CreatedAt)
		}
		return members[i].Name < members[j].Name
	})
}

func renderMasterDataCleanupReport(audit *MasterDataCleanupAudit) string {
	var builder strings.Builder
	builder.WriteString("# Master Data Cleanup Review\n\n")
	builder.WriteString(fmt.Sprintf("Generated: %s\n\n", audit.GeneratedAt.Format(time.RFC3339)))
	builder.WriteString("## Customers\n\n")
	if len(audit.CustomerCandidates) == 0 {
		builder.WriteString("- No active duplicate customer groups detected.\n\n")
	} else {
		for _, candidate := range audit.CustomerCandidates {
			builder.WriteString(fmt.Sprintf("### %s\n\n", candidate.PrimaryName))
			builder.WriteString(fmt.Sprintf("- Normalized key: `%s`\n", candidate.NormalizedName))
			builder.WriteString(fmt.Sprintf("- Auto-merge: `%t`\n", candidate.AutoMergeSafe))
			builder.WriteString(fmt.Sprintf("- Reason: %s\n", candidate.AutoMergeReason))
			for _, member := range candidate.Members {
				builder.WriteString(fmt.Sprintf("- %s (`%s`) invoices=%d orders=%d offers=%d contacts=%d opportunities=%d score=%d\n",
					member.Name, member.ID, member.InvoiceCount, member.OrderCount, member.OfferCount, member.ContactCount, member.OpportunityCount, member.ActivityScore))
			}
			builder.WriteString("\n")
		}
	}
	builder.WriteString("## Suppliers\n\n")
	if len(audit.SupplierCandidates) == 0 {
		builder.WriteString("- No active duplicate supplier groups detected.\n")
	} else {
		for _, candidate := range audit.SupplierCandidates {
			builder.WriteString(fmt.Sprintf("### %s\n\n", candidate.PrimaryName))
			builder.WriteString(fmt.Sprintf("- Normalized key: `%s`\n", candidate.NormalizedName))
			builder.WriteString(fmt.Sprintf("- Auto-merge: `%t`\n", candidate.AutoMergeSafe))
			builder.WriteString(fmt.Sprintf("- Reason: %s\n", candidate.AutoMergeReason))
			for _, member := range candidate.Members {
				builder.WriteString(fmt.Sprintf("- %s (`%s`) pos=%d supplier_invoices=%d supplier_payments=%d contacts=%d products=%d score=%d\n",
					member.Name, member.ID, member.PurchaseOrderCount, member.SupplierInvoiceCount, member.SupplierPaymentCount, member.ContactCount, member.ProductCount, member.ActivityScore))
			}
			builder.WriteString("\n")
		}
	}
	return builder.String()
}

func normalizeMasterDataName(name string) string {
	normalized := strings.ToUpper(strings.TrimSpace(name))
	normalized = regexp.MustCompile(`W\s*\.?\s*L\s*\.?\s*L`).ReplaceAllString(normalized, " WLL ")
	normalized = regexp.MustCompile(`B\s*\.?\s*S\s*\.?\s*C\s*\.?\s*\(\s*C\s*\)`).ReplaceAllString(normalized, " BSCC ")
	normalized = regexp.MustCompile(`B\s*\.?\s*S\s*\.?\s*C`).ReplaceAllString(normalized, " BSC ")
	replacer := strings.NewReplacer(
		"&", " AND ",
		"+", " AND ",
		".", " ",
		",", " ",
		"-", " ",
		"(", " ",
		")", " ",
		"/", " ",
		"'", "",
		"\"", "",
	)
	normalized = replacer.Replace(normalized)
	normalized = regexp.MustCompile(`\s+`).ReplaceAllString(normalized, " ")
	tokens := strings.Fields(normalized)
	filtered := make([]string, 0, len(tokens))
	for idx, token := range tokens {
		switch token {
		case "WLL", "LLC", "LTD", "BSC", "BSCC", "THE":
			continue
		case "C":
			if idx == len(tokens)-1 {
				continue
			}
		}
		filtered = append(filtered, token)
	}
	return strings.TrimSpace(strings.Join(filtered, " "))
}

func nameCompletenessScore(name string) int {
	normalized := strings.ToUpper(name)
	score := len(strings.Fields(normalized))*10 + len(strings.TrimSpace(name))
	for _, token := range []string{"W.L.L", "WLL", "B.S.C", "BSC", "LLC", "LTD"} {
		if strings.Contains(normalized, token) {
			score += 15
			break
		}
	}
	return score
}

func masterDataCoalesceString(current, fallback string) string {
	if strings.TrimSpace(current) != "" {
		return current
	}
	return strings.TrimSpace(fallback)
}

func pickLongerString(current, fallback string) string {
	if len(strings.TrimSpace(fallback)) > len(strings.TrimSpace(current)) {
		return strings.TrimSpace(fallback)
	}
	return strings.TrimSpace(current)
}

func mergeDelimitedText(current, fallback string) string {
	cur := strings.TrimSpace(current)
	add := strings.TrimSpace(fallback)
	switch {
	case cur == "":
		return add
	case add == "":
		return cur
	case strings.Contains(cur, add):
		return cur
	default:
		return cur + "\n" + add
	}
}

func primaryCustomerUpdates(customer CustomerMaster) map[string]any {
	return map[string]any{
		"short_code":    customer.ShortCode,
		"trading_name":  customer.TradingName,
		"cr_number":     customer.CRNumber,
		"status":        customer.Status,
		"primary_phone": customer.PrimaryPhone,
		"primary_email": customer.PrimaryEmail,
		"mobile_number": customer.MobileNumber,
		"website":       customer.Website,
		"address_line1": customer.AddressLine1,
		"city":          customer.City,
		"country":       customer.Country,
		"industry":      customer.Industry,
		"trn":           customer.TRN,
		"updated_at":    time.Now(),
	}
}

func primarySupplierUpdates(supplier SupplierMaster) map[string]any {
	return map[string]any{
		"country":         supplier.Country,
		"tax_id":          supplier.TaxID,
		"supplier_type":   supplier.SupplierType,
		"brands_handled":  supplier.BrandsHandled,
		"product_types":   supplier.ProductTypes,
		"primary_contact": supplier.PrimaryContact,
		"email":           supplier.Email,
		"phone":           supplier.Phone,
		"address":         supplier.Address,
		"bank_name":       supplier.BankName,
		"account_number":  supplier.AccountNumber,
		"iban":            supplier.IBAN,
		"swift_code":      supplier.SwiftCode,
		"payment_terms":   supplier.PaymentTerms,
		"rating":          supplier.Rating,
		"notes":           supplier.Notes,
		"updated_at":      time.Now(),
	}
}
