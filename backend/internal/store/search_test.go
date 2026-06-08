package store

import (
	"testing"
	"time"
)

func TestBuildFTSMatchOR(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"empty", "", ""},
		{"whitespace only", "   \t\n ", ""},
		{"single token", "rust", `"rust"`},
		{"OR joins multiple tokens", "rust async runtime", `"rust" OR "async" OR "runtime"`},
		{"escapes embedded quotes", `say "hi"`, `"say" OR """hi"""`},
		{"collapses extra whitespace", "  foo    bar  ", `"foo" OR "bar"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildFTSMatchOR(tt.input)
			if got != tt.want {
				t.Errorf("buildFTSMatchOR(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSearchPassages(t *testing.T) {
	q := setupTestDB(t)
	userID := createTestUser(t, q)
	feed, err := q.CreateFeed(userID, "https://example.com/rss", "Example Feed", "", "", nil)
	if err != nil {
		t.Fatalf("create feed failed: %v", err)
	}

	now := time.Now()
	if err := q.CreateArticle(feed.ID, "guid-rust", "Learning Rust", "https://example.com/rust", "Author",
		"<p>Rust raw</p>", "<p>An article about the Rust programming language and ownership.</p>", "", &now, 100, 1); err != nil {
		t.Fatalf("create article 1 failed: %v", err)
	}
	if err := q.CreateArticle(feed.ID, "guid-cooking", "Cooking Pasta", "https://example.com/pasta", "Author",
		"<p>Pasta raw</p>", "<p>How to boil water and cook spaghetti for dinner.</p>", "", &now, 100, 1); err != nil {
		t.Fatalf("create article 2 failed: %v", err)
	}

	t.Run("matching term returns the article", func(t *testing.T) {
		passages, err := q.SearchPassages(userID, "Rust", 8)
		if err != nil {
			t.Fatalf("SearchPassages failed: %v", err)
		}
		if len(passages) == 0 {
			t.Fatalf("expected at least one passage for 'Rust', got 0")
		}
		found := false
		for _, p := range passages {
			if p.Title == "Learning Rust" {
				found = true
				if p.URL != "https://example.com/rust" {
					t.Errorf("expected URL https://example.com/rust, got %q", p.URL)
				}
				if p.FeedTitle != "Example Feed" {
					t.Errorf("expected FeedTitle 'Example Feed', got %q", p.FeedTitle)
				}
				if p.Excerpt == "" {
					t.Errorf("expected non-empty excerpt")
				}
			}
		}
		if !found {
			t.Errorf("expected 'Learning Rust' in passages, got %+v", passages)
		}
	})

	t.Run("non-matching term returns empty slice", func(t *testing.T) {
		passages, err := q.SearchPassages(userID, "zzzznonexistentterm", 8)
		if err != nil {
			t.Fatalf("SearchPassages failed: %v", err)
		}
		if len(passages) != 0 {
			t.Errorf("expected 0 passages for non-matching term, got %d: %+v", len(passages), passages)
		}
	})

	t.Run("empty query returns empty slice", func(t *testing.T) {
		passages, err := q.SearchPassages(userID, "   ", 8)
		if err != nil {
			t.Fatalf("SearchPassages failed: %v", err)
		}
		if len(passages) != 0 {
			t.Errorf("expected 0 passages for empty query, got %d", len(passages))
		}
	})

	t.Run("scoped to user — other users' articles are excluded", func(t *testing.T) {
		other, err := q.CreateUser("other", "other@example.com", "hash")
		if err != nil {
			t.Fatalf("create other user failed: %v", err)
		}
		otherFeed, err := q.CreateFeed(other.ID, "https://other.com/rss", "Other Feed", "", "", nil)
		if err != nil {
			t.Fatalf("create other feed failed: %v", err)
		}
		if err := q.CreateArticle(otherFeed.ID, "guid-other-rust", "Other Rust", "https://other.com/rust", "Author",
			"", "<p>Another Rust article owned by a different user.</p>", "", &now, 100, 1); err != nil {
			t.Fatalf("create other article failed: %v", err)
		}

		passages, err := q.SearchPassages(userID, "Rust", 8)
		if err != nil {
			t.Fatalf("SearchPassages failed: %v", err)
		}
		for _, p := range passages {
			if p.Title == "Other Rust" {
				t.Errorf("user-scoped search leaked another user's article: %+v", p)
			}
		}
	})
}
