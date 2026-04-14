package db

import (
	"time"

	"github.com/5000K/5000mails/domain"
	"gorm.io/gorm"
)

type MailingList struct {
	Name  string `gorm:"primaryKey"`
	Users []User `gorm:"foreignKey:MailingListName"`
}

type User struct {
	gorm.Model
	Name             string `gorm:"not null"`
	Email            string `gorm:"not null;uniqueIndex:idx_user_email_list"`
	ConfirmedAt      *time.Time
	MailingListName  string `gorm:"not null;uniqueIndex:idx_user_email_list"`
	UnsubscribeToken string `gorm:"not null;uniqueIndex"`
}

func ToGORMUser(u *domain.User) *User {
	return &User{
		Name:             u.Name,
		Email:            u.Email,
		ConfirmedAt:      u.ConfirmedAt,
		MailingListName:  u.MailingListName,
		UnsubscribeToken: u.UnsubscribeToken,
	}
}

func ToDomainUser(u *User) *domain.User {
	return &domain.User{
		ID:               u.ID,
		Name:             u.Name,
		Email:            u.Email,
		ConfirmedAt:      u.ConfirmedAt,
		MailingListName:  u.MailingListName,
		UnsubscribeToken: u.UnsubscribeToken,
	}
}

func ToDomainUsers(users []User) []domain.User {
	result := make([]domain.User, len(users))
	for i, u := range users {
		result[i] = *ToDomainUser(&u)
	}
	return result
}

func ToDomainList(l *MailingList) *domain.MailingList {
	return &domain.MailingList{
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
