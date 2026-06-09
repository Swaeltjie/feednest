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

// ftsStopwords are common English function words plus feed-query "meta" words
// (feeds, articles, stories, recently, …). They are dropped from OR-recall
// passage retrieval so that rare, topical terms dominate bm25 ranking instead
// of being diluted by ubiquitous words: a natural-language question like
// "what stories have my feeds covered recently about X" otherwise OR-matches
// nearly every article through its filler words, burying the relevant ones.
var ftsStopwords = map[string]bool{
	"a": true, "an": true, "the": true, "and": true, "or": true, "but": true,
	"of": true, "to": true, "in": true, "on": true, "at": true, "by": true,
	"for": true, "with": true, "about": true, "from": true, "into": true,
	"over": true, "as": true, "than": true, "then": true, "so": true, "if": true,
	"i": true, "me": true, "my": true, "mine": true, "we": true, "our": true,
	"us": true, "you": true, "your": true, "it": true, "its": true, "they": true,
	"them": true, "their": true, "this": true, "that": true, "these": true,
	"those": true, "any": true, "some": true, "all": true, "more": true, "most": true,
	"what": true, "which": true, "who": true, "whom": true, "whose": true,
	"when": true, "where": true, "why": true, "how": true,
	"is": true, "are": true, "was": true, "were": true, "be": true, "been": true,
	"being": true, "am": true, "do": true, "does": true, "did": true, "doing": true,
	"have": true, "has": true, "had": true, "having": true, "will": true,
	"would": true, "can": true, "could": true, "should": true, "may": true,
	"might": true, "must": true, "not": true, "no": true, "there": true,
	"here": true, "just": true, "only": true, "also": true, "up": true, "out": true,
	// feed-query meta words (rarely the actual topic of a question)
	"feed": true, "feeds": true, "article": true, "articles": true,
	"story": true, "stories": true, "post": true, "posts": true,
	"recently": true, "recent": true, "lately": true,
}

// ftsSentencePunct is trimmed from the ends of each token before the stopword
// check and quoting, so "recently?" matches the stopword "recently" and
// "Anthropic." searches for "Anthropic". Double quotes are intentionally NOT
// trimmed — they are handled by the FTS escaping below.
const ftsSentencePunct = ".,!?;:"

// topicalTerms splits free-text input into the bag of "topical" terms used for
// OR-recall retrieval: each whitespace-separated token has trailing sentence
// punctuation trimmed, and common stopwords (see ftsStopwords) are dropped so
// rare topical terms dominate ranking instead of being diluted by ubiquitous
// filler words. If a query is ENTIRELY stopwords (e.g. "what is that") every
// token is kept so retrieval still runs. Returns the raw (unquoted) terms;
// callers quote them for FTS5 or wrap them for LIKE as needed. The returned
// slice is empty only when there is nothing searchable at all.
//
// Shared by buildFTSMatchOR (FTS5 path) and SearchPassages' LIKE fallback so
// both honour stopwords identically regardless of whether FTS5 is compiled in.
func topicalTerms(search string) []string {
	var kept, all []string
	for _, f := range strings.Fields(search) {
		f = strings.Trim(f, ftsSentencePunct)
		if f == "" {
			continue
		}
		all = append(all, f)
		if !ftsStopwords[strings.ToLower(f)] {
			kept = append(kept, f)
		}
	}
	// Fall back to every token when the query is nothing but stopwords, so a
	// question like "what is that about" still retrieves something.
	if len(kept) == 0 {
		return all
	}
	return kept
}

// buildFTSMatchOR is like buildFTSMatch but OR-joins the tokens instead of
// AND-ing them, trading precision for recall. It is used by passage retrieval
// ("Ask Your Feeds") where surfacing any article mentioning a query term is
// preferable to requiring all terms to co-occur. Stopwords are dropped via
// topicalTerms first. Returns "" when there is nothing searchable.
func buildFTSMatchOR(search string) string {
	terms := topicalTerms(search)
	if len(terms) == 0 {
		return ""
	}
	tokens := make([]string, len(terms))
	for i, t := range terms {
		// Escape embedded double quotes by doubling them, then wrap as a
		// quoted FTS5 string token so operators inside are treated literally.
		tokens[i] = `"` + strings.ReplaceAll(t, `"`, `""`) + `"`
	}
	return strings.Join(tokens, " OR ")
}
