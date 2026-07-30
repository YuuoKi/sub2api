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
	price := func(value float64) *float64 { return &value }
	videoGroup := func(id int64) *Group {
		group := activeGroup(id)
		group.Platform = PlatformHCAtom
		group.VideoPrice480P = price(0.05)
		group.VideoPrice720P = price(0.07)
		group.VideoPrice1080P = price(0.25)
		group.ModelsListConfig = GroupModelsListConfig{Enabled: true, Models: []string{
			HCAtomVideoV1PublicModel,
			HCAtomSeedanceV3PublicModel,
		}}
		return group
	}
	mediaGroup := func(id int64) *Group {
		group := activeGroup(id)
		group.Platform = PlatformHCAtom
		group.AllowImageGeneration = true
		group.AllowBatchImageGeneration = true
		group.ImagePrice1K = price(0.134)
		group.ImagePrice2K = price(0.201)
		group.ImagePrice4K = price(0.268)
		group.ModelsListConfig = GroupModelsListConfig{Enabled: true, Models: []string{
			HCAtomImageSeedreamModel,
			HCAtomImageDoubaoSeedreamModel,
			HCAtomImageGeminiModel,
			HCAtomImageGPTModel,
			HCAtomImageSGPTModel,
		}}
		return group
	}

	t.Run("creates two logical keys for one user", func(t *testing.T) {
		repo := &qcanvasPairAPIKeyRepoStub{}
		svc := &APIKeyService{
			cfg:        &config.Config{},
			apiKeyRepo: repo,
			userRepo:   &qcanvasPairUserRepoStub{user: &User{ID: 7, Role: RoleUser, Status: StatusActive}},
			groupRepo:  &qcanvasPairGroupRepoStub{groups: map[int64]*Group{11: videoGroup(11), 22: mediaGroup(22)}},
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

	t.Run("rejects swapped image and video capabilities before any write", func(t *testing.T) {
		repo := &qcanvasPairAPIKeyRepoStub{}
		svc := &APIKeyService{
			cfg:        &config.Config{},
			apiKeyRepo: repo,
			userRepo:   &qcanvasPairUserRepoStub{user: &User{ID: 7, Role: RoleUser, Status: StatusActive}},
			groupRepo:  &qcanvasPairGroupRepoStub{groups: map[int64]*Group{11: mediaGroup(11), 22: videoGroup(22)}},
		}

		_, err := svc.CreateQCanvasKeyPair(context.Background(), 7, CreateQCanvasKeyPairRequest{VideoGroupID: 11, MediaGroupID: 22})
		require.ErrorContains(t, err, "QC_KEY_PAIR_VIDEO_GROUP_INVALID")
		require.Empty(t, repo.persisted)
	})

	t.Run("rejects an image group missing one authorized model", func(t *testing.T) {
		repo := &qcanvasPairAPIKeyRepoStub{}
		incomplete := mediaGroup(22)
		incomplete.ModelsListConfig.Models = incomplete.ModelsListConfig.Models[:4]
		svc := &APIKeyService{
			cfg:        &config.Config{},
			apiKeyRepo: repo,
			userRepo:   &qcanvasPairUserRepoStub{user: &User{ID: 7, Role: RoleUser, Status: StatusActive}},
			groupRepo:  &qcanvasPairGroupRepoStub{groups: map[int64]*Group{11: videoGroup(11), 22: incomplete}},
		}

		_, err := svc.CreateQCanvasKeyPair(context.Background(), 7, CreateQCanvasKeyPairRequest{VideoGroupID: 11, MediaGroupID: 22})
		require.ErrorContains(t, err, "QC_KEY_PAIR_MEDIA_GROUP_INVALID")
		require.Empty(t, repo.persisted)
	})

	t.Run("rejects an image group containing any extra model", func(t *testing.T) {
		repo := &qcanvasPairAPIKeyRepoStub{}
		mixed := mediaGroup(22)
		mixed.ModelsListConfig.Models = append(mixed.ModelsListConfig.Models, "gpt-5.6-sol")
		svc := &APIKeyService{
			cfg:        &config.Config{},
			apiKeyRepo: repo,
			userRepo:   &qcanvasPairUserRepoStub{user: &User{ID: 7, Role: RoleUser, Status: StatusActive}},
			groupRepo:  &qcanvasPairGroupRepoStub{groups: map[int64]*Group{11: videoGroup(11), 22: mixed}},
		}

		_, err := svc.CreateQCanvasKeyPair(context.Background(), 7, CreateQCanvasKeyPairRequest{VideoGroupID: 11, MediaGroupID: 22})
		require.ErrorContains(t, err, "QC_KEY_PAIR_MEDIA_GROUP_INVALID")
		require.Empty(t, repo.persisted)
	})

	t.Run("rejects matching media and video groups before any write", func(t *testing.T) {
		repo := &qcanvasPairAPIKeyRepoStub{}
		svc := &APIKeyService{
			cfg:        &config.Config{},
			apiKeyRepo: repo,
			userRepo:   &qcanvasPairUserRepoStub{user: &User{ID: 7, Role: RoleUser, Status: StatusActive}},
			groupRepo:  &qcanvasPairGroupRepoStub{groups: map[int64]*Group{11: activeGroup(11)}},
		}

		pair, err := svc.CreateQCanvasKeyPair(context.Background(), 7, CreateQCanvasKeyPairRequest{VideoGroupID: 11, MediaGroupID: 11})
		require.Error(t, err)
		require.Nil(t, pair)
		require.ErrorContains(t, err, "distinct groups")
		require.Empty(t, repo.persisted)
	})

	t.Run("rejects unavailable groups before any write", func(t *testing.T) {
		repo := &qcanvasPairAPIKeyRepoStub{}
		svc := &APIKeyService{
			cfg:        &config.Config{},
			apiKeyRepo: repo,
			userRepo:   &qcanvasPairUserRepoStub{user: &User{ID: 7, Role: RoleUser, Status: StatusActive}},
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
			userRepo:   &qcanvasPairUserRepoStub{user: &User{ID: 7, Role: RoleUser, Status: StatusActive}},
			groupRepo:  &qcanvasPairGroupRepoStub{groups: map[int64]*Group{11: videoGroup(11), 22: mediaGroup(22)}},
		}

		_, err := svc.CreateQCanvasKeyPair(context.Background(), 7, CreateQCanvasKeyPairRequest{VideoGroupID: 11, MediaGroupID: 22})
		require.Error(t, err)
		require.Empty(t, repo.persisted)
	})

	t.Run("rejects admin target before minting keys", func(t *testing.T) {
		repo := &qcanvasPairAPIKeyRepoStub{}
		svc := &APIKeyService{
			cfg:        &config.Config{},
			apiKeyRepo: repo,
			userRepo:   &qcanvasPairUserRepoStub{user: &User{ID: 7, Role: RoleAdmin, Status: StatusActive}},
			groupRepo:  &qcanvasPairGroupRepoStub{groups: map[int64]*Group{11: videoGroup(11), 22: mediaGroup(22)}},
		}

		_, err := svc.CreateQCanvasKeyPair(context.Background(), 7, CreateQCanvasKeyPairRequest{VideoGroupID: 11, MediaGroupID: 22})
		require.Error(t, err)
		require.Contains(t, err.Error(), "管理员")
		require.Empty(t, repo.persisted)
	})

	t.Run("allows admin target when explicitly opted in", func(t *testing.T) {
		repo := &qcanvasPairAPIKeyRepoStub{}
		svc := &APIKeyService{
			cfg:        &config.Config{},
			apiKeyRepo: repo,
			userRepo:   &qcanvasPairUserRepoStub{user: &User{ID: 7, Role: RoleAdmin, Status: StatusActive}},
			groupRepo:  &qcanvasPairGroupRepoStub{groups: map[int64]*Group{11: videoGroup(11), 22: mediaGroup(22)}},
		}

		pair, err := svc.CreateQCanvasKeyPair(context.Background(), 7, CreateQCanvasKeyPairRequest{
			VideoGroupID:     11,
			MediaGroupID:     22,
			AllowAdminTarget: true,
		})
		require.NoError(t, err)
		require.NotNil(t, pair)
		require.NotNil(t, pair.Video)
		require.NotNil(t, pair.Media)
		require.Len(t, repo.persisted, 2)
	})

	t.Run("rejects disabled target before minting keys", func(t *testing.T) {
		repo := &qcanvasPairAPIKeyRepoStub{}
		svc := &APIKeyService{
			cfg:        &config.Config{},
			apiKeyRepo: repo,
			userRepo:   &qcanvasPairUserRepoStub{user: &User{ID: 7, Role: RoleUser, Status: StatusDisabled}},
			groupRepo:  &qcanvasPairGroupRepoStub{groups: map[int64]*Group{11: activeGroup(11), 22: activeGroup(22)}},
		}

		_, err := svc.CreateQCanvasKeyPair(context.Background(), 7, CreateQCanvasKeyPairRequest{VideoGroupID: 11, MediaGroupID: 22})
		require.Error(t, err)
		require.Contains(t, err.Error(), "停用")
		require.Empty(t, repo.persisted)
	})

	t.Run("rejects disabled admin even with allow_admin_target", func(t *testing.T) {
		repo := &qcanvasPairAPIKeyRepoStub{}
		svc := &APIKeyService{
			cfg:        &config.Config{},
			apiKeyRepo: repo,
			userRepo:   &qcanvasPairUserRepoStub{user: &User{ID: 7, Role: RoleAdmin, Status: StatusDisabled}},
			groupRepo:  &qcanvasPairGroupRepoStub{groups: map[int64]*Group{11: activeGroup(11), 22: activeGroup(22)}},
		}

		_, err := svc.CreateQCanvasKeyPair(context.Background(), 7, CreateQCanvasKeyPairRequest{
			VideoGroupID:     11,
			MediaGroupID:     22,
			AllowAdminTarget: true,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "停用")
		require.Empty(t, repo.persisted)
	})
}
