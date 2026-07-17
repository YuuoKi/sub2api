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

	VideoReservationReserved = "reserved"
	VideoReservationReleased = "released"
	VideoReservationCaptured = "captured"

	VideoSettlementRelease         = "release"
	VideoSettlementCaptureActual   = "capture_actual"
	VideoSettlementCaptureReserved = "capture_reserved"

	VideoPricingSourceConfig                         = "config.video_gateway"
	VideoPricingVersionSeedanceCompletionTokensUSDV1 = "seedance_completion_tokens_usd_v1"
)

var (
	ErrVideoTaskNotFound           = errors.New("video task not found")
	ErrVideoTaskTerminalConflict   = errors.New("video task terminal status conflicts with requested status")
	ErrVideoTaskForbidden          = errors.New("video task is outside employee scope")
	ErrVideoProviderNotFound       = errors.New("video provider not found")
	ErrVideoCancelConflict         = errors.New("video task cannot be cancelled after dispatch started")
	ErrVideoPricingSnapshotInvalid = errors.New("video pricing snapshot is invalid")
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
	ID                   int64      `json:"id"`
	GroupID              int64      `json:"group_id"`
	GroupName            string     `json:"group_name,omitempty"`
	Provider             string     `json:"provider"`
	DisplayName          string     `json:"display_name"`
	Enabled              bool       `json:"enabled"`
	EncryptedAPIKey      string     `json:"-"`
	MaskedKey            string     `json:"masked_key"`
	BaseURL              string     `json:"-"`
	DefaultModel         string     `json:"default_model"`
	APIKeyConfigured     bool       `json:"api_key_configured"`
	TinyRealAuthorizedAt *time.Time `json:"tiny_real_authorized_at,omitempty"`
	TinyRealAuthorizedBy int64      `json:"tiny_real_authorized_by,omitempty"`
	TinyRealConsumedAt   *time.Time `json:"tiny_real_consumed_at,omitempty"`
}

type VideoTask struct {
	ID                                   int64
	APIKeyID                             int64
	GroupID                              int64
	ProviderAccountID                    int64
	Provider                             string
	Model                                string
	TaskType                             string
	Prompt                               string
	Status                               string
	UpstreamTaskID                       string
	ResultURL                            string
	LastFrameURL                         string
	DurationSeconds                      int
	Resolution                           string
	UsageTotalTokens                     *int64
	CostAmount                           float64
	Currency                             string
	PricingSource                        string
	PricingVersion                       string
	PricingCNYPerMillionCompletionTokens *float64
	PricingUSDCNYExchangeRate            *float64
	PricingMaximumCNY                    *float64
	RealDispatchCount                    int
	ProviderErrorCode                    string
	ProviderErrorMessage                 string
	ErrorMessage                         string
	CreationKey                          string
	Version                              int64
	DispatchState                        string
	CreatedBy                            int64
	CreatedAt                            time.Time
	UpdatedAt                            time.Time
	CompletedAt                          *time.Time
	ReservedCostUSD                      float64
	ReservationState                     string
	ReservedAt                           *time.Time
	ReservationWindow5h                  *time.Time
	ReservationWindow1d                  *time.Time
	ReservationWindow7d                  *time.Time
	ProviderActualCostUSD                float64
	UpstreamModel                        *string
	UpstreamDurationSeconds              *int
	UpstreamResolution                   *string
	BillingModel                         *string
	BillingDurationSeconds               *int
	BillingResolution                    *string
	BalanceBeforeUSD                     *float64
	BalanceAfterUSD                      *float64
	BalanceDeltaUSD                      *float64
	AuthorizationConsumedAt              *time.Time
	AuthorizationConsumedBy              *int64
}

type VideoTaskFinalization struct {
	TaskID                  int64
	ExpectedVersion         int64
	Status                  string
	ResultURL               string
	LastFrameURL            string
	UsageTotalTokens        *int64
	CostAmount              float64
	Currency                string
	ProviderErrorCode       string
	ProviderErrorMessage    string
	ErrorMessage            string
	CompletedAt             time.Time
	Settlement              string
	ProviderActualCostUSD   float64
	UpstreamModel           *string
	UpstreamDurationSeconds *int
	UpstreamResolution      *string
	BillingModel            *string
	BillingDurationSeconds  *int
	BillingResolution       *string
}

type VideoTaskFinalizationResult struct {
	Applied    bool
	Idempotent bool
	Status     string
	Version    int64
}

type VideoGatewayRepository interface {
	ReserveAndCreateTask(context.Context, *VideoTask, float64) error
	GetTask(context.Context, int64) (*VideoTask, error)
	GetTaskForScope(context.Context, int64, VideoTaskScope) (*VideoTask, error)
	CancelTaskForScope(context.Context, int64, VideoTaskScope) (*VideoTask, error)
	ClaimRunnableTasks(context.Context, int, time.Duration) ([]*VideoTask, error)
	FinalizeTask(context.Context, VideoTaskFinalization) (VideoTaskFinalizationResult, error)
}

type VideoGatewayRuntimeRepository interface {
	VideoGatewayRepository
	ListEnabledVideoProviders(context.Context, int64) ([]VideoProviderAccount, error)
	GetVideoProvider(context.Context, int64, int64) (*VideoProviderAccount, error)
	BeginRealDispatch(context.Context, int64, int64) (bool, error)
	MarkVideoSubmitted(context.Context, int64, int64, string) error
	UpdateVideoProgress(context.Context, int64, int64, string) error
}
