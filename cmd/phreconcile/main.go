// phreconcile is the PH → OSS migration reconciliation gate (PH convergence
// Mission H). It compares aggregate counts and money sums — to the fils —
// between the PH-format source file and the imported OSS-format destination,
// and exits non-zero on any mismatch. Run it after every phimport; the
// cutover runbook treats a non-zero exit as an abort.
//
// Usage:
//
//	phreconcile -source C:\path\to\ph_holdings.db -dest C:\path\to\asymmflow.db
//
// Both files are opened read-only; no data ever enters this repository.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"ph_holdings_app/pkg/data/phreconcile"
)

func main() {
	source := flag.String("source", "", "path to the PH-format SQLite file (opened read-only)")
	dest := flag.String("dest", "", "path to the imported OSS-format SQLite file (opened read-only)")
	flag.Parse()

	if *source == "" || *dest == "" {
		flag.Usage()
		os.Exit(2)
	}

	report, err := phreconcile.Run(context.Background(), phreconcile.Options{
		SourcePath: *source,
		DestPath:   *dest,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "reconciliation could not run: %v\n", err)
		os.Exit(1)
	}

	out, _ := json.MarshalIndent(report, "", "  ")
	fmt.Println(string(out))

	if !report.Pass {
		fmt.Fprintf(os.Stderr, "\nRECONCILIATION FAILED: %d/%d checks matched. A mismatch is a live bug in the source or the mapping — stop and ask, do not fudge.\n",
			report.Matched, report.Matched+report.Failed)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "\nRECONCILED: %d/%d checks matched to the fils.\n", report.Matched, report.Matched)
}
