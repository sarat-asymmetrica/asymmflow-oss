package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/ncruces/go-sqlite3/driver"
)

var keyTables = []string{
	"customers",
	"suppliers",
	"products",
	"opportunities",
	"orders",
	"invoices",
	"purchase_orders",
	"supplier_invoices",
	"payments",
	"serial_numbers",
}

func main() {
	dbPath := flag.String("db", "ph_holdings.db", "SQLite database path to verify")
	flag.Parse()

	absPath, err := filepath.Abs(*dbPath)
	if err != nil {
		log.Fatalf("resolve database path: %v", err)
	}
	if info, err := os.Stat(absPath); err != nil {
		log.Fatalf("database not found: %s (%v)", absPath, err)
	} else if info.IsDir() {
		log.Fatalf("database path is a directory: %s", absPath)
	}

	db, err := sql.Open("sqlite3", absPath)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("ping database: %v", err)
	}
	if _, err := db.Exec("PRAGMA query_only = ON"); err != nil {
		log.Fatalf("enable read-only query mode: %v", err)
	}

	fmt.Printf("Database: %s\n", absPath)
	printScalar(db, "page_count", "PRAGMA page_count")
	printScalar(db, "page_size", "PRAGMA page_size")

	integrity := queryString(db, "PRAGMA integrity_check")
	fmt.Printf("integrity_check: %s\n", integrity)
	if strings.TrimSpace(strings.ToLower(integrity)) != "ok" {
		os.Exit(2)
	}

	foreignKeyIssues := countRows(db, "PRAGMA foreign_key_check")
	fmt.Printf("foreign_key_check_rows: %d\n", foreignKeyIssues)
	if foreignKeyIssues > 0 {
		os.Exit(3)
	}

	for _, table := range keyTables {
		if tableExists(db, table) {
			fmt.Printf("table_%s_rows: %d\n", table, countRows(db, "SELECT 1 FROM "+table))
		}
	}

	fmt.Println("sqlite_integrity: ok")
}

func printScalar(db *sql.DB, label, query string) {
	var value string
	if err := db.QueryRow(query).Scan(&value); err == nil {
		fmt.Printf("%s: %s\n", label, value)
	}
}

func queryString(db *sql.DB, query string) string {
	var value string
	if err := db.QueryRow(query).Scan(&value); err != nil {
		log.Fatalf("query %q: %v", query, err)
	}
	return value
}

func countRows(db *sql.DB, query string) int64 {
	rows, err := db.Query(query)
	if err != nil {
		log.Fatalf("query %q: %v", query, err)
	}
	defer rows.Close()

	var count int64
	for rows.Next() {
		count++
	}
	if err := rows.Err(); err != nil {
		log.Fatalf("iterate %q: %v", query, err)
	}
	return count
}

func tableExists(db *sql.DB, table string) bool {
	var name string
	err := db.QueryRow(
		"SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?",
		table,
	).Scan(&name)
	return err == nil && name == table
}
