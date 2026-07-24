package service

import (
	"context"
	"errors"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type qcanvasPairAPIKeyRepoStub struct {
	APIKeyRepository
	persisted []*APIKey
	err       error
}

func (s *qcanvasPairAPIKeyRepoStub) CreatePair(_ context.Context, video, media *APIKey) error {
	if s.err != nil {
		return s.err
	}
	videoCopy, mediaCopy := *video, *media
	videoCopy.ID, mediaCopy.ID = 101, 102
	video.ID, media.ID = videoCopy.ID, mediaCopy.ID
	s.persisted = append(s.persisted, &videoCopy, &mediaCopy)
	return nil
}

type qcanvasPairUserRepoStub struct {
	UserRepository
	user *User
}

func (s *qcanvasPairUserRepoStub) GetByID(context.Context, int64) (*User, error) {
	return s.user, nil
}

type qcanvasPairGroupRepoStub struct {
	GroupRepository
	groups map[int64]*Group
	err    error
}

func (s *qcanvasPairGroupRepoStub) GetByID(_ context.Context, id int64) (*Group, error) {
	if s.err != nil {
		return nil, s.err
	}
	group, ok := s.groups[id]
	if !ok {
		return nil, ErrGroupNotFound
	}
	copy := *group
	return &copy, nil
}

func TestAPIKeyService_CreateQCanvasKeyPair(t *testing.T) {
	activeGroup := func(id int64) *Group {
		return &Group{ID: id, Status: StatusActive, SubscriptionType: SubscriptionTypeStandard}
	}

	t.Run("creates two logical keys for one user", func(t *testing.T) {
		repo := &qcanvasPairAPIKeyRepoStub{}
		svc := &APIKeyService{
			cfg:        &config.Config{},
			apiKeyRepo: repo,
			userRepo:   &qcanvasPairUserRepoStub{user: &User{ID: 7}},
			groupRepo:  &qcanvasPairGroupRepoStub{groups: map[int64]*Group{11: activeGroup(11), 22: activeGroup(22)}},
		}

		pair, err := svc.CreateQCanvasKeyPair(context.Background(), 7, CreateQCanvasKeyPairRequest{VideoGroupID: 11, MediaGroupID: 22})
		require.NoError(t, err)
		require.NotNil(t, pair)
		require.Equal(t, int64(7), pair.Video.UserID)
		require.Equal(t, int64(7), pair.Media.UserID)
		require.Equal(t, int64(11), *pair.Video.GroupID)
		require.Equal(t, int64(22), *pair.Media.GroupID)
		require.NotEqual(t, pair.Video.Key, pair.Media.Key)
		require.Len(t, repo.persisted, 2)
	})

	t.Run("creates both keys on the same group when groups match", func(t *testing.T) {
		repo := &qcanvasPairAPIKeyRepoStub{}
		svc := &APIKeyService{
			cfg:        &config.Config{},
			apiKeyRepo: repo,
			userRepo:   &qcanvasPairUserRepoStub{user: &User{ID: 7}},
			groupRepo:  &qcanvasPairGroupRepoStub{groups: map[int64]*Group{11: activeGroup(11)}},
		}

		pair, err := svc.CreateQCanvasKeyPair(context.Background(), 7, CreateQCanvasKeyPairRequest{VideoGroupID: 11, MediaGroupID: 11})
		require.NoError(t, err)
		require.NotNil(t, pair)
		require.Equal(t, int64(11), *pair.Video.GroupID)
		require.Equal(t, int64(11), *pair.Media.GroupID)
		require.NotEqual(t, pair.Video.Key, pair.Media.Key)
		require.Len(t, repo.persisted, 2)
	})

	t.Run("rejects unavailable groups before any write", func(t *testing.T) {
		repo := &qcanvasPairAPIKeyRepoStub{}
		svc := &APIKeyService{
			cfg:        &config.Config{},
			apiKeyRepo: repo,
			userRepo:   &qcanvasPairUserRepoStub{user: &User{ID: 7}},
			groupRepo:  &qcanvasPairGroupRepoStub{groups: map[int64]*Group{11: activeGroup(11), 22: {ID: 22, Status: StatusDisabled, SubscriptionType: SubscriptionTypeStandard}}},
		}

		_, err := svc.CreateQCanvasKeyPair(context.Background(), 7, CreateQCanvasKeyPairRequest{VideoGroupID: 11, MediaGroupID: 22})
		require.Error(t, err)
		require.Empty(t, repo.persisted)
	})

	t.Run("second persistence failure leaves no partial pair", func(t *testing.T) {
		repo := &qcanvasPairAPIKeyRepoStub{err: errors.New("second insert failed")}
		svc := &APIKeyService{
			cfg:        &config.Config{},
			apiKeyRepo: repo,
			userRepo:   &qcanvasPairUserRepoStub{user: &User{ID: 7}},
			groupRepo:  &qcanvasPairGroupRepoStub{groups: map[int64]*Group{11: activeGroup(11), 22: activeGroup(22)}},
		}

		_, err := svc.CreateQCanvasKeyPair(context.Background(), 7, CreateQCanvasKeyPairRequest{VideoGroupID: 11, MediaGroupID: 22})
		require.Error(t, err)
		require.Empty(t, repo.persisted)
	})
}
