package store

import "testing"

func createTestUser(t *testing.T, q *Queries) int64 {
	t.Helper()
	user, err := q.CreateUser("testuser", "test@example.com", "hash")
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	return user.ID
}

func TestCategoryStore_CRUD(t *testing.T) {
	q := setupTestDB(t)
	userID := createTestUser(t, q)

	cat, err := q.CreateCategory(userID, "Tech", 0)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if cat.Name != "Tech" {
		t.Errorf("expected 'Tech', got %q", cat.Name)
	}

	cats, err := q.ListCategories(userID)
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(cats) != 1 {
		t.Fatalf("expected 1 category, got %d", len(cats))
	}

	newName := "Technology"
	err = q.UpdateCategory(cat.ID, userID, &newName, nil)
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
	updated, _ := q.ListCategories(userID)
	if updated[0].Name != "Technology" {
		t.Errorf("expected 'Technology', got %q", updated[0].Name)
	}

	err = q.DeleteCategory(cat.ID, userID)
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	remaining, _ := q.ListCategories(userID)
	if len(remaining) != 0 {
		t.Errorf("expected 0 categories after delete, got %d", len(remaining))
	}
}
