# Wave 0: Archaeological Audit

## Prerequisites

```bash
# Build frontend dist (required for Go embed directive)
cd the AsymmFlow repository\frontend
npm install && npm run build

# If npm unavailable, create stub:
mkdir -p frontend/dist && echo "<html></html>" > frontend/dist/index.html

# Verify Go compilation
cd the AsymmFlow repository
go build ./...
```

## Phase 1: Dead Code Identification

### 1A: Build-Ignored Files (confirmed dead — DELETE)
```bash
grep -rl "// +build ignore\|//go:build ignore" --include="*.go" .
```

Known dead (9 files):
- benchmark_runner.go
- classifier_test_simple.go
- example_simple_ocr.go
- logger_demo.go
- scanner_demo.go
- test_classifier_standalone.go
- verify_password_security.go
- examples/arabic_rtl_demo.go
- examples/file_watcher_demo.go
- examples/generate_sample_invoice.go

### 1B: Manual Test Scripts (one-off ops — QUARANTINE)
```bash
find . -name "manual_*_test.go"
```

Known (13 files): All require env vars, mutate databases. Not real tests.

### 1C: Demo Files (DELETE)
```bash
find . -name "*_demo.go" -o -name "*demo*.go" | grep -v _test
```

Known: unified_demo.go, demo_scenarios.go, business_invariants_demo.go, scanner_demo.go, logger_demo.go

### 1D: Orphan Functions (never called)
```bash
# Extract all exported non-App functions
grep -rn "^func [A-Z]" --include="*.go" | grep -v "_test.go" | grep -v "func (a \*App)" > all_exported_funcs.txt

# Check each for references
while IFS= read -r line; do
  funcname=$(echo "$line" | sed 's/.*func \([A-Za-z0-9_]*\).*/\1/')
  refs=$(grep -rn "$funcname" --include="*.go" | grep -v "_test.go" | wc -l)
  if [ "$refs" -le 1 ]; then echo "ORPHAN: $line"; fi
done < all_exported_funcs.txt > orphan_functions.txt
```

### 1E: Stale Files (untouched 3+ months)
```bash
# Run against ph_holdings (has git history)
cd C:\Projects\asymmflow\ph_holdings
for f in $(find . -maxdepth 1 -name "*.go" -not -name "*_test.go"); do
  last_touch=$(git log -1 --format="%ai" -- "$f" 2>/dev/null)
  echo "$last_touch $f"
done | sort > file_last_touched.txt

# Filter: not touched since 2026-02-05
awk '$1 < "2026-02-05"' file_last_touched.txt > stale_files_3months.txt
```

## Phase 2: Temporal Coupling Analysis

### 2A: Co-Change Matrix (from ph_holdings git history)
```bash
cd C:\Projects\asymmflow\ph_holdings

# Get all commits that changed Go files
git log --format="%H" --diff-filter=M -- "*.go" > go_commits.txt

# For each commit, list which files changed together
while IFS= read -r hash; do
  git diff-tree --no-commit-id --name-only -r "$hash" | grep "\.go$" | sort
  echo "---"
done < go_commits.txt > co_change_raw.txt
```

### 2B: Extract Coupling Pairs (Python)
```python
# extract_coupling.py
from collections import defaultdict
from itertools import combinations

co_changes = defaultdict(int)
current_set = []

for line in open("co_change_raw.txt"):
    line = line.strip()
    if line == "---":
        for a, b in combinations(sorted(current_set), 2):
            co_changes[(a, b)] += 1
        current_set = []
    elif line:
        current_set.append(line)

pairs = sorted(co_changes.items(), key=lambda x: -x[1])
for (a, b), count in pairs[:100]:
    print(f"{count:4d}  {a}  <-->  {b}")
```

**Interpretation:**
- count > 10: Natural domain (files belong together)
- Involves app.go + service: Extraction candidate
- Involves database.go + service: Schema coupling (same domain)

## Phase 3: Hot Path Tracing

### 3A: Offer → Order → Invoice → Payment
```bash
# Trace the function chain
grep -n "func (a \*App) Create.*Offer" app.go
grep -n "func (a \*App) Convert.*Order" app.go
grep -n "func (a \*App) Create.*Invoice" *.go
grep -n "func (a \*App) Record.*Payment" *.go
```

### 3B: Bank Statement Import → Match → Reconcile
```bash
grep -n "func (a \*App).*Bank.*Import\|func (a \*App).*Bank.*Parse" *.go
grep -n "func (a \*App).*Match\|func (a \*App).*Reconcil" *.go
```

### 3C: OCR → Classify → Extract
```bash
grep -n "func (a \*App).*OCR\|func (a \*App).*Classify\|func (a \*App).*Analyze.*Document" *.go
```

### 3D: Butler AI Query
```bash
grep -n "func (a \*App).*Butler\|func (a \*App).*Chat" *.go
```

### 3E: Delivery Note + Serial Numbers
```bash
grep -n "func (a \*App).*Delivery\|func (a \*App).*Serial" *.go
```

## Phase 4: Dependency Analysis

