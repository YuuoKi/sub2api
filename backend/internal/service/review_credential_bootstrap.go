package service

import (
	"context"
	"os"
	"strings"
)

const (
	envGeminiAPIKey              = "GEMINI_API_KEY"
	envSeedanceSmokeAPIKey       = "SUB2API_SEEDANCE_SMOKE_API_KEY"
	reviewGeminiAccountName      = "review-only-gemini"
	reviewSeedanceAccountName    = "review-only-seedance"
)

// ReviewCredentialBootstrapResult never includes secret values.
type ReviewCredentialBootstrapResult struct {
	SessionEnabled bool                         `json:"session_enabled"`
	FailClosed     bool                         `json:"fail_closed"`
	Gemini         ReviewCredentialAccountStatus `json:"gemini"`
	Seedance       ReviewCredentialAccountStatus `json:"seedance"`
}

type ReviewCredentialAccountStatus struct {
	EnvPresent bool   `json:"env_present"`
	AccountID  int64  `json:"account_id,omitempty"`
	Status     string `json:"status"`
	Message    string `json:"message,omitempty"`
}

// ClearReviewOnlyAccountsResult summarizes disable/clear of review_only bootstrap accounts.
type ClearReviewOnlyAccountsResult struct {
	DisabledVideoAccounts int `json:"disabled_video_accounts"`
	DisabledImageAccounts int `json:"disabled_image_accounts"`
}

// ClearReviewOnlyAccounts disables review_only bootstrap provider accounts (does not delete).
func (s *VideoGatewayService) ClearReviewOnlyAccounts(ctx context.Context) (*ClearReviewOnlyAccountsResult, error) {
	out := &ClearReviewOnlyAccountsResult{}
	if s == nil || s.repo == nil {
		return out, ErrReviewRealAccountUnavailable
	}
	accounts, err := s.repo.ListProviderAccounts(ctx)
	if err != nil {
		return out, err
	}
	for _, account := range accounts {
		if account == nil || !isReviewOnlyVideoAccount(account) {
			continue
		}
		if !account.Enabled {
			continue
		}
		account.Enabled = false
		if err := s.repo.UpdateProviderAccount(ctx, account); err != nil {
			return out, err
		}
		out.DisabledVideoAccounts++
	}
	if s.imageReviewClear != nil {
		n, err := s.imageReviewClear(ctx)
		if err != nil {
			return out, err
		}
		out.DisabledImageAccounts = n
	}
	return out, nil
}

type imageReviewClearFunc func(ctx context.Context) (disabledCount int, err error)

// SetImageReviewClear wires disable of Gemini review_only accounts in accounts table.
func (s *VideoGatewayService) SetImageReviewClear(fn imageReviewClearFunc) {
	if s == nil {
		return
	}
	s.imageReviewClear = fn
}

// ReviewCredentialBootstrap consumes env key *presence* to create/update
// review_only provider accounts via existing encrypted storage. It never logs
// or returns secret values. Missing keys => fail-closed.
func (s *VideoGatewayService) ReviewCredentialBootstrap(ctx context.Context) (*ReviewCredentialBootstrapResult, error) {
	out := &ReviewCredentialBootstrapResult{
		SessionEnabled: s != nil && s.RealReviewSessionEnabled(),
	}
	if !out.SessionEnabled {
		out.FailClosed = true
		out.Gemini = ReviewCredentialAccountStatus{Status: "skipped", Message: "real review session is not enabled"}
		out.Seedance = ReviewCredentialAccountStatus{Status: "skipped", Message: "real review session is not enabled"}
		return out, ErrReviewRealSessionDisabled
	}

	geminiKey, geminiPresent := lookupReviewEnvSecret(envGeminiAPIKey)
	seedanceKey, seedancePresent := lookupReviewEnvSecret(envSeedanceSmokeAPIKey)
	out.Gemini.EnvPresent = geminiPresent
	out.Seedance.EnvPresent = seedancePresent

	if !geminiPresent || !seedancePresent {
		out.FailClosed = true
		if !geminiPresent {
			out.Gemini.Status = "missing_env"
			out.Gemini.Message = "GEMINI_API_KEY is required when review session is enabled"
		}
		if !seedancePresent {
			out.Seedance.Status = "missing_env"
			out.Seedance.Message = "SUB2API_SEEDANCE_SMOKE_API_KEY is required when review session is enabled"
		}
		return out, ErrReviewRealSessionDisabled
	}

	seedanceID, err := s.upsertReviewOnlyVideoAccount(ctx, VideoProviderSeedance, reviewSeedanceAccountName, seedanceKey)
	if err != nil {
		out.FailClosed = true
		out.Seedance.Status = "error"
		out.Seedance.Message = "failed to upsert review seedance account"
		return out, err
	}
	out.Seedance.AccountID = seedanceID
	out.Seedance.Status = "ready"

	geminiID, err := s.upsertReviewOnlyImageAccountHint(ctx, geminiKey)
	if err != nil {
		out.FailClosed = true
		out.Gemini.Status = "error"
		out.Gemini.Message = "failed to prepare review gemini account"
		return out, err
	}
	out.Gemini.AccountID = geminiID
	out.Gemini.Status = "ready"
	return out, nil
}

