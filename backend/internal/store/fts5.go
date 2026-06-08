package store

import "strings"

// buildFTSMatch turns free-text user input into a safe FTS5 MATCH expression.
//
// User input can contain FTS5 operators (AND, OR, NOT, NEAR, parentheses,
// asterisks, colons) that would otherwise cause a syntax error or behave
// unexpectedly. We defuse all of that by splitting on whitespace and wrapping
// each token as a double-quoted FTS5 string token; multiple tokens are
// implicitly AND-ed. Embedded double quotes are doubled per FTS5 escaping
// rules. Returns "" when there is nothing searchable, in which case callers
// should fall back to a non-FTS path.
func buildFTSMatch(search string) string {
	fields := strings.Fields(search)
	if len(fields) == 0 {
		return ""
	}
	var tokens []string
	for _, f := range fields {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		// Escape embedded double quotes by doubling them, then wrap as a
		// quoted FTS5 string token so operators inside are treated literally.
		escaped := strings.ReplaceAll(f, `"`, `""`)
		tokens = append(tokens, `"`+escaped+`"`)
	}
	if len(tokens) == 0 {
		return ""
	}
	return strings.Join(tokens, " ")
}
