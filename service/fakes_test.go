package service

import (
	"context"
	"fmt"
	"time"

	"github.com/5000K/5000mails/domain"
)

type fakeListRepo struct {
	lists map[string]*domain.MailingList

	createErr    error
	getAllErr    error
	getByNameErr error
	updateErr    error
	deleteErr    error
}

func newFakeListRepo(seed ...*domain.MailingList) *fakeListRepo {
	r := &fakeListRepo{lists: make(map[string]*domain.MailingList)}
	for _, l := range seed {
		r.lists[l.Name] = l
	}
	return r
}

func (r *fakeListRepo) GetAllLists(_ context.Context) ([]domain.MailingList, error) {
	if r.getAllErr != nil {
		return nil, r.getAllErr
	}
	out := make([]domain.MailingList, 0, len(r.lists))
	for _, l := range r.lists {
		out = append(out, *l)
	}
	return out, nil
}

func (r *fakeListRepo) CreateList(_ context.Context, name string) (*domain.MailingList, error) {
	if r.createErr != nil {
		return nil, r.createErr
	}
	l := &domain.MailingList{Name: name}
	r.lists[name] = l
	return l, nil
}

func (r *fakeListRepo) GetListByName(_ context.Context, name string) (*domain.MailingList, error) {
	if r.getByNameErr != nil {
		return nil, r.getByNameErr
	}
	l, ok := r.lists[name]
	if !ok {
		return nil, fmt.Errorf("list %q not found", name)
	}
	return l, nil
}

func (r *fakeListRepo) RenameList(_ context.Context, name, newName string) (*domain.MailingList, error) {
	if r.updateErr != nil {
		return nil, r.updateErr
	}
	l, ok := r.lists[name]
	if !ok {
		return nil, fmt.Errorf("list %q not found", name)
	}
	delete(r.lists, name)
	l.Name = newName
	r.lists[newName] = l
	return l, nil
}

func (r *fakeListRepo) DeleteList(_ context.Context, name string) error {
	if r.deleteErr != nil {
		return r.deleteErr
	}
	if _, ok := r.lists[name]; !ok {
		return fmt.Errorf("list %q not found", name)
	}
	delete(r.lists, name)
	return nil
}

type fakeUserRepo struct {
	users        map[uint]*domain.User
	deletedUsers map[uint]*domain.User
	nextID       uint

	addErr                    error
	confirmErr                error
	getByEmailErr             error
	getUnsubscribedByEmailErr error
	getByUnsubscribeTokenErr  error
	getUsersErr               error
	getConfirmedErr           error
	removeErr                 error
	reactivateErr             error
}

func newFakeUserRepo(seed ...*domain.User) *fakeUserRepo {
	r := &fakeUserRepo{
		users:        make(map[uint]*domain.User),
		deletedUsers: make(map[uint]*domain.User),
		nextID:       1,
	}
	for _, u := range seed {
		r.users[u.ID] = u
		if u.ID >= r.nextID {
			r.nextID = u.ID + 1
		}
	}
	return r
}

func (r *fakeUserRepo) seedDeleted(u *domain.User) {
	r.deletedUsers[u.ID] = u
	if u.ID >= r.nextID {
		r.nextID = u.ID + 1
	}
}

