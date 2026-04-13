package db

import (
	"time"

	"github.com/5000K/5000mails/domain"
	"gorm.io/gorm"
)

type MailingList struct {
	gorm.Model
	Name  string `gorm:"not null;uniqueIndex"`
	Users []User `gorm:"foreignKey:MailingListID"`
}

type User struct {
	gorm.Model
	Name          string     `gorm:"not null"`
	Email         string     `gorm:"not null;uniqueIndex"`
	ConfirmedAt   *time.Time // nil = unconfirmed, non-nil = confirmed
	MailingListID uint       `gorm:"not null"`
}

func ToGORMUser(u *domain.User) *User {
	return &User{
		Name:          u.Name,
		Email:         u.Email,
		ConfirmedAt:   u.ConfirmedAt,
		MailingListID: u.MailingListID,
	}
}

func ToDomainUser(u *User) *domain.User {
	return &domain.User{
		ID:            u.ID,
		Name:          u.Name,
		Email:         u.Email,
		ConfirmedAt:   u.ConfirmedAt,
		MailingListID: u.MailingListID,
	}
}

func ToDomainUsers(users []User) []domain.User {
	result := make([]domain.User, len(users))
	for i, u := range users {
		result[i] = *ToDomainUser(&u)
	}
	return result
}

func ToGORMList(l *domain.MailingList) *MailingList {
	return &MailingList{
		Name: l.Name,
	}
}

func ToDomainList(l *MailingList) *domain.MailingList {
	return &domain.MailingList{
		ID:   l.ID,
		Name: l.Name,
	}
}

type Confirmation struct {
	gorm.Model
	UserID uint   `gorm:"not null;index"`
	Token  string `gorm:"not null;uniqueIndex"`
}

func ToDomainConfirmation(c *Confirmation) *domain.Confirmation {
	return &domain.Confirmation{
		ID:     c.ID,
		UserID: c.UserID,
		Token:  c.Token,
	}
}

func ToGORMConfirmation(c *domain.Confirmation) *Confirmation {
	return &Confirmation{
		UserID: c.UserID,
		Token:  c.Token,
	}
}
