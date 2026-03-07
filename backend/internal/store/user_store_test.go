package store

import (
	"testing"
)

func setupTestDB(t *testing.T) *Queries {
	t.Helper()
	dbPath := t.TempDir() + "/test.db"
	db, err := NewDB(dbPath)
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return New(db)
}

func TestUserStore_CreateAndGet(t *testing.T) {
	q := setupTestDB(t)

	user, err := q.CreateUser("testuser", "test@example.com", "hashedpassword")
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	if user.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if user.Username != "testuser" {
		t.Errorf("expected username 'testuser', got %q", user.Username)
	}

	got, err := q.GetUserByUsername("testuser")
	if err != nil {
		t.Fatalf("failed to get user: %v", err)
	}
	if got.ID != user.ID {
		t.Errorf("expected ID %d, got %d", user.ID, got.ID)
	}
}

func TestUserStore_GetUserByUsername_NotFound(t *testing.T) {
	q := setupTestDB(t)

	_, err := q.GetUserByUsername("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent user")
	}
}

func TestUserStore_UserCount(t *testing.T) {
	q := setupTestDB(t)

	count, err := q.UserCount()
	if err != nil {
		t.Fatalf("failed to count users: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 users, got %d", count)
	}

	q.CreateUser("user1", "u1@test.com", "hash")
	count, _ = q.UserCount()
	if count != 1 {
		t.Errorf("expected 1 user, got %d", count)
	}
}