func (r *fakeUserRepo) AddUser(_ context.Context, mailingListName string, name, email, unsubscribeToken string) (*domain.User, error) {
	if r.addErr != nil {
		return nil, r.addErr
	}
	u := &domain.User{ID: r.nextID, Name: name, Email: email, MailingListName: mailingListName, UnsubscribeToken: unsubscribeToken}
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

func (r *fakeUserRepo) GetUserByUnsubscribeToken(_ context.Context, token string) (*domain.User, error) {
	if r.getByUnsubscribeTokenErr != nil {
		return nil, r.getByUnsubscribeTokenErr
	}
	for _, u := range r.users {
		if u.UnsubscribeToken == token {
			return u, nil
		}
	}
	return nil, fmt.Errorf("user with unsubscribe token %q not found", token)
}

func (r *fakeUserRepo) GetUsers(_ context.Context, mailingListName string) ([]domain.User, error) {
	if r.getUsersErr != nil {
		return nil, r.getUsersErr
	}
	var out []domain.User
	for _, u := range r.users {
		if u.MailingListName == mailingListName {
			out = append(out, *u)
		}
	}
	return out, nil
}

func (r *fakeUserRepo) GetConfirmedUsers(_ context.Context, mailingListName string) ([]domain.User, error) {
	if r.getConfirmedErr != nil {
		return nil, r.getConfirmedErr
	}
	var out []domain.User
	for _, u := range r.users {
		if u.MailingListName == mailingListName && u.ConfirmedAt != nil {
			out = append(out, *u)
		}
	}
	return out, nil
}

func (r *fakeUserRepo) GetUserByEmail(_ context.Context, mailingListName, email string) (*domain.User, error) {
	if r.getByEmailErr != nil {
		return nil, r.getByEmailErr
	}
	for _, u := range r.users {
		if u.MailingListName == mailingListName && u.Email == email {
			return u, nil
		}
	}
	return nil, fmt.Errorf("user with email %q in list %q not found", email, mailingListName)
}

func (r *fakeUserRepo) GetUnsubscribedUserByEmail(_ context.Context, mailingListName, email string) (*domain.User, error) {
	if r.getUnsubscribedByEmailErr != nil {
		return nil, r.getUnsubscribedByEmailErr
	}
	for _, u := range r.deletedUsers {
		if u.MailingListName == mailingListName && u.Email == email {
			return u, nil
		}
	}
	return nil, fmt.Errorf("unsubscribed user with email %q in list %q not found", email, mailingListName)
}

func (r *fakeUserRepo) ReactivateUser(_ context.Context, userID uint, name, unsubscribeToken string) (*domain.User, error) {
	if r.reactivateErr != nil {
		return nil, r.reactivateErr
	}
	u, ok := r.deletedUsers[userID]
	if !ok {
		return nil, fmt.Errorf("deleted user %d not found", userID)
	}
	u.Name = name
	u.UnsubscribeToken = unsubscribeToken
	u.ConfirmedAt = nil
	delete(r.deletedUsers, userID)
	r.users[userID] = u
	return u, nil
}

func (r *fakeUserRepo) RemoveUser(_ context.Context, userID uint) error {
	if r.removeErr != nil {
		return r.removeErr
	}
	u, ok := r.users[userID]
	if !ok {
		return fmt.Errorf("user %d not found", userID)
	}
	delete(r.users, userID)
	r.deletedUsers[userID] = u
	return nil
}

type fakeConfirmationRepo struct {
	confirmations map[uint]*domain.Confirmation
	nextID        uint

	createErr         error
	getErr            error
	deleteErr         error
	deleteByUserIDErr error
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

func (r *fakeConfirmationRepo) DeleteConfirmationsByUserID(_ context.Context, userID uint) error {
	if r.deleteByUserIDErr != nil {
		return r.deleteByUserIDErr
	}
	for id, c := range r.confirmations {
		if c.UserID == userID {
			delete(r.confirmations, id)
		}
	}
	return nil
}

type fakeTopicRepo struct {
	topics     map[uint]*domain.Topic
	userTopics map[uint]map[uint]bool
	nextID     uint

	createErr            error
	getByListErr         error
	getByNameErr         error
	updateErr            error
	deleteErr            error
	getDefaultErr        error
	subscribeUserErr     error
	unsubscribeUserErr   error
	setUserTopicsErr     error
	getUserTopicsErr     error
	getConfirmedUsersErr error
	subscribeAllErr      error
}

func newFakeTopicRepo(seed ...*domain.Topic) *fakeTopicRepo {
	r := &fakeTopicRepo{
		topics:     make(map[uint]*domain.Topic),
		userTopics: make(map[uint]map[uint]bool),
		nextID:     1,
	}
	for _, t := range seed {
		if t.ID == 0 {
			t.ID = r.nextID
			r.nextID++
		}
		r.topics[t.ID] = t
		if t.ID >= r.nextID {
			r.nextID = t.ID + 1
		}
	}
	return r
}

func (r *fakeTopicRepo) CreateTopic(_ context.Context, mailingListName, name, displayName string, defaultEnabled bool) (*domain.Topic, error) {
	if r.createErr != nil {
		return nil, r.createErr
	}
	t := &domain.Topic{ID: r.nextID, Name: name, DisplayName: displayName, MailingListName: mailingListName, DefaultEnabled: defaultEnabled}
	r.nextID++
	r.topics[t.ID] = t
	return t, nil
}

func (r *fakeTopicRepo) GetTopicsByList(_ context.Context, mailingListName string) ([]domain.Topic, error) {
	if r.getByListErr != nil {
		return nil, r.getByListErr
	}
	var out []domain.Topic
	for _, t := range r.topics {
		if t.MailingListName == mailingListName {
			out = append(out, *t)
		}
	}
	return out, nil
}

func (r *fakeTopicRepo) GetTopicByName(_ context.Context, mailingListName, name string) (*domain.Topic, error) {
	if r.getByNameErr != nil {
		return nil, r.getByNameErr
	}
	for _, t := range r.topics {
		if t.MailingListName == mailingListName && t.Name == name {
			return t, nil
		}
	}
	return nil, fmt.Errorf("topic %q on list %q not found", name, mailingListName)
}

func (r *fakeTopicRepo) UpdateTopic(_ context.Context, mailingListName, name string, displayName *string, defaultEnabled *bool) (*domain.Topic, error) {
	if r.updateErr != nil {
		return nil, r.updateErr
	}
	for _, t := range r.topics {
		if t.MailingListName == mailingListName && t.Name == name {
			if displayName != nil {
				t.DisplayName = *displayName
			}
			if defaultEnabled != nil {
				t.DefaultEnabled = *defaultEnabled
			}
			return t, nil
		}
	}
	return nil, fmt.Errorf("topic %q on list %q not found", name, mailingListName)
}

func (r *fakeTopicRepo) DeleteTopic(_ context.Context, mailingListName, name string) error {
	if r.deleteErr != nil {
		return r.deleteErr
	}
	for id, t := range r.topics {
		if t.MailingListName == mailingListName && t.Name == name {
			delete(r.topics, id)
			return nil
		}
	}
	return fmt.Errorf("topic %q on list %q not found", name, mailingListName)
}

func (r *fakeTopicRepo) GetDefaultEnabledTopics(_ context.Context, mailingListName string) ([]domain.Topic, error) {
	if r.getDefaultErr != nil {
		return nil, r.getDefaultErr
	}
	var out []domain.Topic
	for _, t := range r.topics {
		if t.MailingListName == mailingListName && t.DefaultEnabled {
			out = append(out, *t)
		}
	}
	return out, nil
}

func (r *fakeTopicRepo) SubscribeUserToTopics(_ context.Context, userID uint, topicIDs []uint) error {
	if r.subscribeUserErr != nil {
		return r.subscribeUserErr
	}
	if r.userTopics[userID] == nil {
		r.userTopics[userID] = make(map[uint]bool)
	}
	for _, id := range topicIDs {
		r.userTopics[userID][id] = true
	}
	return nil
}

func (r *fakeTopicRepo) UnsubscribeUserFromTopics(_ context.Context, userID uint, topicIDs []uint) error {
	if r.unsubscribeUserErr != nil {
		return r.unsubscribeUserErr
	}
	if r.userTopics[userID] == nil {
		return nil
	}
	for _, id := range topicIDs {
		delete(r.userTopics[userID], id)
	}
	return nil
}

func (r *fakeTopicRepo) SetUserTopics(_ context.Context, userID uint, topicIDs []uint) error {
	if r.setUserTopicsErr != nil {
		return r.setUserTopicsErr
	}
	r.userTopics[userID] = make(map[uint]bool)
	for _, id := range topicIDs {
		r.userTopics[userID][id] = true
	}
	return nil
}

func (r *fakeTopicRepo) GetUserTopics(_ context.Context, userID uint) ([]domain.Topic, error) {
	if r.getUserTopicsErr != nil {
		return nil, r.getUserTopicsErr
	}
	var out []domain.Topic
	for id := range r.userTopics[userID] {
		if t, ok := r.topics[id]; ok {
			out = append(out, *t)
		}
	}
	return out, nil
}

func (r *fakeTopicRepo) GetConfirmedUsersSubscribedToTopics(_ context.Context, _ string, _ []string) ([]domain.User, error) {
	if r.getConfirmedUsersErr != nil {
		return nil, r.getConfirmedUsersErr
	}
	return nil, nil
}

func (r *fakeTopicRepo) SubscribeAllUsersToTopic(_ context.Context, _ string, _ uint) error {
	if r.subscribeAllErr != nil {
		return r.subscribeAllErr
	}
	return nil
}

type fakeNewsletterRepo struct {
	newsletters map[uint]*domain.SentNewsletter
	nextID      uint

	createErr error
	getAllErr error
	getErr    error
	deleteErr error
}

func newFakeNewsletterRepo(seed ...*domain.SentNewsletter) *fakeNewsletterRepo {
	r := &fakeNewsletterRepo{newsletters: make(map[uint]*domain.SentNewsletter), nextID: 1}
	for _, n := range seed {
		r.newsletters[n.ID] = n
		if n.ID >= r.nextID {
			r.nextID = n.ID + 1
		}
	}
	return r
}

func (r *fakeNewsletterRepo) CreateSentNewsletter(_ context.Context, subject, senderName, rawMarkdown string, recipientIDs []uint, listNames []string, topicNames []string) (*domain.SentNewsletter, error) {
	if r.createErr != nil {
		return nil, r.createErr
	}
	recipients := make([]domain.User, len(recipientIDs))
	for i, id := range recipientIDs {
		recipients[i] = domain.User{ID: id}
	}
	lists := make([]domain.MailingList, len(listNames))
	for i, name := range listNames {
		lists[i] = domain.MailingList{Name: name}
	}
	topics := make([]domain.Topic, len(topicNames))
	for i, name := range topicNames {
		topics[i] = domain.Topic{Name: name}
	}
	n := &domain.SentNewsletter{
		ID:           r.nextID,
		Subject:      subject,
		SenderName:   senderName,
		RawMarkdown:  rawMarkdown,
		SentAt:       time.Now(),
		Recipients:   recipients,
		MailingLists: lists,
		Topics:       topics,
	}
	r.nextID++
	r.newsletters[n.ID] = n
	return n, nil
}

func (r *fakeNewsletterRepo) GetAllSentNewsletters(_ context.Context) ([]domain.SentNewsletter, error) {
	if r.getAllErr != nil {
		return nil, r.getAllErr
	}
	out := make([]domain.SentNewsletter, 0, len(r.newsletters))
	for _, n := range r.newsletters {
		out = append(out, *n)
	}
	return out, nil
}

func (r *fakeNewsletterRepo) GetSentNewsletterByID(_ context.Context, id uint, _ bool) (*domain.SentNewsletter, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	n, ok := r.newsletters[id]
	if !ok {
		return nil, fmt.Errorf("newsletter %d not found", id)
	}
	return n, nil
}

func (r *fakeNewsletterRepo) DeleteSentNewsletter(_ context.Context, id uint) error {
	if r.deleteErr != nil {
		return r.deleteErr
	}
	if _, ok := r.newsletters[id]; !ok {
		return fmt.Errorf("newsletter %d not found", id)
	}
	delete(r.newsletters, id)
	return nil
}

type fakeSender struct {
	calls []sendCall
	err   error
}

type sendCall struct {
	metadata  domain.MailMetadata
	body      string
	recipient domain.User
}

func (s *fakeSender) SendMail(_ context.Context, metadata domain.MailMetadata, body string, recipient domain.User) error {
	if s.err != nil {
		return s.err
	}
	s.calls = append(s.calls, sendCall{metadata: metadata, body: body, recipient: recipient})
	return nil
}

type fakeRenderer struct {
	metadata domain.MailMetadata
	body     string
	err      error
	lastData map[string]any

	htmlBody string
	htmlErr  error
}

func (r *fakeRenderer) Render(_ *string, data map[string]any) (domain.MailMetadata, string, error) {
	r.lastData = data
	return r.metadata, r.body, r.err
}

func (r *fakeRenderer) RenderHTML(_ string, data map[string]any) (string, error) {
	r.lastData = data
	if r.htmlErr != nil {
		return "", r.htmlErr
	}
	if r.htmlBody != "" {
		return r.htmlBody, nil
	}
	return r.body, nil
}
