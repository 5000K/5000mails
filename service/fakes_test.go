package service

import (
	"context"
	"fmt"
	"time"

	"github.com/5000K/5000mails/domain"
)

// fakeListRepo is an in-memory MailingListRepository.
type fakeListRepo struct {
	lists  map[uint]*domain.MailingList
	nextID uint

	createErr    error
	getErr       error
	getByNameErr error
	updateErr    error
	deleteErr    error
}

func newFakeListRepo(seed ...*domain.MailingList) *fakeListRepo {
	r := &fakeListRepo{lists: make(map[uint]*domain.MailingList), nextID: 1}
	for _, l := range seed {
		r.lists[l.ID] = l
		if l.ID >= r.nextID {
			r.nextID = l.ID + 1
		}
	}
	return r
}

func (r *fakeListRepo) CreateList(_ context.Context, name string) (*domain.MailingList, error) {
	if r.createErr != nil {
		return nil, r.createErr
	}
	l := &domain.MailingList{ID: r.nextID, Name: name}
	r.nextID++
	r.lists[l.ID] = l
	return l, nil
}

func (r *fakeListRepo) GetList(_ context.Context, id uint) (*domain.MailingList, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	l, ok := r.lists[id]
	if !ok {
		return nil, fmt.Errorf("list %d not found", id)
	}
	return l, nil
}

func (r *fakeListRepo) GetListByName(_ context.Context, name string) (*domain.MailingList, error) {
	if r.getByNameErr != nil {
		return nil, r.getByNameErr
	}
	for _, l := range r.lists {
		if l.Name == name {
			return l, nil
		}
	}
	return nil, fmt.Errorf("list %q not found", name)
}

func (r *fakeListRepo) UpdateList(_ context.Context, id uint, name string) (*domain.MailingList, error) {
	if r.updateErr != nil {
		return nil, r.updateErr
	}
	l, ok := r.lists[id]
	if !ok {
		return nil, fmt.Errorf("list %d not found", id)
	}
	l.Name = name
	return l, nil
}

func (r *fakeListRepo) DeleteList(_ context.Context, id uint) error {
	if r.deleteErr != nil {
		return r.deleteErr
	}
	if _, ok := r.lists[id]; !ok {
		return fmt.Errorf("list %d not found", id)
	}
	delete(r.lists, id)
	return nil
}

// fakeUserRepo is an in-memory UserRepository.
type fakeUserRepo struct {
	users  map[uint]*domain.User
	nextID uint

	addErr          error
	confirmErr      error
	getByEmailErr   error
	getUsersErr     error
	getConfirmedErr error
	removeErr       error
}

func newFakeUserRepo(seed ...*domain.User) *fakeUserRepo {
	r := &fakeUserRepo{users: make(map[uint]*domain.User), nextID: 1}
	for _, u := range seed {
		r.users[u.ID] = u
		if u.ID >= r.nextID {
			r.nextID = u.ID + 1
		}
	}
	return r
}

func (r *fakeUserRepo) AddUser(_ context.Context, mailingListID uint, name, email string) (*domain.User, error) {
	if r.addErr != nil {
		return nil, r.addErr
	}
	u := &domain.User{ID: r.nextID, Name: name, Email: email, MailingListID: mailingListID}
	r.nextID++
	r.users[u.ID] = u
	return u, nil
}

func (r *fakeUserRepo) ConfirmUser(_ context.Context, userID uint) error {
	if r.confirmErr != nil {
		return r.confirmErr
	}
	u, ok := r.users[userID]
	if !ok {
		return fmt.Errorf("user %d not found", userID)
	}
	now := time.Now()
	u.ConfirmedAt = &now
	return nil
}

func (r *fakeUserRepo) GetUserByEmail(_ context.Context, email string) (*domain.User, error) {
	if r.getByEmailErr != nil {
		return nil, r.getByEmailErr
	}
	for _, u := range r.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, fmt.Errorf("user %q not found", email)
}

func (r *fakeUserRepo) GetUsers(_ context.Context, mailingListID uint) ([]domain.User, error) {
	if r.getUsersErr != nil {
		return nil, r.getUsersErr
	}
	var out []domain.User
	for _, u := range r.users {
		if u.MailingListID == mailingListID {
			out = append(out, *u)
		}
	}
	return out, nil
}

func (r *fakeUserRepo) GetConfirmedUsers(_ context.Context, mailingListID uint) ([]domain.User, error) {
	if r.getConfirmedErr != nil {
		return nil, r.getConfirmedErr
	}
	var out []domain.User
	for _, u := range r.users {
		if u.MailingListID == mailingListID && u.ConfirmedAt != nil {
			out = append(out, *u)
		}
	}
	return out, nil
}

func (r *fakeUserRepo) RemoveUser(_ context.Context, userID uint) error {
	if r.removeErr != nil {
		return r.removeErr
	}
	if _, ok := r.users[userID]; !ok {
		return fmt.Errorf("user %d not found", userID)
	}
	delete(r.users, userID)
	return nil
}

// fakeConfirmationRepo is an in-memory ConfirmationRepository.
type fakeConfirmationRepo struct {
	confirmations map[uint]*domain.Confirmation
	nextID        uint

	createErr error
	getErr    error
	deleteErr error
}

func newFakeConfirmationRepo(seed ...*domain.Confirmation) *fakeConfirmationRepo {
	r := &fakeConfirmationRepo{confirmations: make(map[uint]*domain.Confirmation), nextID: 1}
	for _, c := range seed {
		r.confirmations[c.ID] = c
		if c.ID >= r.nextID {
			r.nextID = c.ID + 1
		}
	}
	return r
}

func (r *fakeConfirmationRepo) CreateConfirmation(_ context.Context, userID uint, token string) (*domain.Confirmation, error) {
	if r.createErr != nil {
		return nil, r.createErr
	}
	c := &domain.Confirmation{ID: r.nextID, UserID: userID, Token: token}
	r.nextID++
	r.confirmations[c.ID] = c
	return c, nil
}

func (r *fakeConfirmationRepo) GetConfirmationByToken(_ context.Context, token string) (*domain.Confirmation, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	for _, c := range r.confirmations {
		if c.Token == token {
			return c, nil
		}
	}
	return nil, fmt.Errorf("confirmation token not found")
}

func (r *fakeConfirmationRepo) DeleteConfirmation(_ context.Context, id uint) error {
	if r.deleteErr != nil {
		return r.deleteErr
	}
	if _, ok := r.confirmations[id]; !ok {
		return fmt.Errorf("confirmation %d not found", id)
	}
	delete(r.confirmations, id)
	return nil
}

// fakeSender records SendMail calls.
type fakeSender struct {
	calls []sendCall
	err   error
}

type sendCall struct {
	metadata   domain.MailMetadata
	body       string
	recipients []domain.User
}

func (s *fakeSender) SendMail(_ context.Context, metadata domain.MailMetadata, body string, recipients []domain.User) error {
	if s.err != nil {
		return s.err
	}
	s.calls = append(s.calls, sendCall{metadata: metadata, body: body, recipients: recipients})
	return nil
}

// fakeRenderer returns configurable metadata / body.
type fakeRenderer struct {
	metadata domain.MailMetadata
	body     string
	err      error
}

func (r *fakeRenderer) Render(_ *string, _ map[string]any) (domain.MailMetadata, string, error) {
	return r.metadata, r.body, r.err
}
