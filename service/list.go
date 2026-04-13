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

// Create creates a new mailing list with the given name.
func (s *ListService) Create(ctx context.Context, name string) (*domain.MailingList, error) {
	list, err := s.lists.CreateList(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("creating list %q: %w", name, err)
	}
	return list, nil
}

// Get returns a mailing list by its ID.
func (s *ListService) Get(ctx context.Context, id uint) (*domain.MailingList, error) {
	list, err := s.lists.GetList(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting list %d: %w", id, err)
	}
	return list, nil
}

// GetByName returns a mailing list by its name.
func (s *ListService) GetByName(ctx context.Context, name string) (*domain.MailingList, error) {
	list, err := s.lists.GetListByName(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("getting list %q: %w", name, err)
	}
	return list, nil
}

// Rename renames a mailing list.
func (s *ListService) Rename(ctx context.Context, id uint, newName string) (*domain.MailingList, error) {
	list, err := s.lists.UpdateList(ctx, id, newName)
	if err != nil {
		return nil, fmt.Errorf("renaming list %d: %w", id, err)
	}
	return list, nil
}

// Delete deletes a mailing list by its ID.
func (s *ListService) Delete(ctx context.Context, id uint) error {
	if err := s.lists.DeleteList(ctx, id); err != nil {
		return fmt.Errorf("deleting list %d: %w", id, err)
	}
	return nil
}

// UserCounts holds the total and confirmed subscriber counts for a mailing list.
type UserCounts struct {
	Total     int
	Confirmed int
}

// CountUsers returns the total and confirmed subscriber counts for a mailing list.
func (s *ListService) CountUsers(ctx context.Context, listID uint) (UserCounts, error) {
	all, err := s.users.GetUsers(ctx, listID)
	if err != nil {
		return UserCounts{}, fmt.Errorf("getting users for list %d: %w", listID, err)
	}

	confirmed, err := s.users.GetConfirmedUsers(ctx, listID)
	if err != nil {
		return UserCounts{}, fmt.Errorf("getting confirmed users for list %d: %w", listID, err)
	}

	return UserCounts{
		Total:     len(all),
		Confirmed: len(confirmed),
	}, nil
}