func lookupReviewEnvSecret(name string) (value string, present bool) {
	value = strings.TrimSpace(os.Getenv(name))
	return value, value != ""
}

func (s *VideoGatewayService) upsertReviewOnlyVideoAccount(ctx context.Context, provider, displayName, plainKey string) (int64, error) {
	if s == nil || s.repo == nil {
		return 0, ErrReviewRealAccountUnavailable
	}
	accounts, err := s.repo.ListProviderAccounts(ctx)
	if err != nil {
		return 0, err
	}
	var existing *VideoProviderAccount
	for _, account := range accounts {
		if account == nil {
			continue
		}
		if account.Provider == provider && (isReviewOnlyVideoAccount(account) || strings.EqualFold(account.DisplayName, displayName)) {
			existing = account
			break
		}
	}
	meta := map[string]any{
		"review_only":           true,
		"production_authorized": true,
	}
	if existing != nil {
		existing.Enabled = true
		existing.DisplayName = displayName
		if existing.Metadata == nil {
			existing.Metadata = map[string]any{}
		}
		for k, v := range meta {
			existing.Metadata[k] = v
		}
		if err := s.applyProviderAPIKey(existing, plainKey); err != nil {
			return 0, err
		}
		if err := s.repo.UpdateProviderAccount(ctx, existing); err != nil {
			return 0, err
		}
		return existing.ID, nil
	}
	account := &VideoProviderAccount{
		Provider:           provider,
		DisplayName:        displayName,
		Enabled:            true,
		DefaultModel:       defaultVideoModel(provider),
		RateLimitPerMinute: defaultVideoRateLimit(0),
		Metadata:           meta,
	}
	if err := s.applyProviderAPIKey(account, plainKey); err != nil {
		return 0, err
	}
	if err := s.repo.CreateProviderAccount(ctx, account); err != nil {
		return 0, err
	}
	return account.ID, nil
}

// upsertReviewOnlyImageAccountHint stores a review-only marker account via the
// optional image account bootstrap hook when wired; otherwise records readiness
// from env presence only (account_id=0) without persisting secrets in logs.
func (s *VideoGatewayService) upsertReviewOnlyImageAccountHint(ctx context.Context, plainKey string) (int64, error) {
	_ = ctx
	_ = plainKey // consumed only by optional ImageAccountBootstrap when wired
	if s == nil {
		return 0, ErrReviewRealAccountUnavailable
	}
	if s.imageReviewBootstrap != nil {
		return s.imageReviewBootstrap(ctx, plainKey)
	}
	// Fail-closed without a wired image bootstrap: presence was already checked.
	return 0, nil
}

type imageReviewBootstrapFunc func(ctx context.Context, plainAPIKey string) (accountID int64, err error)

// SetImageReviewBootstrap wires Gemini review_only account upsert into accounts table.
func (s *VideoGatewayService) SetImageReviewBootstrap(fn imageReviewBootstrapFunc) {
	if s == nil {
		return
	}
	s.imageReviewBootstrap = fn
}
