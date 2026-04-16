package domain

import "context"

type MailingListRepository interface {
	CreateList(ctx context.Context, name string) (*MailingList, error)
	GetAllLists(ctx context.Context) ([]MailingList, error)
	GetListByName(ctx context.Context, name string) (*MailingList, error)
	RenameList(ctx context.Context, name, newName string) (*MailingList, error)
	DeleteList(ctx context.Context, name string) error
}

type UserRepository interface {
	AddUser(ctx context.Context, mailingListName string, name, email, unsubscribeToken string) (*User, error)
	ConfirmUser(ctx context.Context, userID uint) error
	GetUserByEmail(ctx context.Context, mailingListName, email string) (*User, error)
	GetUnsubscribedUserByEmail(ctx context.Context, mailingListName, email string) (*User, error)
	GetUserByUnsubscribeToken(ctx context.Context, token string) (*User, error)
	GetUsers(ctx context.Context, mailingListName string) ([]User, error)
	GetConfirmedUsers(ctx context.Context, mailingListName string) ([]User, error)
	RemoveUser(ctx context.Context, userID uint) error
	ReactivateUser(ctx context.Context, userID uint, name, unsubscribeToken string) (*User, error)
}

type ConfirmationRepository interface {
	CreateConfirmation(ctx context.Context, userID uint, token string) (*Confirmation, error)
	GetConfirmationByToken(ctx context.Context, token string) (*Confirmation, error)
	DeleteConfirmation(ctx context.Context, id uint) error
	DeleteConfirmationsByUserID(ctx context.Context, userID uint) error
}

type TopicRepository interface {
	CreateTopic(ctx context.Context, mailingListName, name, displayName string, defaultEnabled bool) (*Topic, error)
	GetTopicsByList(ctx context.Context, mailingListName string) ([]Topic, error)
	GetTopicByName(ctx context.Context, mailingListName, name string) (*Topic, error)
	UpdateTopic(ctx context.Context, mailingListName, name string, displayName *string, defaultEnabled *bool) (*Topic, error)
	DeleteTopic(ctx context.Context, mailingListName, name string) error
	GetDefaultEnabledTopics(ctx context.Context, mailingListName string) ([]Topic, error)
	SubscribeUserToTopics(ctx context.Context, userID uint, topicIDs []uint) error
	UnsubscribeUserFromTopics(ctx context.Context, userID uint, topicIDs []uint) error
	SetUserTopics(ctx context.Context, userID uint, topicIDs []uint) error
	GetUserTopics(ctx context.Context, userID uint) ([]Topic, error)
	GetConfirmedUsersSubscribedToTopics(ctx context.Context, mailingListName string, topicNames []string) ([]User, error)
	SubscribeAllUsersToTopic(ctx context.Context, mailingListName string, topicID uint) error
}

type SentNewsletterRepository interface {
	CreateSentNewsletter(ctx context.Context, subject, senderName, rawMarkdown string, recipientIDs []uint, listNames []string, topicNames []string) (*SentNewsletter, error)
	GetAllSentNewsletters(ctx context.Context) ([]SentNewsletter, error)
	GetSentNewsletterByID(ctx context.Context, id uint, withRecipients bool) (*SentNewsletter, error)
	DeleteSentNewsletter(ctx context.Context, id uint) error
}

type Renderer interface {
	Render(raw *string, data map[string]any) (metadata MailMetadata, body string, err error)
	RenderHTML(html string, data map[string]any) (string, error)
}

type Sender interface {
	SendMail(ctx context.Context, metadata MailMetadata, body string, recipient User) error
}

type ScheduledMailRepository interface {
	CreateScheduledMail(ctx context.Context, mailingListName, rawMarkdown string, scheduledAt int64, topicNames []string) (*ScheduledMail, error)
	GetAllScheduledMails(ctx context.Context) ([]ScheduledMail, error)
	GetScheduledMailByID(ctx context.Context, id uint) (*ScheduledMail, error)
	GetPendingScheduledMails(ctx context.Context, now int64) ([]ScheduledMail, error)
	UpdateScheduledMailTime(ctx context.Context, id uint, scheduledAt int64) (*ScheduledMail, error)
	UpdateScheduledMailContent(ctx context.Context, id uint, rawMarkdown string) (*ScheduledMail, error)
	MarkScheduledMailSent(ctx context.Context, id uint, sentAt int64) error
	DeleteScheduledMail(ctx context.Context, id uint) error
}
