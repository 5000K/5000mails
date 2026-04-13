package domain

import "context"

type MailingListRepository interface {
	CreateList(ctx context.Context, name string) (*MailingList, error)
	GetList(ctx context.Context, id uint) (*MailingList, error)
	GetListByName(ctx context.Context, name string) (*MailingList, error)
	UpdateList(ctx context.Context, id uint, name string) (*MailingList, error)
	DeleteList(ctx context.Context, id uint) error
}

type UserRepository interface {
	AddUser(ctx context.Context, mailingListID uint, name, email string) (*User, error)
	ConfirmUser(ctx context.Context, userID uint) error
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	GetUsers(ctx context.Context, mailingListID uint) ([]User, error)
	GetConfirmedUsers(ctx context.Context, mailingListID uint) ([]User, error)
	RemoveUser(ctx context.Context, userID uint) error
}

type ConfirmationRepository interface {
	CreateConfirmation(ctx context.Context, userID uint, token string) (*Confirmation, error)
	GetConfirmationByToken(ctx context.Context, token string) (*Confirmation, error)
	DeleteConfirmation(ctx context.Context, id uint) error
}

type Renderer interface {
	Render(raw *string, data map[string]any) (metadata MailMetadata, body string, err error)
}

type Sender interface {
	SendMail(ctx context.Context, metadata MailMetadata, body string, recipient User) error
}
