package main

import (
	"regexp"

	"ph_holdings_app/pkg/kernel/text"
)

// escapeLikeWildcards escapes SQL LIKE wildcards (%, _) for safe LIKE queries
// SECURITY: Prevents LIKE injection attacks by escaping wildcards with backslash
// Usage: db.Where("name LIKE ? ESCAPE '\\\\'", "%"+escapeLikeWildcards(userInput)+"%")
// Canonical implementation: pkg/kernel/text.EscapeLike (Wave 5).
func escapeLikeWildcards(s string) string {
	return text.EscapeLike(s)
}

// isValidSQLIdentifier validates that a string is a safe SQL identifier
// SECURITY: Prevents SQL injection in DDL statements (DROP INDEX, ALTER TABLE)
// Only allows alphanumeric characters and underscores, starting with letter or underscore
func isValidSQLIdentifier(name string) bool {
	matched, _ := regexp.MatchString(`^[a-zA-Z_][a-zA-Z0-9_]*$`, name)
	return matched
}
