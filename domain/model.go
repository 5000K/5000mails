package domain

import (
	"errors"
	"time"
)

var ErrUserAlreadyConfirmed = errors.New("user already confirmed")

type MailingList struct {
	Name string
}

type User struct {
	ID               uint
	Name             string
	Email            string
	ConfirmedAt      *time.Time
	MailingListName  string
	UnsubscribeToken string
}

func (u *User) IsConfirmed() bool {
	return u.ConfirmedAt != nil
}

type MailMetadata struct {
	Subject    string
	SenderName string
}

type Confirmation struct {
	ID     uint
	UserID uint
	Token  string
}

type UserCounts struct {
	Total     int
	Confirmed int
}

type Topic struct {
	ID              uint
	Name            string
	DisplayName     string
	MailingListName string
	DefaultEnabled  bool
}

type SentNewsletter struct {
	ID           uint
	Subject      string
	SenderName   string
	RawMarkdown  string
	SentAt       time.Time
	Recipients   []User
	MailingLists []MailingList
	Topics       []Topic
}

type ScheduledMail struct {
	ID              uint
	MailingListName string
	RawMarkdown     string
	ScheduledAt     int64
	SentAt          *int64
	TopicNames      []string
}
