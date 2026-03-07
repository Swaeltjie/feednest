package store

import (
	"os"
	"testing"
)

func TestNewDB_CreatesTablesOnInit(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"
	db, err := NewDB(dbPath)
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()
	defer os.Remove(dbPath)

	tables := []string{"users", "feeds", "categories", "articles", "tags", "article_tags", "reading_events", "settings"}
	for _, table := range tables {
		var name string
		err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name)
		if err != nil {
			t.Errorf("table %q not found: %v", table, err)
		}
	}
}
