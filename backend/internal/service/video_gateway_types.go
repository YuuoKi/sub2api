package service

import (
	"context"
	"errors"
	"time"
)

const (
	VideoStatusQueued    = "queued"
	VideoStatusSubmitted = "submitted"
	VideoStatusRunning   = "running"
	VideoStatusSucceeded = "succeeded"
	VideoStatusFailed    = "failed"
	VideoStatusCancelled = "cancelled"
)

var (
	ErrVideoTaskNotFound         = errors.New("video task not found")
	ErrVideoTaskTerminalConflict = errors.New("video task terminal status conflicts with requested status")
	ErrVideoTaskForbidden        = errors.New("video task is outside employee scope")
)

type VideoTaskScope struct {
	UserID   int64
	APIKeyID int64
	GroupID  int64
}

type VideoKeyEncryptor interface {
	Encrypt(plaintext string) (string, error)
	Decrypt(ciphertext string) (string, error)
}

type VideoProviderAccount struct {
	ID              int64  `json:"id"`
	Provider        string `json:"provider"`
	DisplayName     string `json:"display_name"`
	Enabled         bool   `json:"enabled"`
	EncryptedAPIKey string `json:"-"`
	MaskedKey       string `json:"masked_key"`
	BaseURL         string `json:"-"`
	DefaultModel    string `json:"default_model"`
}

type VideoTask struct {
	ID                   int64
	APIKeyID             int64
	GroupID              int64
	ProviderAccountID    int64
	Provider             string
	Model                string
	TaskType             string
	Prompt               string
	Status               string
	UpstreamTaskID       string
	ResultURL            string
	LastFrameURL         string
	DurationSeconds      int
	Resolution           string
	UsageTotalTokens     *int64
	CostAmount           float64
	Currency             string
	RealDispatchCount    int
	ProviderErrorCode    string
	ProviderErrorMessage string
	ErrorMessage         string
	CreationKey          string
	Version              int64
	DispatchState        string
	CreatedBy            int64
	CreatedAt            time.Time
	UpdatedAt            time.Time
	CompletedAt          *time.Time
}

type VideoTaskFinalization struct {
	TaskID               int64
	ExpectedVersion      int64
	Status               string
	ResultURL            string
	LastFrameURL         string
	UsageTotalTokens     *int64
	CostAmount           float64
	Currency             string
	ProviderErrorCode    string
	ProviderErrorMessage string
	ErrorMessage         string
	CompletedAt          time.Time
}

type VideoTaskFinalizationResult struct {
	Applied    bool
	Idempotent bool
	Status     string
	Version    int64
}

type VideoGatewayRepository interface {
	CreateTask(context.Context, *VideoTask) error
	GetTask(context.Context, int64) (*VideoTask, error)
	GetTaskForScope(context.Context, int64, VideoTaskScope) (*VideoTask, error)
	CancelTaskForScope(context.Context, int64, VideoTaskScope) (*VideoTask, error)
	ClaimRunnableTasks(context.Context, int, time.Duration) ([]*VideoTask, error)
	FinalizeTask(context.Context, VideoTaskFinalization) (VideoTaskFinalizationResult, error)
}

type VideoGatewayRuntimeRepository interface {
	VideoGatewayRepository
	ListEnabledVideoProviders(context.Context) ([]VideoProviderAccount, error)
	GetVideoProvider(context.Context, int64) (*VideoProviderAccount, error)
	BeginRealDispatch(context.Context, int64, int64) (bool, error)
	MarkVideoSubmitted(context.Context, int64, int64, string) error
}
