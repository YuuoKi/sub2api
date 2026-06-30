package service

import (
	"context"
	"errors"
	"strings"
	"time"
)

const defaultSkillEngineMaxInputBytes = 64 * 1024

var (
	ErrSkillEngineDisabled       = errors.New("skill engine disabled")
	ErrSkillEngineEmptyInput     = errors.New("skill engine input is empty")
	ErrSkillEngineNotImplemented = errors.New("skill engine v0 is not implemented")
)

type SkillEngineStatus string

const (
	SkillEngineStatusDisabled       SkillEngineStatus = "disabled"
	SkillEngineStatusInvalidInput   SkillEngineStatus = "invalid_input"
	SkillEngineStatusNotImplemented SkillEngineStatus = "not_implemented"
)

type SkillEngineConfig struct {
	Enabled       bool
	MaxInputBytes int
}

type SkillEngineInput struct {
	Source           string
	Model            string
	PromptRedacted   string
	ResponseRedacted string
	CapturedAt       time.Time
}

type SkillEngineResult struct {
	Status      SkillEngineStatus
	Reason      string
	InputBytes  int
	GeneratedAt time.Time
}

type SkillEngine struct {
	cfg SkillEngineConfig
	now func() time.Time
}

func NewSkillEngine(cfg SkillEngineConfig) *SkillEngine {
	if cfg.MaxInputBytes <= 0 {
		cfg.MaxInputBytes = defaultSkillEngineMaxInputBytes
	}
	return &SkillEngine{
		cfg: cfg,
		now: time.Now,
	}
}

func (e *SkillEngine) Run(ctx context.Context, input SkillEngineInput) (SkillEngineResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case <-ctx.Done():
		return SkillEngineResult{
			Status: SkillEngineStatusInvalidInput,
			Reason: ctx.Err().Error(),
		}, ctx.Err()
	default:
	}

	if e == nil || !e.cfg.Enabled {
		return SkillEngineResult{
			Status: SkillEngineStatusDisabled,
			Reason: ErrSkillEngineDisabled.Error(),
		}, ErrSkillEngineDisabled
	}

	inputBytes := boundedSkillInputBytes(input, e.cfg.MaxInputBytes)
	if strings.TrimSpace(input.PromptRedacted) == "" && strings.TrimSpace(input.ResponseRedacted) == "" {
		return SkillEngineResult{
			Status:      SkillEngineStatusInvalidInput,
			Reason:      ErrSkillEngineEmptyInput.Error(),
			InputBytes:  inputBytes,
			GeneratedAt: e.now(),
		}, ErrSkillEngineEmptyInput
	}

	return SkillEngineResult{
		Status:      SkillEngineStatusNotImplemented,
		Reason:      ErrSkillEngineNotImplemented.Error(),
		InputBytes:  inputBytes,
		GeneratedAt: e.now(),
	}, ErrSkillEngineNotImplemented
}

func boundedSkillInputBytes(input SkillEngineInput, maxBytes int) int {
	if maxBytes <= 0 {
		maxBytes = defaultSkillEngineMaxInputBytes
	}
	total := len([]byte(input.PromptRedacted)) + len([]byte(input.ResponseRedacted))
	if total > maxBytes {
		return maxBytes
	}
	return total
}
