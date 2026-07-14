package service

import (
	"net/http"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// Product execution modes for employee create paths.
const (
	ExecutionModeMock         = "mock"
	ExecutionModeReviewReal   = "review_real"
	ExecutionModeInternalReal = "internal_real"
)

var (
	ErrExecutionModeInvalid = infraerrors.New(
		http.StatusBadRequest,
		"EXECUTION_MODE_INVALID",
		"execution_mode must be mock, review_real, or internal_real",
	)
	ErrExecutionModeProviderAccountForbidden = infraerrors.New(
		http.StatusForbidden,
		"PROVIDER_ACCOUNT_ID_FORBIDDEN",
		"普通员工不能指定 Provider 账号；请使用 execution_mode 选择试跑或复核模式",
	)
	ErrReviewRealSessionDisabled = infraerrors.New(
		http.StatusForbidden,
		"REAL_REVIEW_SESSION_DISABLED",
		"真实复核会话未启用。请联系管理员开启复核会话后再试，或改用免费试跑",
	)
	ErrReviewRealAccountUnavailable = infraerrors.New(
		http.StatusForbidden,
		"REVIEW_REAL_ACCOUNT_UNAVAILABLE",
		"真实复核通道未就绪。请联系管理员完成复核凭证引导，或改用免费试跑",
	)
	ErrInternalRealChannelMissing = infraerrors.New(
		http.StatusBadRequest,
		"INTERNAL_REAL_CHANNEL_MISSING",
		"请先配置正式内部通道",
	)
	ErrInternalRealPolicyDenied = infraerrors.New(
		http.StatusForbidden,
		"INTERNAL_REAL_POLICY_DENIED",
		"内部真实调用未被策略允许。请联系管理员开通策略或改用免费试跑",
	)
	ErrInternalRealBudgetExceeded = infraerrors.New(
		http.StatusPaymentRequired,
		"INTERNAL_REAL_BUDGET_EXCEEDED",
		"内部真实调用预算不足。请联系管理员调整额度或改用免费试跑",
	)
)

// NormalizeExecutionMode returns a canonical mode. Empty defaults to mock (免费试跑).
func NormalizeExecutionMode(raw string) (string, error) {
	mode := strings.TrimSpace(strings.ToLower(raw))
	if mode == "" {
		return ExecutionModeMock, nil
	}
	switch mode {
	case ExecutionModeMock, ExecutionModeReviewReal, ExecutionModeInternalReal:
		return mode, nil
	default:
		return "", ErrExecutionModeInvalid
	}
}

func isReviewOnlyVideoAccount(account *VideoProviderAccount) bool {
	if account == nil {
		return false
	}
	return videoMetadataBool(account.Metadata, "review_only")
}

// IsReviewOnlyVideoAccount reports whether a video provider account is temporary review-only.
func IsReviewOnlyVideoAccount(account *VideoProviderAccount) bool {
	return isReviewOnlyVideoAccount(account)
}

func videoMetadataBool(metadata map[string]any, key string) bool {
	if metadata == nil {
		return false
	}
	value, ok := metadata[key]
	if !ok || value == nil {
		return false
	}
	switch v := value.(type) {
	case bool:
		return v
	case string:
		switch strings.TrimSpace(strings.ToLower(v)) {
		case "1", "true", "yes", "on":
			return true
		default:
			return false
		}
	default:
		return false
	}
}

func isReviewOnlyImageAccount(account *Account) bool {
	if account == nil {
		return false
	}
	if truthyAny(account.Extra["review_only"]) {
		return true
	}
	if truthyAny(account.Credentials["review_only"]) {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(account.GetCredential("review_only")), "true")
}

func truthyAny(raw any) bool {
	switch v := raw.(type) {
	case nil:
		return false
	case bool:
		return v
	case string:
		switch strings.TrimSpace(strings.ToLower(v)) {
		case "1", "true", "yes", "on":
			return true
		default:
			return false
		}
	default:
		return false
	}
}
