package service

import (
	"context"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	VideoProviderMock     = "mock"
	VideoProviderSeedance = "seedance"
	VideoProviderKling    = "kling"

	VideoTaskTypeTextToVideo      = "text_to_video"
	VideoTaskTypeImageToVideo     = "image_to_video"
	VideoTaskTypeReferenceToVideo = "reference_to_video"

	VideoStatusQueued    = "queued"
	VideoStatusSubmitted = "submitted"
	VideoStatusRunning   = "running"
	VideoStatusSucceeded = "succeeded"
	VideoStatusFailed    = "failed"
	VideoStatusCancelled = "cancelled"
)

var (
	ErrVideoProviderNotFound = infraerrors.NotFound("VIDEO_PROVIDER_NOT_FOUND", "video provider account not found")
	ErrVideoTaskNotFound     = infraerrors.NotFound("VIDEO_TASK_NOT_FOUND", "video task not found")
	ErrVideoProviderDisabled = infraerrors.BadRequest("VIDEO_PROVIDER_DISABLED", "video provider account is disabled")
	ErrVideoInvalidProvider  = infraerrors.BadRequest("VIDEO_INVALID_PROVIDER", "provider must be one of mock/seedance/kling")
	ErrVideoInvalidTaskType  = infraerrors.BadRequest("VIDEO_INVALID_TASK_TYPE", "task_type must be one of text_to_video/image_to_video/reference_to_video")
	ErrVideoInvalidStatus    = infraerrors.BadRequest("VIDEO_INVALID_STATUS", "invalid video task status")
	ErrVideoMissingPrompt    = infraerrors.BadRequest("VIDEO_MISSING_PROMPT", "prompt is required")
	ErrVideoMissingProvider  = infraerrors.BadRequest("VIDEO_MISSING_PROVIDER", "provider_account_id is required")
	ErrVideoKeyDecryptFailed = infraerrors.InternalServer("VIDEO_KEY_DECRYPT_FAILED", "video provider key decryption failed; please reconfigure the provider")
)

type VideoKeyEncryptor interface {
	Encrypt(plaintext string) (string, error)
	Decrypt(ciphertext string) (string, error)
}

type VideoProviderAccount struct {
	ID                  int64
	Provider            string
	DisplayName         string
	Enabled             bool
	EncryptedAPIKey     string
	PlainAPIKey         string
	APIKeyConfigured    bool
	APIKeyDecryptFailed bool
	MaskedKey           string
	BaseURL             string
	DefaultModel        string
	RateLimitPerMinute  int
	Metadata            map[string]any
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type VideoTask struct {
	ID                int64
	ProviderAccountID int64
	Provider          string
	Model             string
	TaskType          string
	Prompt            string
	NegativePrompt    string
	ReferenceImageURL string
	ReferenceVideoURL string
	AspectRatio       string
	Duration          int
	Resolution        string
	Status            string
	UpstreamTaskID    string
	ResultURL         string
	ErrorMessage      string
	CostEstimate      float64
	CreatedBy         int64
	CreatedAt         time.Time
	UpdatedAt         time.Time
	CompletedAt       *time.Time
}

type VideoTaskEvent struct {
	ID          int64
	VideoTaskID int64
	EventType   string
	Message     string
	Payload     map[string]any
	CreatedAt   time.Time
}

type VideoUsageSummary struct {
	Provider     string
	Model        string
	Status       string
	Count        int64
	CostEstimate float64
	Duration     int64
}

type VideoProviderStatus struct {
	Provider         string
	DisplayName      string
	Enabled          bool
	APIKeyConfigured bool
	MaskedKey        string
	DefaultModel     string
	UpdatedAt        time.Time
	TodayTasks       int64
	RunningTasks     int64
	FailedTasks      int64
}

type VideoDashboard struct {
	TodayTasks      int64
	SuccessRate     float64
	FailedTasks     int64
	QueuedTasks     int64
	RunningTasks    int64
	ProviderStatus  []VideoProviderStatus
	RecentFailures  []*VideoTask
	RecentSuccesses []*VideoTask
	UsageOverview   []VideoUsageSummary
}

type VideoProviderCreateParams struct {
	Provider           string
	DisplayName        string
	Enabled            bool
	APIKey             string
	BaseURL            string
	DefaultModel       string
	RateLimitPerMinute int
	Metadata           map[string]any
}

type VideoProviderUpdateParams struct {
	DisplayName        *string
	Enabled            *bool
	APIKey             *string
	BaseURL            *string
	DefaultModel       *string
	RateLimitPerMinute *int
	Metadata           *map[string]any
}

type VideoProviderTestResult struct {
	Provider         string         `json:"provider"`
	Configured       bool           `json:"configured"`
	Reachable        bool           `json:"reachable"`
	Message          string         `json:"message"`
	NormalizedStatus string         `json:"normalized_status"`
	PayloadPreview   map[string]any `json:"payload_preview,omitempty"`
}

type VideoTaskCreateParams struct {
	ProviderAccountID int64
	TaskType          string
	Model             string
	Prompt            string
	NegativePrompt    string
	ReferenceImageURL string
	ReferenceVideoURL string
	AspectRatio       string
	Duration          int
	Resolution        string
	CreatedBy         int64
}

type VideoTaskListParams struct {
	Page      int
	PageSize  int
	Status    string
	Provider  string
	CreatedBy int64
	IsAdmin   bool
}

type VideoGatewayRepository interface {
	CreateProviderAccount(ctx context.Context, account *VideoProviderAccount) error
	GetProviderAccount(ctx context.Context, id int64) (*VideoProviderAccount, error)
	ListProviderAccounts(ctx context.Context) ([]*VideoProviderAccount, error)
	UpdateProviderAccount(ctx context.Context, account *VideoProviderAccount) error

	CreateTask(ctx context.Context, task *VideoTask) error
	GetTask(ctx context.Context, id int64) (*VideoTask, error)
	ListTasks(ctx context.Context, params VideoTaskListParams) ([]*VideoTask, int64, error)
	ListRunnableTasks(ctx context.Context, limit int) ([]*VideoTask, error)
	UpdateTask(ctx context.Context, task *VideoTask) error

	AddTaskEvent(ctx context.Context, event *VideoTaskEvent) error
	ListTaskEvents(ctx context.Context, taskID int64, limit int) ([]*VideoTaskEvent, error)
	InsertUsageLog(ctx context.Context, task *VideoTask) error

	CountTasksSince(ctx context.Context, since time.Time) (map[string]int64, error)
	CountProviderTasksSince(ctx context.Context, since time.Time) (map[string]map[string]int64, error)
	ListRecentTasksByStatus(ctx context.Context, status string, limit int) ([]*VideoTask, error)
	UsageSummarySince(ctx context.Context, since time.Time) ([]VideoUsageSummary, error)
}

func IsTerminalVideoStatus(status string) bool {
	switch status {
	case VideoStatusSucceeded, VideoStatusFailed, VideoStatusCancelled:
		return true
	default:
		return false
	}
}
