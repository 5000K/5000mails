package service

import (
	"context"
	"fmt"

	"github.com/5000K/5000mails/domain"
)

// ListService manages mailing list CRUD and user counts.
type ListService struct {
	lists domain.MailingListRepository
	users domain.UserRepository
}

// NewListService creates a new ListService.
func NewListService(lists domain.MailingListRepository, users domain.UserRepository) *ListService {
	return &ListService{lists: lists, users: users}
}

// All returns all mailing lists.
func (s *ListService) All(ctx context.Context) ([]domain.MailingList, error) {
	lists, err := s.lists.GetAllLists(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing all lists: %w", err)
	}
	return lists, nil
}

// Create creates a new mailing list with the given name.
func (s *ListService) Create(ctx context.Context, name string) (*domain.MailingList, error) {
	list, err := s.lists.CreateList(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("creating list %q: %w", name, err)
	}
	return list, nil
}

// Get returns a mailing list by name.
func (s *ListService) Get(ctx context.Context, name string) (*domain.MailingList, error) {
	list, err := s.lists.GetListByName(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("getting list %q: %w", name, err)
	}
	return list, nil
}

// Rename renames a mailing list.
func (s *ListService) Rename(ctx context.Context, name, newName string) (*domain.MailingList, error) {
	list, err := s.lists.RenameList(ctx, name, newName)
	if err != nil {
		return nil, fmt.Errorf("renaming list %q: %w", name, err)
	}
	return list, nil
}

// Delete deletes a mailing list by name.
func (s *ListService) Delete(ctx context.Context, name string) error {
	if err := s.lists.DeleteList(ctx, name); err != nil {
		return fmt.Errorf("deleting list %q: %w", name, err)
	}
	return nil
}

// CountUsers returns the total and confirmed subscriber counts for a mailing list.
func (s *ListService) CountUsers(ctx context.Context, listName string) (domain.UserCounts, error) {
	all, err := s.users.GetUsers(ctx, listName)
	if err != nil {
		return domain.UserCounts{}, fmt.Errorf("getting users for list %q: %w", listName, err)
	}

	confirmed, err := s.users.GetConfirmedUsers(ctx, listName)
	if err != nil {
		return domain.UserCounts{}, fmt.Errorf("getting confirmed users for list %q: %w", listName, err)
	}

	return domain.UserCounts{
		Total:     len(all),
		Confirmed: len(confirmed),
	}, nil
}

// Users returns all subscribers for a mailing list, confirmed or not.
func (s *ListService) Users(ctx context.Context, listName string) ([]domain.User, error) {
	users, err := s.users.GetUsers(ctx, listName)
	if err != nil {
		return nil, fmt.Errorf("getting users for list %q: %w", listName, err)
	}
	return users, nil
}
