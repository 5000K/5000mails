package domain

import "time"

// MailingList represents a named list that users can subscribe to.
type MailingList struct {
	ID   uint
	Name string
}

// User represents a subscriber on a mailing list.
type User struct {
	ID               uint
	Name             string
	Email            string
	ConfirmedAt      *time.Time
	MailingListID    uint
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
