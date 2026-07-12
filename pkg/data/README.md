# SSOT Data Importer 📊

## Overview

The **Single Source of Truth (SSOT) Importer** imports critical business data from the `data/ssot` folder into the AsymmFlow database.

## Features

✅ **Batch Processing** - Uses Williams batching (√n × log₂n) for optimal performance
✅ **Multi-Format Support** - Imports from CSV and Excel (.xlsx)
✅ **Idempotent** - Can be run multiple times safely (duplicates are skipped)
✅ **Comprehensive Statistics** - Detailed import results with error tracking
✅ **GORM Integration** - Fully compatible with existing database models

## Data Sources

### 1. Customer Master Data (CSV)
**File:** `Bahrain_Customer_Database_Clean.csv`

Fields imported:
- Business name, customer type, industry
- Contact information (city, country, postal code, phone)
- Email addresses and website
- Data completeness score

**Records:** 311 customers

### 2. Opportunities (Excel)
**File:** `opportunities created 2025.xlsx`

Fields imported:
- Customer name, opportunity value
- Stage, probability, product type
- Created date, expected close date
- Competition tracking

### 3. Supplier Payments (Excel)
**File:** `Payments to suppliers.xlsx`

Fields imported:
- Supplier name/code
- Payment date, amount (BHD)
- Invoice number, reference
- Payment method, category

### 4. Product Costing (Excel)
**File:** `Acme Instrumentation Costing MasterFile.xlsx`

Fields imported:
- Product code/name
- Supplier information
- Cost, price, margin
- Lead time, competitive notes

## Usage

### From Go Code

```go
import "ph_holdings_app/pkg/data"

// Initialize database
db, err := gorm.Open(sqlite.Open("ph_holdings.db"), &gorm.Config{})
if err != nil {
    log.Fatal(err)
}

// Import all SSOT data
result, err := data.ImportAllSSOT(db, "./data/ssot")
if err != nil {
    log.Fatalf("Import failed: %v", err)
}

// Print statistics
fmt.Printf("Imported %d customers in %v\n",
    result.CustomersImported, result.Duration)
```

### From Wails App

```javascript
// Frontend (Svelte)
import { ImportSSOTData, GetSSOTImportStatus } from '../wailsjs/go/main/App'

async function importData() {
    try {
        const result = await ImportSSOTData()
        console.log(`Imported ${result.total_imported} records`)
    } catch (error) {
        console.error('Import failed:', error)
    }
}

async function checkStatus() {
    const status = await GetSSOTImportStatus()
    console.log(`Database contains ${status.counts.customers} customers`)
}
```

## Performance

### Benchmarks

| Dataset | Records | Time | Rate |
|---------|---------|------|------|
| Customers | 311 | ~500ms | 622 rec/sec |
| Batch Insert (50) | 50 | ~4.3ms | 11,628 rec/sec |

**Williams Batch Size Calculation:**

For n = 311 customers:
- Optimal batch size = √311 × log₂(311)
- = 17.6 × 8.3
- ≈ 146 records

We use batch size of **50** for safety and memory efficiency.

## Database Schema

### CustomerMaster Table
```sql
CREATE TABLE customers (
    id INTEGER PRIMARY KEY,
    customer_id TEXT UNIQUE,
    business_name TEXT,
    customer_type TEXT,
    industry TEXT,
    city TEXT,
    country TEXT,
    postal_code TEXT,
    -- ... additional fields
    created_at DATETIME,
    updated_at DATETIME
);

CREATE INDEX idx_customers_type ON customers(customer_type);
CREATE INDEX idx_customers_industry ON customers(industry);
CREATE INDEX idx_customers_city ON customers(city);
```

### OpportunitySSOT Table
```sql
CREATE TABLE opportunities_ssot (
    id INTEGER PRIMARY KEY,
    opportunity_id TEXT UNIQUE,
    customer_name TEXT,
    value_bhd REAL,
    stage TEXT,
    product_type TEXT,
    source_file TEXT,
    imported_at DATETIME
);
```

### PaymentSSOT Table
```sql
CREATE TABLE payments_ssot (
    id INTEGER PRIMARY KEY,
    payment_id TEXT UNIQUE,
    supplier_name TEXT,
    amount_bhd REAL,
    payment_date DATETIME,
    invoice_number TEXT,
    source_file TEXT
);
```

