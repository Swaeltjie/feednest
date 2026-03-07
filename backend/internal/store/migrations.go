package store

import "database/sql"

func runMigrations(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		email TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS categories (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		name TEXT NOT NULL,
		position INTEGER DEFAULT 0,
		UNIQUE(user_id, name)
	);

	CREATE TABLE IF NOT EXISTS feeds (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		url TEXT NOT NULL,
		title TEXT,
		site_url TEXT,
		icon_url TEXT,
		category_id INTEGER REFERENCES categories(id) ON DELETE SET NULL,
		fetch_interval INTEGER DEFAULT 900,
		last_fetched DATETIME,
		engagement_score REAL DEFAULT 0.0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(user_id, url)
	);

	CREATE TABLE IF NOT EXISTS articles (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		feed_id INTEGER NOT NULL REFERENCES feeds(id) ON DELETE CASCADE,
		guid TEXT NOT NULL,
		title TEXT,
		url TEXT,
		author TEXT,
		content_raw TEXT,
		content_clean TEXT,
		thumbnail_url TEXT,
		published_at DATETIME,
		fetched_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		word_count INTEGER DEFAULT 0,
		reading_time INTEGER DEFAULT 0,
		is_read BOOLEAN DEFAULT 0,
		is_starred BOOLEAN DEFAULT 0,
		read_at DATETIME,
		score REAL DEFAULT 0.0,
		UNIQUE(feed_id, guid)
	);

	CREATE TABLE IF NOT EXISTS tags (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		name TEXT NOT NULL,
		UNIQUE(user_id, name)
	);

	CREATE TABLE IF NOT EXISTS article_tags (
		article_id INTEGER NOT NULL REFERENCES articles(id) ON DELETE CASCADE,
		tag_id INTEGER NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
		PRIMARY KEY (article_id, tag_id)
	);

	CREATE TABLE IF NOT EXISTS reading_events (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		article_id INTEGER NOT NULL REFERENCES articles(id) ON DELETE CASCADE,
		event_type TEXT NOT NULL,
		duration_seconds INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS settings (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		key TEXT NOT NULL,
		value TEXT,
		UNIQUE(user_id, key)
	);

	CREATE INDEX IF NOT EXISTS idx_articles_feed_id ON articles(feed_id);
	CREATE INDEX IF NOT EXISTS idx_articles_published_at ON articles(published_at);
	CREATE INDEX IF NOT EXISTS idx_articles_score ON articles(score);
	CREATE INDEX IF NOT EXISTS idx_articles_is_read ON articles(is_read);
	CREATE INDEX IF NOT EXISTS idx_feeds_user_id ON feeds(user_id);
	CREATE INDEX IF NOT EXISTS idx_reading_events_article_id ON reading_events(article_id);
	`

	_, err := db.Exec(schema)
	return err
}
