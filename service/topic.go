package service

import (
	"context"
	"fmt"

	"github.com/5000K/5000mails/domain"
)

type TopicService struct {
	topics domain.TopicRepository
	lists  domain.MailingListRepository
}

func NewTopicService(topics domain.TopicRepository, lists domain.MailingListRepository) *TopicService {
	return &TopicService{topics: topics, lists: lists}
}

func (s *TopicService) Create(ctx context.Context, listName, name, displayName string, defaultEnabled, subscribeExisting bool) (*domain.Topic, error) {
	if _, err := s.lists.GetListByName(ctx, listName); err != nil {
		return nil, fmt.Errorf("list %q not found: %w", listName, err)
	}

	topic, err := s.topics.CreateTopic(ctx, listName, name, displayName, defaultEnabled)
	if err != nil {
		return nil, fmt.Errorf("creating topic %q on list %q: %w", name, listName, err)
	}

	if subscribeExisting {
		if err := s.topics.SubscribeAllUsersToTopic(ctx, listName, topic.ID); err != nil {
			return nil, fmt.Errorf("subscribing existing users to topic %q: %w", name, err)
		}
	}

	return topic, nil
}

func (s *TopicService) List(ctx context.Context, listName string) ([]domain.Topic, error) {
	topics, err := s.topics.GetTopicsByList(ctx, listName)
	if err != nil {
		return nil, fmt.Errorf("listing topics for list %q: %w", listName, err)
	}
	return topics, nil
}

func (s *TopicService) Get(ctx context.Context, listName, name string) (*domain.Topic, error) {
	topic, err := s.topics.GetTopicByName(ctx, listName, name)
	if err != nil {
		return nil, fmt.Errorf("getting topic %q on list %q: %w", name, listName, err)
	}
	return topic, nil
}

func (s *TopicService) Update(ctx context.Context, listName, name string, displayName *string, defaultEnabled *bool) (*domain.Topic, error) {
	topic, err := s.topics.UpdateTopic(ctx, listName, name, displayName, defaultEnabled)
	if err != nil {
		return nil, fmt.Errorf("updating topic %q on list %q: %w", name, listName, err)
	}
	return topic, nil
}

func (s *TopicService) Delete(ctx context.Context, listName, name string) error {
	if err := s.topics.DeleteTopic(ctx, listName, name); err != nil {
		return fmt.Errorf("deleting topic %q on list %q: %w", name, listName, err)
	}
	return nil
}

func (s *TopicService) GetUserTopics(ctx context.Context, _ string, userID uint) ([]domain.Topic, error) {
	topics, err := s.topics.GetUserTopics(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("getting topics for user %d: %w", userID, err)
	}
	return topics, nil
}

func (s *TopicService) SetUserTopics(ctx context.Context, _ string, userID uint, topicIDs []uint) error {
	if err := s.topics.SetUserTopics(ctx, userID, topicIDs); err != nil {
		return fmt.Errorf("setting topics for user %d: %w", userID, err)
	}
	return nil
}
