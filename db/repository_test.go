package db

import (
	"context"
	"log/slog"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newTestRepo(t *testing.T) *MailingListRepository {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open in-memory db: %v", err)
	}
	if err := db.AutoMigrate(&MailingList{}, &User{}, &Confirmation{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return NewMailingListRepository(db, slog.Default())
}

// ---------- MailingList ----------

func TestCreateList(t *testing.T) {
	repo := newTestRepo(t)
	list, err := repo.CreateList(context.Background(), "weekly")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if list.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if list.Name != "weekly" {
		t.Errorf("expected name %q, got %q", "weekly", list.Name)
	}
}

func TestCreateList_DuplicateNameErrors(t *testing.T) {
	repo := newTestRepo(t)
	if _, err := repo.CreateList(context.Background(), "weekly"); err != nil {
		t.Fatalf("first create: %v", err)
	}
	_, err := repo.CreateList(context.Background(), "weekly")
	if err == nil {
		t.Fatal("expected error for duplicate name, got nil")
	}
}

func TestGetList(t *testing.T) {
	repo := newTestRepo(t)
	created, _ := repo.CreateList(context.Background(), "monthly")

	got, err := repo.GetList(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != created.ID || got.Name != created.Name {
		t.Errorf("got %+v, want %+v", got, created)
	}
}

func TestGetList_NotFound(t *testing.T) {
	repo := newTestRepo(t)
	_, err := repo.GetList(context.Background(), 9999)
	if err == nil {
		t.Fatal("expected error for unknown ID, got nil")
	}
}

func TestGetListByName(t *testing.T) {
	repo := newTestRepo(t)
	repo.CreateList(context.Background(), "daily")

	got, err := repo.GetListByName(context.Background(), "daily")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "daily" {
		t.Errorf("expected name %q, got %q", "daily", got.Name)
	}
}

func TestGetListByName_NotFound(t *testing.T) {
	repo := newTestRepo(t)
	_, err := repo.GetListByName(context.Background(), "ghost")
	if err == nil {
		t.Fatal("expected error for unknown name, got nil")
	}
}

// ---------- User ----------

func seedList(t *testing.T, repo *MailingListRepository, name string) uint {
	t.Helper()
	list, err := repo.CreateList(context.Background(), name)
	if err != nil {
		t.Fatalf("seed list %q: %v", name, err)
	}
	return list.ID
}

func TestAddUser(t *testing.T) {
	repo := newTestRepo(t)
	listID := seedList(t, repo, "weekly")

	user, err := repo.AddUser(context.Background(), listID, "Alice", "alice@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.ID == 0 {
		t.Error("expected non-zero user ID")
	}
	if user.Email != "alice@example.com" {
		t.Errorf("expected email %q, got %q", "alice@example.com", user.Email)
	}
	if user.ConfirmedAt != nil {
		t.Error("new user should be unconfirmed")
	}
}

func TestAddUser_DuplicateEmailErrors(t *testing.T) {
	repo := newTestRepo(t)
	listID := seedList(t, repo, "weekly")
	repo.AddUser(context.Background(), listID, "Alice", "alice@example.com")

	_, err := repo.AddUser(context.Background(), listID, "Alice2", "alice@example.com")
	if err == nil {
		t.Fatal("expected error for duplicate email, got nil")
	}
}

func TestConfirmUser(t *testing.T) {
	repo := newTestRepo(t)
	listID := seedList(t, repo, "weekly")
	user, _ := repo.AddUser(context.Background(), listID, "Alice", "alice@example.com")

	if err := repo.ConfirmUser(context.Background(), user.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, _ := repo.GetUserByEmail(context.Background(), "alice@example.com")
	if got.ConfirmedAt == nil {
		t.Error("expected ConfirmedAt to be set after confirmation")
	}
}

func TestConfirmUser_AlreadyConfirmedErrors(t *testing.T) {
	repo := newTestRepo(t)
	listID := seedList(t, repo, "weekly")
	user, _ := repo.AddUser(context.Background(), listID, "Alice", "alice@example.com")
	repo.ConfirmUser(context.Background(), user.ID)

	err := repo.ConfirmUser(context.Background(), user.ID)
	if err == nil {
		t.Fatal("expected error when confirming already-confirmed user, got nil")
	}
}

func TestConfirmUser_NotFoundErrors(t *testing.T) {
	repo := newTestRepo(t)
	err := repo.ConfirmUser(context.Background(), 9999)
	if err == nil {
		t.Fatal("expected error for unknown user, got nil")
	}
}

func TestGetUserByEmail(t *testing.T) {
	repo := newTestRepo(t)
	listID := seedList(t, repo, "weekly")
	repo.AddUser(context.Background(), listID, "Bob", "bob@example.com")

	got, err := repo.GetUserByEmail(context.Background(), "bob@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Email != "bob@example.com" {
		t.Errorf("expected email %q, got %q", "bob@example.com", got.Email)
	}
}

func TestGetUserByEmail_NotFound(t *testing.T) {
	repo := newTestRepo(t)
	_, err := repo.GetUserByEmail(context.Background(), "nobody@example.com")
	if err == nil {
		t.Fatal("expected error for unknown email, got nil")
	}
}

func TestGetConfirmedUsers(t *testing.T) {
	repo := newTestRepo(t)
	listID := seedList(t, repo, "weekly")

	confirmed, _ := repo.AddUser(context.Background(), listID, "Alice", "alice@example.com")
	repo.AddUser(context.Background(), listID, "Bob", "bob@example.com")
	repo.ConfirmUser(context.Background(), confirmed.ID)

	users, err := repo.GetConfirmedUsers(context.Background(), listID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(users) != 1 {
		t.Fatalf("expected 1 confirmed user, got %d", len(users))
	}
	if users[0].Email != "alice@example.com" {
		t.Errorf("expected alice, got %q", users[0].Email)
	}
}

func TestGetConfirmedUsers_ExcludesOtherLists(t *testing.T) {
	repo := newTestRepo(t)
	listA := seedList(t, repo, "list-a")
	listB := seedList(t, repo, "list-b")

	userA, _ := repo.AddUser(context.Background(), listA, "Alice", "alice@example.com")
	userB, _ := repo.AddUser(context.Background(), listB, "Bob", "bob@example.com")
	repo.ConfirmUser(context.Background(), userA.ID)
	repo.ConfirmUser(context.Background(), userB.ID)

	users, err := repo.GetConfirmedUsers(context.Background(), listA)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(users) != 1 || users[0].Email != "alice@example.com" {
		t.Errorf("expected only alice, got %+v", users)
	}
}

func TestRemoveUser(t *testing.T) {
	repo := newTestRepo(t)
	listID := seedList(t, repo, "weekly")
	user, _ := repo.AddUser(context.Background(), listID, "Alice", "alice@example.com")

	if err := repo.RemoveUser(context.Background(), user.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err := repo.GetUserByEmail(context.Background(), "alice@example.com")
	if err == nil {
		t.Fatal("expected error after removing user, got nil")
	}
}

func TestRemoveUser_NotFound(t *testing.T) {
	repo := newTestRepo(t)
	err := repo.RemoveUser(context.Background(), 9999)
	if err == nil {
		t.Fatal("expected error for unknown user, got nil")
	}
}
