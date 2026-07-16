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
)

type VideoKeyEncryptor interface {
	Encrypt(plaintext string) (string, error)
	Decrypt(ciphertext string) (string, error)
}

type VideoTask struct {
	ID                int64
	ProviderAccountID int64
	Provider          string
	Model             string
	TaskType          string
	Prompt            string
	Status            string
	ResultURL         string
	ErrorMessage      string
	CreationKey       string
	Version           int64
	DispatchState     string
	CreatedBy         int64
	CreatedAt         time.Time
	UpdatedAt         time.Time
	CompletedAt       *time.Time
}

type VideoTaskFinalization struct {
	TaskID          int64
	ExpectedVersion int64
	Status          string
	ResultURL       string
	ErrorMessage    string
	CompletedAt     time.Time
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
	ClaimRunnableTasks(context.Context, int, time.Duration) ([]*VideoTask, error)
	FinalizeTask(context.Context, VideoTaskFinalization) (VideoTaskFinalizationResult, error)
}
