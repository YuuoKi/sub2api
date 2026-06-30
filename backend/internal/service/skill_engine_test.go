package service

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestSkillEngineDisabledByDefault(t *testing.T) {
	engine := NewSkillEngine(SkillEngineConfig{})

	result, err := engine.Run(context.Background(), SkillEngineInput{
		PromptRedacted: "redacted prompt",
	})

	if !errors.Is(err, ErrSkillEngineDisabled) {
		t.Fatalf("expected ErrSkillEngineDisabled, got %v", err)
	}
	if result.Status != SkillEngineStatusDisabled {
		t.Fatalf("status = %q, want %q", result.Status, SkillEngineStatusDisabled)
	}
}

func TestSkillEngineEnabledRejectsEmptyInput(t *testing.T) {
	engine := NewSkillEngine(SkillEngineConfig{Enabled: true})

	result, err := engine.Run(context.Background(), SkillEngineInput{})

	if !errors.Is(err, ErrSkillEngineEmptyInput) {
		t.Fatalf("expected ErrSkillEngineEmptyInput, got %v", err)
	}
	if result.Status != SkillEngineStatusInvalidInput {
		t.Fatalf("status = %q, want %q", result.Status, SkillEngineStatusInvalidInput)
	}
}

func TestSkillEngineEnabledIsExplicitlyNotImplemented(t *testing.T) {
	engine := NewSkillEngine(SkillEngineConfig{Enabled: true, MaxInputBytes: 10})
	fixedNow := time.Date(2026, 7, 2, 3, 0, 0, 0, time.UTC)
	engine.now = func() time.Time { return fixedNow }

	result, err := engine.Run(context.Background(), SkillEngineInput{
		Source:           "ai_generation_content",
		Model:            "seedance",
		PromptRedacted:   "1234567890",
		ResponseRedacted: "abc",
	})

	if !errors.Is(err, ErrSkillEngineNotImplemented) {
		t.Fatalf("expected ErrSkillEngineNotImplemented, got %v", err)
	}
	if result.Status != SkillEngineStatusNotImplemented {
		t.Fatalf("status = %q, want %q", result.Status, SkillEngineStatusNotImplemented)
	}
	if result.InputBytes != 10 {
		t.Fatalf("input bytes = %d, want cap 10", result.InputBytes)
	}
	if !result.GeneratedAt.Equal(fixedNow) {
		t.Fatalf("generated at = %s, want %s", result.GeneratedAt, fixedNow)
	}
}

func TestSkillEngineRespectsCanceledContext(t *testing.T) {
	engine := NewSkillEngine(SkillEngineConfig{Enabled: true})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result, err := engine.Run(ctx, SkillEngineInput{PromptRedacted: "redacted"})

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if result.Status != SkillEngineStatusInvalidInput {
		t.Fatalf("status = %q, want %q", result.Status, SkillEngineStatusInvalidInput)
	}
}
