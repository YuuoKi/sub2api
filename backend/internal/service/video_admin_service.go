package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

var (
	ErrVideoAdminInvalidRequest        = errors.New("invalid video provider request")
	ErrVideoAdminInvalidGroup          = errors.New("video provider group must be active and standard")
	ErrVideoAdminConflict              = errors.New("video provider model already exists")
	ErrVideoAdminAuthorizationConflict = errors.New("tiny_real authorization is unavailable")
)

type VideoProviderAdminCreate struct {
	GroupID                                              int64
	Provider, DisplayName, APIKey, BaseURL, DefaultModel string
	Enabled                                              bool
}

type VideoProviderAdminUpdate struct {
	GroupID         *int64
	DisplayName     *string
	APIKey          *string
	BaseURL         *string
	DefaultModel    *string
	Enabled         *bool
	EncryptedAPIKey *string
	MaskedKey       *string
}

type VideoAdminTaskFilter struct {
	Page, PageSize int
	Status         string
}
type VideoSystemCheck struct {
	ProviderCount           int64 `json:"provider_count"`
	EnabledProviderCount    int64 `json:"enabled_provider_count"`
	AuthorizedProviderCount int64 `json:"authorized_provider_count"`
	TaskCount               int64 `json:"task_count"`
	RealDispatchCount       int64 `json:"real_dispatch_count"`
	GlobalTinyRealConsumed  bool  `json:"global_tiny_real_consumed"`
}

type VideoAdminRepository interface {
	ListVideoProviders(context.Context) ([]VideoProviderAccount, error)
	CreateVideoProvider(context.Context, VideoProviderAccount) (*VideoProviderAccount, error)
	UpdateVideoProvider(context.Context, int64, VideoProviderAdminUpdate) (*VideoProviderAccount, error)
	AuthorizeTinyReal(context.Context, int64, int64) (*VideoProviderAccount, error)
	ListVideoTasks(context.Context, VideoAdminTaskFilter) ([]VideoTask, int64, error)
	GetVideoTaskAdmin(context.Context, int64) (*VideoTask, error)
	VideoSystemCheck(context.Context) (VideoSystemCheck, error)
}

type VideoAdminService struct {
	repo      VideoAdminRepository
	encryptor VideoKeyEncryptor
}

func NewVideoAdminService(repo VideoAdminRepository, encryptor VideoKeyEncryptor) *VideoAdminService {
	return &VideoAdminService{repo: repo, encryptor: encryptor}
}

func (s *VideoAdminService) ListProviders(ctx context.Context) ([]VideoProviderAccount, error) {
	return s.repo.ListVideoProviders(ctx)
}
func (s *VideoAdminService) CreateProvider(ctx context.Context, in VideoProviderAdminCreate) (*VideoProviderAccount, error) {
	if strings.ToLower(strings.TrimSpace(in.Provider)) != "seedance" {
		return nil, fmt.Errorf("%w: only seedance provider is supported", ErrVideoAdminInvalidRequest)
	}
	if in.GroupID <= 0 || strings.TrimSpace(in.DisplayName) == "" || strings.TrimSpace(in.APIKey) == "" {
		return nil, fmt.Errorf("%w: group, display name and api key are required", ErrVideoAdminInvalidRequest)
	}
	encrypted, err := s.encryptor.Encrypt(strings.TrimSpace(in.APIKey))
	if err != nil {
		return nil, err
	}
	item := VideoProviderAccount{GroupID: in.GroupID, Provider: "seedance", DisplayName: strings.TrimSpace(in.DisplayName), Enabled: in.Enabled, EncryptedAPIKey: encrypted, MaskedKey: maskVideoSecret(in.APIKey), BaseURL: SeedanceBaseURL, DefaultModel: SeedanceModel, APIKeyConfigured: true}
	created, err := s.repo.CreateVideoProvider(ctx, item)
	if err != nil {
		return nil, err
	}
	redactVideoProvider(created)
	return created, nil
}
func (s *VideoAdminService) UpdateProvider(ctx context.Context, id int64, in VideoProviderAdminUpdate) (*VideoProviderAccount, error) {
	if id <= 0 {
		return nil, ErrVideoProviderNotFound
	}
	if in.APIKey != nil && strings.TrimSpace(*in.APIKey) != "" {
		encrypted, err := s.encryptor.Encrypt(strings.TrimSpace(*in.APIKey))
		if err != nil {
			return nil, err
		}
		masked := maskVideoSecret(*in.APIKey)
		in.EncryptedAPIKey, in.MaskedKey = &encrypted, &masked
	}
	canonicalBaseURL, canonicalModel := SeedanceBaseURL, SeedanceModel
	in.BaseURL, in.DefaultModel = &canonicalBaseURL, &canonicalModel
	item, err := s.repo.UpdateVideoProvider(ctx, id, in)
	if err != nil {
		return nil, err
	}
	redactVideoProvider(item)
	return item, nil
}
func (s *VideoAdminService) AuthorizeTinyReal(ctx context.Context, id, actor int64) (*VideoProviderAccount, error) {
	item, err := s.repo.AuthorizeTinyReal(ctx, id, actor)
	if err == nil {
		redactVideoProvider(item)
	}
	return item, err
}
func (s *VideoAdminService) ListTasks(ctx context.Context, filter VideoAdminTaskFilter) ([]VideoTask, int64, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 || filter.PageSize > 100 {
		filter.PageSize = 20
	}
	return s.repo.ListVideoTasks(ctx, filter)
}
func (s *VideoAdminService) GetTask(ctx context.Context, id int64) (*VideoTask, error) {
	return s.repo.GetVideoTaskAdmin(ctx, id)
}
func (s *VideoAdminService) SystemCheck(ctx context.Context) (VideoSystemCheck, error) {
	return s.repo.VideoSystemCheck(ctx)
}
func maskVideoSecret(value string) string {
	value = strings.TrimSpace(value)
	if len(value) <= 8 {
		return "••••••••"
	}
	return value[:4] + "…" + value[len(value)-4:]
}
func redactVideoProvider(item *VideoProviderAccount) {
	if item != nil {
		item.EncryptedAPIKey = ""
		item.APIKeyConfigured = item.MaskedKey != ""
	}
}
