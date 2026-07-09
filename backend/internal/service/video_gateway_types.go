package service

import (
	"context"
	"fmt"
	"log/slog"
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

	VideoRouteStrategyLeastInflight = "least_inflight"
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
	ErrVideoMockUnavailable  = infraerrors.ServiceUnavailable("VIDEO_MOCK_PROVIDER_UNAVAILABLE", "mock video provider is unavailable")
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
	PlainAPIKey         string `json:"-"` // decrypted transient credential; never serialize
	PlainAccessKey      string `json:"-"` // Kling AK (transient); never serialize
	PlainSecretKey      string `json:"-"` // Kling SK (transient); never serialize
	APIKeyConfigured    bool
	APIKeyDecryptFailed bool
	AuthMode            string // "bearer" | "kling_aksk" (response hint; never a secret)
	MaskedKey           string
	BaseURL             string
	DefaultModel        string
	RateLimitPerMinute  int
	Metadata            map[string]any
	KeyStatus           string
	HealthStatus        string
	DiagnosticType      string
	SuggestedAction     string
	Priority            int
	CurrentInflight     int64
	TodayTasks          int64
	TodayFailures       int64
	LastError           string
	LastTestAt          *time.Time
	RouteAvailable      bool
	RouteSkipReason     string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// String implements fmt.Stringer so that the transient plaintext credentials are
// never rendered by %v/%+v/%s formatting (defensive bottom: PlainAPIKey /
// PlainAccessKey / PlainSecretKey must not land in logs, panics or dumps).
func (a VideoProviderAccount) String() string {
	return fmt.Sprintf("VideoProviderAccount{ID:%d, Provider:%q, DisplayName:%q, Enabled:%t, APIKeyConfigured:%t, AuthMode:%q, MaskedKey:%q, PlainAPIKey:[REDACTED], PlainAccessKey:[REDACTED], PlainSecretKey:[REDACTED]}",
		a.ID, a.Provider, a.DisplayName, a.Enabled, a.APIKeyConfigured, a.AuthMode, a.MaskedKey)
}

// GoString implements fmt.GoStringer so the %#v verb (Go-syntax dump) also masks
// the plaintext credentials instead of printing every struct field verbatim.
func (a VideoProviderAccount) GoString() string {
	return a.String()
}

// LogValue implements slog.LogValuer so that structured logging of the account
// (e.g. slog.Any("account", acc)) never exposes plaintext credentials.
func (a VideoProviderAccount) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Int64("id", a.ID),
		slog.String("provider", a.Provider),
		slog.String("display_name", a.DisplayName),
		slog.Bool("enabled", a.Enabled),
		slog.Bool("api_key_configured", a.APIKeyConfigured),
		slog.String("auth_mode", a.AuthMode),
		slog.String("masked_key", a.MaskedKey),
		slog.String("plain_api_key", "[REDACTED]"),
		slog.String("plain_access_key", "[REDACTED]"),
		slog.String("plain_secret_key", "[REDACTED]"),
	)
}

type VideoTask struct {
	ID                  int64
	ProviderAccountID   int64
	ProviderAccountName string
	Provider            string
	Model               string
	TaskType            string
	Prompt              string
	NegativePrompt      string
	ReferenceImageURL   string
	ReferenceVideoURL   string
	Content             []VideoTaskContentItem
	HasVideoInput       bool
	AspectRatio         string
	Duration            int
	Resolution          string
	GenerateAudio       *bool
	Watermark           *bool
	CameraFixed         *bool
	ReturnLastFrame     *bool
	Status              string
	UpstreamTaskID      string
	// UpstreamVideoID is the provider asset id for a generated video (e.g. Kling
	// task_result.videos[].id). Required for video-extend; distinct from UpstreamTaskID.
	UpstreamVideoID string
	// AudioID is an optional Kling avatar audio asset id (mutually exclusive with
	// content[] audio_url → sound_file on the wire).
	AudioID           string
	ResultURL         string
	UsageTotalTokens  *int64
	ActualResolution  string
	ActualDuration    *int
	LastFrameURL      string
	ErrorMessage      string
	CostEstimate      float64
	Currency          string
	PricingSource     string
	PricingVersion    string
	PollCount         int
	LocalAssetPath    string
	LocalAssetSavedAt *time.Time
	CreatedBy         int64
	CreatedByEmail    string
	CreatedByName     string
	CreatedAt         time.Time
	UpdatedAt         time.Time
	CompletedAt       *time.Time
}

type VideoTaskContentItem struct {
	Type string `json:"type"`
	Role string `json:"role,omitempty"`
	URL  string `json:"url,omitempty"`
	Text string `json:"text,omitempty"`
	// VideoID is an optional Kling video asset id for extend (content metadata).
	VideoID string `json:"video_id,omitempty"`
	// AudioID is an optional Kling audio asset id for avatar (content metadata).
	AudioID string `json:"audio_id,omitempty"`
	// Metadata carries optional provider-specific passthrough keys (e.g. omni
	// refer_type / keep_original_sound). Unknown keys are ignored by adapters.
	Metadata map[string]any `json:"metadata,omitempty"`
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
	KeyStatus        string
	HealthStatus     string
	DiagnosticType   string
	SuggestedAction  string
	RouteAvailable   bool
	RouteSkipReason  string
	Priority         int
	CurrentInflight  int64
	LastError        string
	LastTestAt       *time.Time
	UpdatedAt        time.Time
	TodayTasks       int64
	RunningTasks     int64
	FailedTasks      int64
}

