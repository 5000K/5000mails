package domain

import "time"

// MailingList represents a named list that users can subscribe to.
type MailingList struct {
	Name string
}

// User represents a subscriber on a mailing list.
type User struct {
	ID               uint
	Name             string
	Email            string
	ConfirmedAt      *time.Time
	MailingListName  string
	UnsubscribeToken string
}

// IsConfirmed returns true if the user has completed double opt-in.
func (u *User) IsConfirmed() bool {
	return u.ConfirmedAt != nil
}

type MailMetadata struct {
	Subject    string
	SenderName string
}

// Confirmation holds a pending double opt-in token for a user.
type Confirmation struct {
	ID     uint
	UserID uint
	Token  string
}

// UserCounts holds subscriber totals for a mailing list.
type UserCounts struct {
	Total     int
	Confirmed int
}

// SentNewsletter is an archived record of a dispatched newsletter.
type SentNewsletter struct {
	ID           uint
	Subject      string
	SenderName   string
	RawMarkdown  string
	SentAt       time.Time
	Recipients   []User
	MailingLists []MailingList
}

// ScheduledMail is a pending newsletter queued for future delivery.
// ScheduledAt and SentAt are unix timestamps (UTC).
type ScheduledMail struct {
	ID              uint
	MailingListName string
	RawMarkdown     string
	ScheduledAt     int64
	SentAt          *int64
}
