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

func seedList(t *testing.T, repo *MailingListRepository, name string) string {
	t.Helper()
	list, err := repo.CreateList(context.Background(), name)
	if err != nil {
		t.Fatalf("seed list %q: %v", name, err)
	}
	return list.Name
}

func TestAddUser(t *testing.T) {
	repo := newTestRepo(t)
	listName := seedList(t, repo, "weekly")

	user, err := repo.AddUser(context.Background(), listName, "Alice", "alice@example.com", "tok-alice")
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
	if user.UnsubscribeToken != "tok-alice" {
		t.Errorf("expected unsubscribe token %q, got %q", "tok-alice", user.UnsubscribeToken)
	}
}

func TestAddUser_DuplicateEmailErrors(t *testing.T) {
	repo := newTestRepo(t)
	listName := seedList(t, repo, "weekly")
	repo.AddUser(context.Background(), listName, "Alice", "alice@example.com", "tok-alice")

	_, err := repo.AddUser(context.Background(), listName, "Alice2", "alice@example.com", "tok-alice-2")
	if err == nil {
		t.Fatal("expected error for duplicate email on same list, got nil")
	}
}

func TestConfirmUser(t *testing.T) {
	repo := newTestRepo(t)
	listName := seedList(t, repo, "weekly")
	user, _ := repo.AddUser(context.Background(), listName, "Alice", "alice@example.com", "tok-alice")

	if err := repo.ConfirmUser(context.Background(), user.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, _ := repo.GetUserByUnsubscribeToken(context.Background(), "tok-alice")
	if got.ConfirmedAt == nil {
		t.Error("expected ConfirmedAt to be set after confirmation")
	}
}

func TestConfirmUser_AlreadyConfirmedErrors(t *testing.T) {
	repo := newTestRepo(t)
	listName := seedList(t, repo, "weekly")
	user, _ := repo.AddUser(context.Background(), listName, "Alice", "alice@example.com", "tok-alice")
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

func TestGetUserByUnsubscribeToken(t *testing.T) {
	repo := newTestRepo(t)
	listName := seedList(t, repo, "weekly")
	repo.AddUser(context.Background(), listName, "Bob", "bob@example.com", "tok-bob")

	got, err := repo.GetUserByUnsubscribeToken(context.Background(), "tok-bob")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Email != "bob@example.com" {
		t.Errorf("expected email %q, got %q", "bob@example.com", got.Email)
	}
	if got.UnsubscribeToken != "tok-bob" {
		t.Errorf("expected token %q, got %q", "tok-bob", got.UnsubscribeToken)
	}
}

func TestGetUserByUnsubscribeToken_NotFound(t *testing.T) {
	repo := newTestRepo(t)
	_, err := repo.GetUserByUnsubscribeToken(context.Background(), "no-such-token")
	if err == nil {
		t.Fatal("expected error for unknown token, got nil")
	}
}

func TestGetConfirmedUsers(t *testing.T) {
	repo := newTestRepo(t)
	listName := seedList(t, repo, "weekly")

	confirmed, _ := repo.AddUser(context.Background(), listName, "Alice", "alice@example.com", "tok-alice")
	repo.AddUser(context.Background(), listName, "Bob", "bob@example.com", "tok-bob")
	repo.ConfirmUser(context.Background(), confirmed.ID)

	users, err := repo.GetConfirmedUsers(context.Background(), listName)
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

	userA, _ := repo.AddUser(context.Background(), listA, "Alice", "alice@example.com", "tok-alice")
	userB, _ := repo.AddUser(context.Background(), listB, "Bob", "bob@example.com", "tok-bob")
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
	listName := seedList(t, repo, "weekly")
	user, _ := repo.AddUser(context.Background(), listName, "Alice", "alice@example.com", "tok-alice")

	if err := repo.RemoveUser(context.Background(), user.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err := repo.GetUserByUnsubscribeToken(context.Background(), "tok-alice")
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

func TestUpdateList(t *testing.T) {
	repo := newTestRepo(t)
	repo.CreateList(context.Background(), "original")

	updated, err := repo.RenameList(context.Background(), "original", "renamed")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Name != "renamed" {
		t.Errorf("expected %q, got %q", "renamed", updated.Name)
	}
}

func TestUpdateList_NotFound(t *testing.T) {
	repo := newTestRepo(t)
	_, err := repo.RenameList(context.Background(), "ghost", "nope")
	if err == nil {
		t.Fatal("expected error for unknown list, got nil")
	}
}

func TestDeleteList(t *testing.T) {
	repo := newTestRepo(t)
	repo.CreateList(context.Background(), "doomed")

	if err := repo.DeleteList(context.Background(), "doomed"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err := repo.GetListByName(context.Background(), "doomed")
	if err == nil {
		t.Fatal("expected error after deleting list, got nil")
	}
}

func TestDeleteList_NotFound(t *testing.T) {
	repo := newTestRepo(t)
	err := repo.DeleteList(context.Background(), "ghost")
	if err == nil {
		t.Fatal("expected error for unknown list, got nil")
	}
}

func TestGetUsers(t *testing.T) {
	repo := newTestRepo(t)
	list, _ := repo.CreateList(context.Background(), "weekly")
	repo.AddUser(context.Background(), list.Name, "Alice", "a@test.com", "tok-a")
	repo.AddUser(context.Background(), list.Name, "Bob", "b@test.com", "tok-b")

	users, err := repo.GetUsers(context.Background(), list.Name)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(users) != 2 {
		t.Errorf("expected 2 users, got %d", len(users))
	}
}

func TestCreateConfirmation(t *testing.T) {
	repo := newTestRepo(t)
	list, _ := repo.CreateList(context.Background(), "weekly")
	user, _ := repo.AddUser(context.Background(), list.Name, "Alice", "a@test.com", "tok-a")

	conf, err := repo.CreateConfirmation(context.Background(), user.ID, "confirm-tok")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conf.UserID != user.ID || conf.Token != "confirm-tok" {
		t.Errorf("unexpected confirmation: %+v", conf)
	}
}

func TestGetConfirmationByToken(t *testing.T) {
	repo := newTestRepo(t)
	list, _ := repo.CreateList(context.Background(), "weekly")
	user, _ := repo.AddUser(context.Background(), list.Name, "Alice", "a@test.com", "tok-a")
	repo.CreateConfirmation(context.Background(), user.ID, "find-me")

	conf, err := repo.GetConfirmationByToken(context.Background(), "find-me")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conf.Token != "find-me" || conf.UserID != user.ID {
		t.Errorf("unexpected confirmation: %+v", conf)
	}
}

func TestGetConfirmationByToken_NotFound(t *testing.T) {
	repo := newTestRepo(t)
	_, err := repo.GetConfirmationByToken(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown token, got nil")
	}
}

func TestDeleteConfirmation(t *testing.T) {
	repo := newTestRepo(t)
	list, _ := repo.CreateList(context.Background(), "weekly")
	user, _ := repo.AddUser(context.Background(), list.Name, "Alice", "a@test.com", "tok-a")
	conf, _ := repo.CreateConfirmation(context.Background(), user.ID, "del-me")

	if err := repo.DeleteConfirmation(context.Background(), conf.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err := repo.GetConfirmationByToken(context.Background(), "del-me")
	if err == nil {
		t.Fatal("expected error after deleting confirmation, got nil")
	}
}

func TestDeleteConfirmation_NotFound(t *testing.T) {
	repo := newTestRepo(t)
	err := repo.DeleteConfirmation(context.Background(), 9999)
	if err == nil {
		t.Fatal("expected error for unknown confirmation, got nil")
	}
}