type VideoHealthDiagnostic struct {
	Provider        string
	DisplayName     string
	RouteAccount    string
	KeyStatus       string
	LastTestAt      *time.Time
	ExceptionType   string
	ImpactTasks     int64
	RecentError     string
	SuggestedAction string
	Status          string
}

type VideoDashboard struct {
	TodayTasks        int64
	SuccessRate       float64
	FailedTasks       int64
	QueuedTasks       int64
	RunningTasks      int64
	ProviderStatus    []VideoProviderStatus
	HealthDiagnostics []VideoHealthDiagnostic
	RecentFailures    []*VideoTask
	RecentSuccesses   []*VideoTask
	UsageOverview     []VideoUsageSummary
}

type VideoProviderCreateParams struct {
	Provider           string
	DisplayName        string
	Enabled            bool
	APIKey             string
	AccessKey          string // Kling AK; ignored for seedance/mock
	SecretKey          string // Kling SK; ignored for seedance/mock
	BaseURL            string
	DefaultModel       string
	RateLimitPerMinute int
	Metadata           map[string]any
}

type VideoProviderUpdateParams struct {
	DisplayName        *string
	Enabled            *bool
	APIKey             *string
	AccessKey          *string // nil/empty = keep existing (Kling)
	SecretKey          *string // nil/empty = keep existing (Kling)
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
	ProviderAccountID                      int64
	TaskType                               string
	Model                                  string
	Prompt                                 string
	NegativePrompt                         string
	ReferenceImageURL                      string
	ReferenceVideoURL                      string
	UpstreamVideoID                        string // Kling extend: official video_id
	AudioID                                string // Kling avatar: official audio_id (xor sound_file)
	Content                                []VideoTaskContentItem
	AspectRatio                            string
	Duration                               int
	Resolution                             string
	GenerateAudio                          *bool
	Watermark                              *bool
	CameraFixed                            *bool
	ReturnLastFrame                        *bool
	CreatedBy                              int64
	EnforceRealProviderTrial               bool // JWT user paths: seedance requires daily trial + smoke gate
	RequireSeedanceProductionAuthorization bool // Admin production path: seedance requires provider metadata production_authorized=true
	SafeDemoOnly                           bool // Drama safe demo: route only to mock provider
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
	CreateDailyTrialTask(ctx context.Context, task *VideoTask, provider string, createdBy int64, trialDate time.Time) (bool, error)
	GetTask(ctx context.Context, id int64) (*VideoTask, error)
	ListTasks(ctx context.Context, params VideoTaskListParams) ([]*VideoTask, int64, error)
	ListDramaTasks(ctx context.Context, params VideoTaskListParams, filters map[string]string) ([]*VideoTask, int64, error)
	ListRunnableTasks(ctx context.Context, limit int) ([]*VideoTask, error)
	ListUnchargedSucceededVideoTasks(ctx context.Context, limit int) ([]*VideoTask, error)
	ClaimTaskForSubmit(ctx context.Context, taskID int64) (bool, error)
	UpdateTask(ctx context.Context, task *VideoTask) error
	SetTaskLocalAsset(ctx context.Context, taskID int64, path string, savedAt time.Time) error
	ClearTaskLocalAsset(ctx context.Context, taskID int64) error
	ListExpiredLocalAssets(ctx context.Context, olderThan time.Time, limit int) ([]*VideoTask, error)

	AddTaskEvent(ctx context.Context, event *VideoTaskEvent) error
	ListTaskEvents(ctx context.Context, taskID int64, limit int) ([]*VideoTaskEvent, error)
	InsertUsageLog(ctx context.Context, task *VideoTask) error
	ClaimVideoBalanceCharge(ctx context.Context, taskID int64) (time.Time, bool, error)
	ClearVideoBalanceChargeIfClaimedAt(ctx context.Context, taskID int64, claimedAt time.Time) (bool, error)

	CountTasksSince(ctx context.Context, since time.Time) (map[string]int64, error)
	CountProviderTasksSince(ctx context.Context, since time.Time) (map[string]map[string]int64, error)
	ProviderAccountTaskStatsSince(ctx context.Context, since time.Time) (map[int64]VideoProviderRuntimeStats, error)
	ListRecentTasksByStatus(ctx context.Context, status string, limit int) ([]*VideoTask, error)
	UsageSummarySince(ctx context.Context, since time.Time) ([]VideoUsageSummary, error)
}

type VideoProviderRuntimeStats struct {
	TodayTasks      int64
	CurrentInflight int64
	TodayFailures   int64
	LastError       string
	LastErrorAt     *time.Time
}

func IsTerminalVideoStatus(status string) bool {
	switch status {
	case VideoStatusSucceeded, VideoStatusFailed, VideoStatusCancelled:
		return true
	default:
		return false
	}
}