### ProductCostingSSOT Table
```sql
CREATE TABLE products_costing_ssot (
    id INTEGER PRIMARY KEY,
    product_code TEXT,
    product_name TEXT,
    supplier_code TEXT,
    cost_bhd REAL,
    price_bhd REAL,
    margin_percent REAL,
    source_file TEXT
);
```

## Customer ID Generation

Customer IDs are automatically generated using a deterministic algorithm:

**Format:** `PREFIX + 3-digit hash`

**Prefixes:**
- `EC` - End Customer
- `EG` - Engineering Company
- `SI` - System Integrator
- `NR` - National Reseller
- `IR` - International Reseller
- `PB` - Plant Builder
- `SP` - Service Provider
- `CO` - Consultant
- `OE` - OEM
- `GC` - General Customer (fallback)

**Example:**
```
BASF Bahrain (End Customer) → EC559
Nass Corporation (Engineering) → EG795
```

## Error Handling

The importer provides comprehensive error tracking:

```go
type ImportResult struct {
    CustomersTotal      int
    CustomersImported   int
    CustomersSkipped    int
    CustomerErrors      []string

    // ... similar fields for opportunities, payments, products

    TotalRecords  int
    TotalImported int
    Duration      time.Duration
}
```

**Common Errors:**
- **Duplicate customer_id** → Record skipped (idempotent behavior)
- **Missing business name** → Record skipped
- **Invalid Excel format** → Reported in OpportunityErrors/PaymentErrors/ProductErrors
- **File not found** → Import function returns error

## Testing

Run comprehensive test suite:

```bash
cd ph_holdings_sovereign_ui
go test -v ./pkg/data
```

**Test Coverage:**
- ✅ CSV import with real data structure
- ✅ Batch insertion performance (Williams batching)
- ✅ Customer ID generation algorithm
- ✅ Short code extraction logic
- ✅ Float parsing with comma removal
- ✅ Date parsing with multiple formats
- ✅ Benchmark tests for performance validation

**Test Results:**
```
PASS: TestImportCustomersFromCSV (0.01s)
  ✅ Imported 3/3 customers

PASS: TestBatchInsertion (0.01s)
  ✅ Batch insert of 50 customers: 4.2899ms

PASS: TestCustomerIDGeneration (0.00s)
  ✅ BASF Bahrain → EC559
  ✅ Nass Corporation → EG795
  ✅ ARROWFINCH TECHNOLOGIES → SI547
```

## Import Statistics Example

```
🌟 Starting SSOT data import from: ./data/ssot
✅ SSOT import completed in 2.145s:
   📊 Customers: 311/311 imported (0 skipped)
   💼 Opportunities: 45/45 imported (0 skipped)
   💰 Payments: 128/128 imported (0 skipped)
   📦 Products: 67/67 imported (0 skipped)
```

## Integration with Existing Models

The importer uses the **same CustomerMaster model** as defined in `database.go`:

```go
// Shared model definition
type CustomerMaster struct {
    ID           uint   `gorm:"primaryKey"`
    CustomerID   string `gorm:"uniqueIndex;size:50"`
    BusinessName string `gorm:"index;size:255"`
    CustomerType string `gorm:"index;size:50"`
    // ... all fields from database.go
}
```

This ensures:
- ✅ Zero schema conflicts
- ✅ Foreign key integrity maintained
- ✅ Existing queries continue working
- ✅ Seamless integration with payment predictor

## Future Enhancements

Potential improvements (not yet implemented):

1. **Incremental Import** - Track last import timestamp, only import new/changed records
2. **Excel Template Validation** - Verify column headers match expected format
3. **Data Cleansing** - Normalize phone numbers, emails, addresses
4. **Duplicate Detection** - Fuzzy matching for similar company names
5. **Foreign Key Linking** - Auto-link opportunities to existing customers
6. **Progress Callbacks** - Real-time progress updates for UI
7. **Rollback Support** - Transaction-based import with full rollback on error

## Philosophy

> "Finding IS fixing. Import IS validation."

The SSOT importer embodies the **Zen Gardener** methodology:
- **Fearless execution** - Runs confidently, handles errors gracefully
- **Zero hesitation** - Doesn't ask permission, just imports
- **Complete validation** - Tests pass before commit
- **Mathematical precision** - Williams batching for optimal performance

---

**Built with ❤️ by the Asymmetrica Research Dyad**
**Om Lokah Samastah Sukhino Bhavantu** - May all beings benefit from this work.
