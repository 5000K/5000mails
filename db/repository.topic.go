package db

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/5000K/5000mails/domain"
)

func (r *MailingListRepository) CreateTopic(ctx context.Context, mailingListName, name, displayName string, defaultEnabled bool) (*domain.Topic, error) {
	t := &Topic{
		Name:            name,
		DisplayName:     displayName,
		MailingListName: mailingListName,
		DefaultEnabled:  defaultEnabled,
	}
	result := r.db.WithContext(ctx).Create(t)
	if result.Error != nil {
		r.logger.ErrorContext(ctx, "failed to create topic",
			slog.String("mailing_list_name", mailingListName),
			slog.String("name", name),
			slog.Any("error", result.Error),
		)
		return nil, fmt.Errorf("create topic %q on list %q: %w", name, mailingListName, result.Error)
	}
	return ToDomainTopic(t), nil
}

func (r *MailingListRepository) GetTopicsByList(ctx context.Context, mailingListName string) ([]domain.Topic, error) {
	var topics []Topic
	result := r.db.WithContext(ctx).Where("mailing_list_name = ?", mailingListName).Find(&topics)
	if result.Error != nil {
		r.logger.ErrorContext(ctx, "failed to get topics",
			slog.String("mailing_list_name", mailingListName),
			slog.Any("error", result.Error),
		)
		return nil, fmt.Errorf("get topics for list %q: %w", mailingListName, result.Error)
	}
	return ToDomainTopics(topics), nil
}

func (r *MailingListRepository) GetTopicByName(ctx context.Context, mailingListName, name string) (*domain.Topic, error) {
	var t Topic
	result := r.db.WithContext(ctx).Where("mailing_list_name = ? AND name = ?", mailingListName, name).First(&t)
	if result.Error != nil {
		return nil, fmt.Errorf("get topic %q on list %q: %w", name, mailingListName, result.Error)
	}
	return ToDomainTopic(&t), nil
}

func (r *MailingListRepository) UpdateTopic(ctx context.Context, mailingListName, name string, displayName *string, defaultEnabled *bool) (*domain.Topic, error) {
	updates := map[string]any{}
	if displayName != nil {
		updates["display_name"] = *displayName
	}
	if defaultEnabled != nil {
		updates["default_enabled"] = *defaultEnabled
	}
	if len(updates) == 0 {
		return r.GetTopicByName(ctx, mailingListName, name)
	}

	result := r.db.WithContext(ctx).
		Model(&Topic{}).
		Where("mailing_list_name = ? AND name = ?", mailingListName, name).
		Updates(updates)
	if result.Error != nil {
		return nil, fmt.Errorf("update topic %q on list %q: %w", name, mailingListName, result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, fmt.Errorf("update topic %q on list %q: not found", name, mailingListName)
	}
	return r.GetTopicByName(ctx, mailingListName, name)
}

func (r *MailingListRepository) DeleteTopic(ctx context.Context, mailingListName, name string) error {
	result := r.db.WithContext(ctx).
		Where("mailing_list_name = ? AND name = ?", mailingListName, name).
		Delete(&Topic{})
	if result.Error != nil {
		return fmt.Errorf("delete topic %q on list %q: %w", name, mailingListName, result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("delete topic %q on list %q: not found", name, mailingListName)
	}
	return nil
}

func (r *MailingListRepository) GetDefaultEnabledTopics(ctx context.Context, mailingListName string) ([]domain.Topic, error) {
	var topics []Topic
	result := r.db.WithContext(ctx).
		Where("mailing_list_name = ? AND default_enabled = ?", mailingListName, true).
		Find(&topics)
	if result.Error != nil {
		return nil, fmt.Errorf("get default-enabled topics for list %q: %w", mailingListName, result.Error)
	}
	return ToDomainTopics(topics), nil
}

func (r *MailingListRepository) SubscribeUserToTopics(ctx context.Context, userID uint, topicIDs []uint) error {
	user := &User{}
	user.ID = userID
	topics := make([]Topic, len(topicIDs))
	for i, id := range topicIDs {
		topics[i].ID = id
	}
	if err := r.db.WithContext(ctx).Model(user).Association("Topics").Append(topics); err != nil {
		return fmt.Errorf("subscribing user %d to topics: %w", userID, err)
	}
	return nil
}

func (r *MailingListRepository) UnsubscribeUserFromTopics(ctx context.Context, userID uint, topicIDs []uint) error {
	user := &User{}
	user.ID = userID
	topics := make([]Topic, len(topicIDs))
	for i, id := range topicIDs {
		topics[i].ID = id
	}
	if err := r.db.WithContext(ctx).Model(user).Association("Topics").Delete(topics); err != nil {
		return fmt.Errorf("unsubscribing user %d from topics: %w", userID, err)
	}
	return nil
}

func (r *MailingListRepository) SetUserTopics(ctx context.Context, userID uint, topicIDs []uint) error {
	user := &User{}
	user.ID = userID
	topics := make([]Topic, len(topicIDs))
	for i, id := range topicIDs {
		topics[i].ID = id
	}
	if err := r.db.WithContext(ctx).Model(user).Association("Topics").Replace(topics); err != nil {
		return fmt.Errorf("setting topics for user %d: %w", userID, err)
	}
	return nil
}

func (r *MailingListRepository) GetUserTopics(ctx context.Context, userID uint) ([]domain.Topic, error) {
	var topics []Topic
	user := &User{}
	user.ID = userID
	if err := r.db.WithContext(ctx).Model(user).Association("Topics").Find(&topics); err != nil {
		return nil, fmt.Errorf("get topics for user %d: %w", userID, err)
	}
	return ToDomainTopics(topics), nil
}

func (r *MailingListRepository) GetConfirmedUsersSubscribedToTopics(ctx context.Context, mailingListName string, topicNames []string) ([]domain.User, error) {
	var users []User
	result := r.db.WithContext(ctx).
		Distinct().
		Joins("JOIN user_topic_subscriptions ON user_topic_subscriptions.user_id = users.id").
		Joins("JOIN topics ON topics.id = user_topic_subscriptions.topic_id AND topics.deleted_at IS NULL").
		Where("users.mailing_list_name = ? AND users.confirmed_at IS NOT NULL AND users.deleted_at IS NULL", mailingListName).
		Where("topics.mailing_list_name = ? AND topics.name IN ?", mailingListName, topicNames).
		Find(&users)
	if result.Error != nil {
		return nil, fmt.Errorf("get users subscribed to topics on list %q: %w", mailingListName, result.Error)
	}
	return ToDomainUsers(users), nil
}

func (r *MailingListRepository) SubscribeAllUsersToTopic(ctx context.Context, mailingListName string, topicID uint) error {
	var users []User
	result := r.db.WithContext(ctx).Where("mailing_list_name = ?", mailingListName).Find(&users)
	if result.Error != nil {
		return fmt.Errorf("listing users for mass topic subscribe on list %q: %w", mailingListName, result.Error)
	}
	topic := &Topic{}
	topic.ID = topicID
	for _, u := range users {
		if err := r.db.WithContext(ctx).Model(&u).Association("Topics").Append(topic); err != nil {
			r.logger.ErrorContext(ctx, "failed to subscribe user to topic",
				slog.Uint64("user_id", uint64(u.ID)),
				slog.Uint64("topic_id", uint64(topicID)),
				slog.Any("error", err),
			)
			return fmt.Errorf("subscribing user %d to topic %d: %w", u.ID, topicID, err)
		}
	}
	return nil
}
