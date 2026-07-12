// phimport is the one-time PH-format → OSS-format SQLite importer (PH
// convergence Mission C). It moves rows between two database FILES on the
// operator's machine; no data ever enters this repository.
//
// Usage:
//
//	phimport -source C:\path\to\ph_holdings.db -dest C:\path\to\asymmflow.db
//
// The destination must already carry the OSS schema: run the OSS app once
// against a fresh database file to provision it, close the app, then import.
// Always work on COPIES — this tool inserts into the destination and aborts
// (rolling back) on any foreign-key violation, but it does not back up files.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"ph_holdings_app/pkg/data/phimport"
)

func main() {
	source := flag.String("source", "", "path to the PH-format SQLite file (opened read-only)")
	dest := flag.String("dest", "", "path to the OSS-format SQLite file (schema already provisioned)")
	flag.Parse()

	if *source == "" || *dest == "" {
		flag.Usage()
		os.Exit(2)
	}
	if _, err := os.Stat(*source); err != nil {
		fmt.Fprintf(os.Stderr, "source: %v\n", err)
		os.Exit(1)
	}
	if _, err := os.Stat(*dest); err != nil {
		fmt.Fprintf(os.Stderr, "dest: %v (provision it by running the OSS app once against a fresh database file)\n", err)
		os.Exit(1)
	}

	report, err := phimport.Run(context.Background(), phimport.Options{
		SourcePath: *source,
		DestPath:   *dest,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "import failed (destination rolled back): %v\n", err)
		os.Exit(1)
	}

	out, _ := json.MarshalIndent(report, "", "  ")
	fmt.Println(string(out))

	fmt.Fprintln(os.Stderr, "\nImport committed.")
	fmt.Fprintln(os.Stderr, "Reminders:")
	fmt.Fprintln(os.Stderr, "  - Invoice/credit-note HMAC hashes were blanked; the app recomputes them on next startup.")
	if report.EncryptedSettingsSkipped > 0 {
		fmt.Fprintf(os.Stderr, "  - %d encrypted settings row(s) were NOT carried (machine-bound ciphertext) — re-enter those secrets in the app.\n", report.EncryptedSettingsSkipped)
	}
	if report.ReceiptsTransformed > 0 {
		fmt.Fprintf(os.Stderr, "  - %d customer receipt(s) carried %.3f BHD of unapplied on-account money as invoice-less payments (PC-D7); apply them to invoices in the app (reference carries receipt number + customer id).\n", report.ReceiptsTransformed, report.ReceiptsOnAccountBHD)
	}
	if len(report.Skipped) > 0 {
		fmt.Fprintf(os.Stderr, "  - %d table(s) skipped with reasons — review the report, especially PENDING DECISION rows.\n", len(report.Skipped))
	}
	if len(report.Unmapped) > 0 {
		fmt.Fprintf(os.Stderr, "  - %d UNMAPPED source table(s) — these were not copied; decide carry/skip explicitly before cutover.\n", len(report.Unmapped))
	}
}
