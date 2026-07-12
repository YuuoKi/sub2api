//go:build realsmoke

package service

import "github.com/Wei-Shaw/sub2api/internal/reviewguard"

type realReviewKind = reviewguard.Kind

const (
	realReviewImage = reviewguard.Image
	realReviewVideo = reviewguard.Video

	realReviewMaxImages = reviewguard.MaxImages
	realReviewMaxVideos = reviewguard.MaxVideos
	realReviewMaxCNY    = reviewguard.MaxCNY
)

type realReviewSessionState = reviewguard.State

type realReviewSessionGuard struct {
	guard *reviewguard.Guard
}

func newRealReviewSessionGuard(statePath string, enabled bool) *realReviewSessionGuard {
	return &realReviewSessionGuard{guard: reviewguard.New(statePath, enabled)}
}

func (g *realReviewSessionGuard) ReserveBefore(kind realReviewKind, estimatedCNY float64, call func()) error {
	return g.guard.ReserveBefore(kind, estimatedCNY, call)
}

func (g *realReviewSessionGuard) Reserve(kind realReviewKind, estimatedCNY float64) error {
	return g.guard.Reserve(kind, estimatedCNY)
}

func (g *realReviewSessionGuard) loadStateForTest() (realReviewSessionState, error) {
	return g.guard.LoadState()
}
