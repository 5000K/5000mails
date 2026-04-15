package db

import (
	"time"

	"github.com/5000K/5000mails/domain"
	"gorm.io/gorm"
)

type MailingList struct {
	Name   string  `gorm:"primaryKey"`
	Users  []User  `gorm:"foreignKey:MailingListName"`
	Topics []Topic `gorm:"foreignKey:MailingListName"`
}

type User struct {
	gorm.Model
	Name             string `gorm:"not null"`
	Email            string `gorm:"not null;uniqueIndex:idx_user_email_list"`
	ConfirmedAt      *time.Time
	MailingListName  string  `gorm:"not null;uniqueIndex:idx_user_email_list"`
	UnsubscribeToken string  `gorm:"not null;uniqueIndex"`
	Topics           []Topic `gorm:"many2many:user_topic_subscriptions;"`
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

type Topic struct {
	gorm.Model
	Name            string `gorm:"not null;uniqueIndex:idx_topic_list_name"`
	DisplayName     string `gorm:"not null"`
	MailingListName string `gorm:"not null;uniqueIndex:idx_topic_list_name"`
	DefaultEnabled  bool   `gorm:"not null;default:false"`
}

func ToDomainTopic(t *Topic) *domain.Topic {
	return &domain.Topic{
		ID:              t.ID,
		Name:            t.Name,
		DisplayName:     t.DisplayName,
		MailingListName: t.MailingListName,
		DefaultEnabled:  t.DefaultEnabled,
	}
}

func ToDomainTopics(topics []Topic) []domain.Topic {
	result := make([]domain.Topic, len(topics))
	for i, t := range topics {
		result[i] = *ToDomainTopic(&t)
	}
	return result
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

type SentNewsletter struct {
	gorm.Model
	Subject      string
	SenderName   string
	RawMarkdown  string
	Recipients   []User        `gorm:"many2many:sent_newsletter_recipients;"`
	MailingLists []MailingList `gorm:"many2many:sent_newsletter_mailing_lists;"`
	Topics       []Topic       `gorm:"many2many:sent_newsletter_topics;"`
}

func ToDomainSentNewsletter(n *SentNewsletter) *domain.SentNewsletter {
	lists := make([]domain.MailingList, len(n.MailingLists))
	for i, l := range n.MailingLists {
		lists[i] = *ToDomainList(&l)
	}
	return &domain.SentNewsletter{
		ID:           n.ID,
		Subject:      n.Subject,
		SenderName:   n.SenderName,
		RawMarkdown:  n.RawMarkdown,
		SentAt:       n.CreatedAt,
		Recipients:   ToDomainUsers(n.Recipients),
		MailingLists: lists,
		Topics:       ToDomainTopics(n.Topics),
	}
}

func ToDomainSentNewsletters(newsletters []SentNewsletter) []domain.SentNewsletter {
	result := make([]domain.SentNewsletter, len(newsletters))
	for i := range newsletters {
		result[i] = *ToDomainSentNewsletter(&newsletters[i])
	}
	return result
}

type ScheduledMailTopic struct {
	ScheduledMailID uint   `gorm:"primaryKey"`
	TopicName       string `gorm:"primaryKey"`
}

type ScheduledMail struct {
	gorm.Model
	MailingListName string `gorm:"not null;index"`
	RawMarkdown     string `gorm:"not null"`
	ScheduledAt     int64  `gorm:"not null;index"`
	SentAt          *int64
	Topics          []ScheduledMailTopic `gorm:"foreignKey:ScheduledMailID"`
}

func ToDomainScheduledMail(m *ScheduledMail) *domain.ScheduledMail {
	topicNames := make([]string, len(m.Topics))
	for i, t := range m.Topics {
		topicNames[i] = t.TopicName
	}
	return &domain.ScheduledMail{
		ID:              m.ID,
		MailingListName: m.MailingListName,
		RawMarkdown:     m.RawMarkdown,
		ScheduledAt:     m.ScheduledAt,
		SentAt:          m.SentAt,
		TopicNames:      topicNames,
	}
}

func ToDomainScheduledMails(mails []ScheduledMail) []domain.ScheduledMail {
	result := make([]domain.ScheduledMail, len(mails))
	for i := range mails {
		result[i] = *ToDomainScheduledMail(&mails[i])
	}
	return result
}
