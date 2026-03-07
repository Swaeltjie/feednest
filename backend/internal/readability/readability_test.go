package readability

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExtractContent(t *testing.T) {
	html := `<html><head><title>Test</title></head><body>
		<nav>Navigation links here</nav>
		<article>
			<h1>Hello World</h1>
			<p>This is the main content of the article. It has enough text to be extracted properly by the readability algorithm. We need quite a bit of text here for the algorithm to properly identify this as the main content block. The readability algorithm looks for substantial text blocks and tries to filter out navigation, headers, footers, and other non-content elements.</p>
			<p>Here is another paragraph with more content to make sure the algorithm has enough to work with. Articles typically have multiple paragraphs of text that form the body of the content.</p>
		</article>
		<footer>Footer content</footer>
	</body></html>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(html))
	}))
	defer server.Close()

	content, err := ExtractContent(server.URL)
	if err != nil {
		t.Fatalf("extract failed: %v", err)
	}
	if content == "" {
		t.Error("expected non-empty content")
	}
}

func TestExtractThumbnail(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		expected string
	}{
		{
			name:     "og:image property first",
			html:     `<meta property="og:image" content="https://example.com/image.jpg">`,
			expected: "https://example.com/image.jpg",
		},
		{
			name:     "og:image content first",
			html:     `<meta content="https://example.com/image2.jpg" property="og:image">`,
			expected: "https://example.com/image2.jpg",
		},
		{
			name:     "no og:image",
			html:     `<meta property="og:title" content="Title">`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractThumbnailFromHTML(tt.html)
			if got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}
