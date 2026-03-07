package fetcher

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchFeed(t *testing.T) {
	rssXML := `<?xml version="1.0"?>
<rss version="2.0">
  <channel>
    <title>Test Feed</title>
    <link>https://example.com</link>
    <item>
      <title>Article 1</title>
      <link>https://example.com/1</link>
      <guid>guid-1</guid>
      <description>Content 1</description>
    </item>
    <item>
      <title>Article 2</title>
      <link>https://example.com/2</link>
      <guid>guid-2</guid>
      <description>Content 2</description>
    </item>
  </channel>
</rss>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(rssXML))
	}))
	defer server.Close()

	result, err := FetchFeed(server.URL)
	if err != nil {
		t.Fatalf("fetch failed: %v", err)
	}
	if result.Title != "Test Feed" {
		t.Errorf("expected 'Test Feed', got %q", result.Title)
	}
	if len(result.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result.Items))
	}
	if result.Items[0].Title != "Article 1" {
		t.Errorf("expected 'Article 1', got %q", result.Items[0].Title)
	}
}

func TestCountWords(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"hello world", 2},
		{"<p>hello <b>world</b></p>", 2},
		{"", 0},
		{"one two three four five", 5},
	}

	for _, tt := range tests {
		got := countWords(tt.input)
		if got != tt.expected {
			t.Errorf("countWords(%q) = %d, want %d", tt.input, got, tt.expected)
		}
	}
}
