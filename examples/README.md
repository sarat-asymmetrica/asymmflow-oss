# Examples Directory

This directory contains standalone example programs demonstrating Acme Instrumentation app functionality.

## Build Tags

All example files use `//go:build ignore` which excludes them from normal `go build ./...` commands.
This prevents "multiple main functions" errors.

## Usage

### Run Individual Examples

Each example can be run directly:

```bash
# Invoice generator demo
go run generate_sample_invoice.go

# File watcher demo
go run file_watcher_demo.go [watch_directory]
```

### Examples

#### generate_sample_invoice.go
Demonstrates creating invoice data structures for PDF generation.

**Output:**
```
=== Acme Instrumentation PDF Invoice Generator Demo ===
Invoice Type:   TAX INVOICE
Invoice Number: INV-2025-0106
Date:           15-Sep-2025
Customer:       North Grid Authority
...
```

#### file_watcher_demo.go
Demonstrates real-time file watching for RFQs, supplier XML, offers, and invoices.

**Usage:**
```bash
go run file_watcher_demo.go "C:\Data\Acme Instrumentation\RFQs"
```

**Output:**
```
╔═══════════════════════════════════════════════════════════════╗
║ Acme Instrumentation File Watcher Demo                       ║
╚═══════════════════════════════════════════════════════════════╝

Watching: C:\Data\Acme Instrumentation\RFQs
Press Ctrl+C to stop

✓ Watcher started successfully

Try creating files in the following directories:
  - RFQs/    → Create .msg or .eml files
  - EH_XML/  → Create .xml files
  - Offers/  → Create any files
  - Invoices/ → Create .pdf files
```

## Technical Details

### Why Build Tags?

Go doesn't allow multiple `package main` with `func main()` in the same package.
Using `//go:build ignore` at the top of each example file:

1. Excludes them from `go build ./...`
2. Allows `go run <file>.go` to work
3. Prevents "main redeclared" compilation errors

### Standard Go Practice

This is the standard pattern for Go example programs that don't integrate with the main codebase.
See: https://pkg.go.dev/cmd/go#hdr-Build_constraints
