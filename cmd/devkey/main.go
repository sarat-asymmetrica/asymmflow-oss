// devkey prints one unactivated admin license key from a dev database.
// INTEG-spike tooling: the lab frontend needs a key to activate against a
// fresh scratch DB (startup seeds 10 unactivated keys per role).
//
// Usage: go run ./cmd/devkey <path-to-db>
package main

import (
	"fmt"
	"os"

	"gorm.io/gorm"

	"github.com/ncruces/go-sqlite3/gormlite"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: devkey <db-path>")
		os.Exit(2)
	}
	db, err := gorm.Open(gormlite.Open(os.Args[1]), &gorm.Config{})
	if err != nil {
		fmt.Fprintln(os.Stderr, "open:", err)
		os.Exit(1)
	}
	var key string
	row := db.Raw("SELECT key FROM license_keys WHERE role = 'admin' AND activated = 0 LIMIT 1").Row()
	if err := row.Scan(&key); err != nil {
		fmt.Fprintln(os.Stderr, "scan:", err)
		os.Exit(1)
	}
	fmt.Println(key)
}