### 4A: Internal Package Imports
```bash
grep -rn '"ph_holdings_app/' --include="*.go" | \
  sed 's/.*"ph_holdings_app\/\([^"]*\)".*/\1/' | \
  sort | uniq -c | sort -rn > internal_imports.txt
```

### 4B: Most-Referenced Structs
```bash
for struct in CustomerMaster Invoice Order Offer Payment PurchaseOrder DeliveryNote SerialNumber BankStatement Opportunity CreditNote Expense; do
  count=$(grep -rn "$struct" --include="*.go" | grep -v "_test.go" | wc -l)
  echo "$count $struct"
done | sort -rn > struct_usage.txt
```

### 4C: Service Fan-In
```bash
for svc in $(find . -maxdepth 1 -name "*_service.go"); do
  basename=$(basename "$svc" .go)
  funcs=$(grep -c "^func " "$svc")
  fanin=$(grep -rl "$basename" --include="*.go" | grep -v "$svc" | grep -v "_test.go" | wc -l)
  echo "fan-in:$fanin funcs:$funcs $svc"
done | sort -t: -k2 -rn > service_coupling.txt
```

## Phase 5: Technical Debt

### 5A: TODOs and FIXMEs
```bash
grep -rn "// TODO\|// FIXME\|// HACK\|// XXX" --include="*.go" > tech_debt_todos.txt
```

### 5B: Functions > 100 Lines
```bash
awk '/^func /{name=$0; start=NR} /^}/{if(NR-start>100) print FILENAME":"start" ("NR-start" lines) "name}' *.go > long_functions.txt
```

### 5C: CGO Dependencies
```bash
grep -rn "go-sqlite3\|go-fitz\|go-ole" --include="*.go" | grep -v "_test.go" > cgo_deps.txt
```

Known CGO deps (all replaceable):
- mattn/go-sqlite3 → ncruces/go-sqlite3
- gen2brain/go-fitz → alternative OCR or Wasm
- go-ole → keep (Windows-only, build-tagged)

## Phase 6: Test Suite Health

### 6A: Run Tests
```bash
go test ./... -count=1 -timeout 300s 2>&1 | tee test_results.txt
```

### 6B: Classify Tests
```bash
# Unit tests (in-memory, no env vars)
grep -l "setupTestApp\|:memory:" *_test.go > tests_unit.txt

# Integration tests (require env/network)
grep -l "t.Skip\|os.Getenv" *_test.go | grep -v "manual_" > tests_integration.txt

# Manual scripts (not real tests)
find . -name "manual_*_test.go" > tests_manual.txt
```

### 6C: Coverage Gaps
```bash
# Source files with no corresponding test
for src in $(find . -maxdepth 1 -name "*.go" -not -name "*_test.go"); do
  base=$(basename "$src" .go)
  if ! ls "${base}_test.go" 2>/dev/null | grep -q .; then
    echo "NO TEST: $src"
  fi
done > coverage_gaps.txt
```

## Phase 7: Unique Domain Logic Inventory

**The most important output: what CANNOT be generated.**

### What to look for:
- Acme Instrumentation-specific business rules (not generic CRUD)
- Bahrain bank statement formats
- Multi-division financial logic (Acme Instrumentation / Beacon Controls / PH Machinery)
- Arabic RTL document generation
- Specific offer numbering/revision schemes
- Butler AI prompts (Sarvam 105B context)
- Payment prediction algorithms
- FX revaluation logic (BHD multi-currency)

### Extract unique logic:
```bash
# Business rules
grep -rn "// Business rule\|// Acme Instrumentation\|// Division\|BHD\|bahrain" --include="*.go" > unique_business_rules.txt

# Custom algorithms (not CRUD)
grep -rn "func.*predict\|func.*match\|func.*classify\|func.*calculate\|func.*reconcil" --include="*.go" | grep -v _test > unique_algorithms.txt
```

## Deliverable

After running all phases, synthesize into:

```
docs/ARCHAEOLOGICAL_REPORT.md
├── Executive Summary (total LOC, health score, top 5 risks)
├── Dead Code Map (files to delete, quarantine, keep)
├── Domain Discovery (temporal coupling → natural boundaries)
├── Hot Path Traces (5 workflows mapped)
├── Dependency Graph (imports, struct usage, coupling)
├── Technical Debt (TODOs, long functions, CGO)
├── Test Health (pass/fail, unit/integration/manual)
├── Unique Logic Inventory (what generators CANNOT produce)
└── Recommended Wave Priority (which domain first)
```

## Interpretation Signals

**"Clean domain" (easy to extract/generate):**
- High internal co-change
- Low external coupling
- Tests exist and pass
- Functions < 100 LOC
- Mostly CRUD (generators handle this!)

**"Tangled domain" (burn and rebuild):**
- Co-changes with many unrelated files
- Heavy *App dependency
- Tests need full setupTestApp()
- Functions > 200 LOC
- Contains unique algorithms

**Action for tangled domains:**
1. Extract the unique logic
2. DELETE everything else
3. GENERATE the CRUD from schema_alchemy
4. Wire the unique logic into generated scaffolding
