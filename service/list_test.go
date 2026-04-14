package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/5000K/5000mails/domain"
)

func TestListService_Create(t *testing.T) {
	t.Run("returns new list on success", func(t *testing.T) {
		svc := NewListService(newFakeListRepo(), newFakeUserRepo())
		list, err := svc.Create(context.Background(), "newsletter")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if list.Name != "newsletter" {
			t.Errorf("expected name %q, got %q", "newsletter", list.Name)
		}
	})

	t.Run("wraps repo error", func(t *testing.T) {
		repo := newFakeListRepo()
		repo.createErr = errors.New("db failure")
		svc := NewListService(repo, newFakeUserRepo())
		_, err := svc.Create(context.Background(), "newsletter")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, repo.createErr) {
			t.Errorf("expected error to wrap repo error, got: %v", err)
		}
	})
}

func TestListService_Get(t *testing.T) {
	list := &domain.MailingList{Name: "weekly"}

	t.Run("returns list by name", func(t *testing.T) {
		svc := NewListService(newFakeListRepo(list), newFakeUserRepo())
		got, err := svc.Get(context.Background(), "weekly")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Name != list.Name {
			t.Errorf("got %+v, want %+v", got, list)
		}
	})

	t.Run("wraps repo error", func(t *testing.T) {
		repo := newFakeListRepo()
		repo.getByNameErr = errors.New("not found")
		svc := NewListService(repo, newFakeUserRepo())
		_, err := svc.Get(context.Background(), "ghost")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, repo.getByNameErr) {
			t.Errorf("expected wrapped repo error, got: %v", err)
		}
	})
}

func TestListService_GetByName(t *testing.T) {
	list := &domain.MailingList{Name: "monthly"}

	t.Run("returns list by name", func(t *testing.T) {
		svc := NewListService(newFakeListRepo(list), newFakeUserRepo())
		got, err := svc.Get(context.Background(), "monthly")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Name != "monthly" {
			t.Errorf("expected name %q, got %q", "monthly", got.Name)
		}
	})

	t.Run("wraps repo error for unknown name", func(t *testing.T) {
		repo := newFakeListRepo()
		repo.getByNameErr = errors.New("not found")
		svc := NewListService(repo, newFakeUserRepo())
		_, err := svc.Get(context.Background(), "ghost")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, repo.getByNameErr) {
			t.Errorf("expected wrapped repo error, got: %v", err)
		}
	})
}

func TestListService_Rename(t *testing.T) {
	list := &domain.MailingList{Name: "old-name"}

	t.Run("updates list name", func(t *testing.T) {
		svc := NewListService(newFakeListRepo(list), newFakeUserRepo())
		got, err := svc.Rename(context.Background(), "old-name", "new-name")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Name != "new-name" {
			t.Errorf("expected %q, got %q", "new-name", got.Name)
		}
	})

	t.Run("wraps repo error", func(t *testing.T) {
		repo := newFakeListRepo()
		repo.updateErr = errors.New("update failed")
		svc := NewListService(repo, newFakeUserRepo())
		_, err := svc.Rename(context.Background(), "old-name", "new-name")
		if !errors.Is(err, repo.updateErr) {
			t.Errorf("expected wrapped repo error, got: %v", err)
		}
	})
}

func TestListService_Delete(t *testing.T) {
	list := &domain.MailingList{Name: "doomed"}

	t.Run("deletes list", func(t *testing.T) {
		repo := newFakeListRepo(list)
		svc := NewListService(repo, newFakeUserRepo())
		if err := svc.Delete(context.Background(), "doomed"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, exists := repo.lists["doomed"]; exists {
			t.Error("expected list to be deleted")
		}
	})

	t.Run("wraps repo error", func(t *testing.T) {
		repo := newFakeListRepo()
		repo.deleteErr = errors.New("delete failed")
		svc := NewListService(repo, newFakeUserRepo())
		err := svc.Delete(context.Background(), "doomed")
		if !errors.Is(err, repo.deleteErr) {
			t.Errorf("expected wrapped repo error, got: %v", err)
		}
	})
}

func TestListService_CountUsers(t *testing.T) {
	now := time.Now()
	users := []*domain.User{
		{ID: 1, MailingListName: "weekly", Email: "a@example.com", ConfirmedAt: &now},
		{ID: 2, MailingListName: "weekly", Email: "b@example.com", ConfirmedAt: nil},
		{ID: 3, MailingListName: "weekly", Email: "c@example.com", ConfirmedAt: &now},
	}

	t.Run("counts total and confirmed users", func(t *testing.T) {
		svc := NewListService(newFakeListRepo(), newFakeUserRepo(users...))
		counts, err := svc.CountUsers(context.Background(), "weekly")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if counts.Total != 3 {
			t.Errorf("expected Total=3, got %d", counts.Total)
		}
		if counts.Confirmed != 2 {
			t.Errorf("expected Confirmed=2, got %d", counts.Confirmed)
		}
	})

	t.Run("wraps GetUsers error", func(t *testing.T) {
		repo := newFakeUserRepo()
		repo.getUsersErr = errors.New("db down")
		svc := NewListService(newFakeListRepo(), repo)
		_, err := svc.CountUsers(context.Background(), "weekly")
		if !errors.Is(err, repo.getUsersErr) {
			t.Errorf("expected wrapped error, got: %v", err)
		}
	})

	t.Run("wraps GetConfirmedUsers error", func(t *testing.T) {
		repo := newFakeUserRepo(users...)
		repo.getConfirmedErr = errors.New("confirmed query failed")
		svc := NewListService(newFakeListRepo(), repo)
		_, err := svc.CountUsers(context.Background(), "weekly")
		if !errors.Is(err, repo.getConfirmedErr) {
			t.Errorf("expected wrapped error, got: %v", err)
		}
	})
}
