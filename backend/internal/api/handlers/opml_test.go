package handlers

import (
	"strings"
	"testing"
)

func TestParseOPML(t *testing.T) {
	opml := `<?xml version="1.0" encoding="UTF-8"?>
<opml version="2.0">
  <body>
    <outline text="Tech" title="Tech">
      <outline type="rss" text="Ars Technica" xmlUrl="https://feeds.arstechnica.com/arstechnica/features" htmlUrl="https://arstechnica.com"/>
      <outline type="rss" text="Hacker News" xmlUrl="https://news.ycombinator.com/rss"/>
    </outline>
    <outline type="rss" text="Uncategorized Feed" xmlUrl="https://example.com/rss"/>
  </body>
</opml>`

	feeds, err := parseOPML(strings.NewReader(opml))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(feeds) != 3 {
		t.Fatalf("expected 3 feeds, got %d", len(feeds))
	}
	if feeds[0].Category != "Tech" {
		t.Errorf("expected category 'Tech', got %q", feeds[0].Category)
	}
	if feeds[2].Category != "" {
		t.Errorf("expected empty category, got %q", feeds[2].Category)
	}
}

func TestGenerateOPML(t *testing.T) {
	feeds := []opmlFeed{
		{Title: "Ars", XMLURL: "https://feeds.ars.com/rss", HTMLURL: "https://ars.com", Category: "Tech"},
		{Title: "BBC", XMLURL: "https://bbc.com/rss", HTMLURL: "https://bbc.com", Category: "News"},
	}

	output, err := generateOPML(feeds)
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	if !strings.Contains(output, "Ars") {
		t.Error("output should contain feed title")
	}
	if !strings.Contains(output, "Tech") {
		t.Error("output should contain category name")
	}
}
