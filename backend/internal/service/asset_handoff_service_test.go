package service

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type assetHandoffVideoRepo struct {
	fakeVideoAdminRepo
	task *VideoTask
}

func (r *assetHandoffVideoRepo) GetVideoTaskAdmin(_ context.Context, id int64) (*VideoTask, error) {
	if r.task == nil || r.task.ID != id {
		return nil, ErrVideoTaskNotFound
	}
	return r.task, nil
}

type fakeAssetInspector struct {
	inspection AssetInspection
	err        error
	urls       []string
}

func (f *fakeAssetInspector) Inspect(_ context.Context, rawURL string) (AssetInspection, error) {
	f.urls = append(f.urls, rawURL)
	return f.inspection, f.err
}

func newSucceededAssetTask() *VideoTask {
	return &VideoTask{
		ID:           501,
		Status:       VideoStatusSucceeded,
		ResultURL:    "https://assets.example.test/result.mp4",
		LastFrameURL: "https://assets.example.test/tail.png",
	}
}

func TestAssetHandoffIssuesOpaqueTicketAndConsumesItOnce(t *testing.T) {
	now := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)
	repo := &assetHandoffVideoRepo{task: newSucceededAssetTask()}
	inspector := &fakeAssetInspector{inspection: AssetInspection{MIME: "video/mp4", SizeBytes: 1024}}
	service := NewAssetHandoffService(repo, inspector, func() time.Time { return now }, strings.NewReader(strings.Repeat("a", 64)))

	issued, err := service.Issue(context.Background(), 9, repo.task.ID, AssetHandoffVideo)
	require.NoError(t, err)
	require.NotEmpty(t, issued.Ticket)
	require.NotContains(t, issued.Ticket, repo.task.ResultURL)
	require.Equal(t, now.Add(5*time.Minute), issued.ExpiresAt)
	require.Len(t, service.tickets, 1)
	for digest, record := range service.tickets {
		require.NotEqual(t, issued.Ticket, digest)
		require.NotContains(t, fmt.Sprint(record), repo.task.ResultURL)
		require.Equal(t, int64(9), record.IssuerID)
	}

	asset, err := service.Consume(context.Background(), issued.Ticket)
	require.NoError(t, err)
	require.Equal(t, repo.task.ResultURL, asset.URL)
	require.Equal(t, "video/mp4", asset.MIME)

	_, err = service.Consume(context.Background(), issued.Ticket)
	require.ErrorIs(t, err, ErrAssetHandoffConsumed)
}

func TestAssetHandoffRejectsExpiredTicket(t *testing.T) {
	now := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)
	repo := &assetHandoffVideoRepo{task: newSucceededAssetTask()}
	inspector := &fakeAssetInspector{inspection: AssetInspection{MIME: "image/png", SizeBytes: 1024}}
	service := NewAssetHandoffService(repo, inspector, func() time.Time { return now }, strings.NewReader(strings.Repeat("b", 64)))
	issued, err := service.Issue(context.Background(), 9, repo.task.ID, AssetHandoffImage)
	require.NoError(t, err)

	now = now.Add(6 * time.Minute)
	_, err = service.Consume(context.Background(), issued.Ticket)
	require.ErrorIs(t, err, ErrAssetHandoffExpired)
}

func TestAssetHandoffRejectsNonSucceededTask(t *testing.T) {
	task := newSucceededAssetTask()
	task.Status = VideoStatusRunning
	service := NewAssetHandoffService(
		&assetHandoffVideoRepo{task: task},
		&fakeAssetInspector{inspection: AssetInspection{MIME: "video/mp4", SizeBytes: 1024}},
		time.Now,
		strings.NewReader(strings.Repeat("c", 64)),
	)

	_, err := service.Issue(context.Background(), 9, task.ID, AssetHandoffVideo)
	require.ErrorIs(t, err, ErrAssetHandoffTaskNotSucceeded)
}

func TestAssetHandoffRejectsMIMEAndSizeOutsideContract(t *testing.T) {
	tests := []struct {
		name       string
		kind       AssetHandoffKind
		inspection AssetInspection
		want       error
	}{
		{name: "video MIME", kind: AssetHandoffVideo, inspection: AssetInspection{MIME: "text/html", SizeBytes: 1024}, want: ErrAssetHandoffInvalidMIME},
		{name: "image MIME", kind: AssetHandoffImage, inspection: AssetInspection{MIME: "image/jpeg", SizeBytes: 1024}, want: ErrAssetHandoffInvalidMIME},
		{name: "too large", kind: AssetHandoffVideo, inspection: AssetInspection{MIME: "video/mp4", SizeBytes: 30*1024*1024 + 1}, want: ErrAssetHandoffTooLarge},
		{name: "unverifiable size", kind: AssetHandoffVideo, inspection: AssetInspection{MIME: "video/mp4", SizeBytes: -1}, want: ErrAssetHandoffUnverifiable},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			task := newSucceededAssetTask()
			service := NewAssetHandoffService(
				&assetHandoffVideoRepo{task: task},
				&fakeAssetInspector{inspection: test.inspection},
				time.Now,
				strings.NewReader(strings.Repeat("d", 64)),
			)
			_, err := service.Issue(context.Background(), 9, task.ID, test.kind)
			require.ErrorIs(t, err, test.want)
		})
	}
}

func TestAssetHandoffRevalidatesTaskBeforeConsumption(t *testing.T) {
	task := newSucceededAssetTask()
	inspector := &fakeAssetInspector{inspection: AssetInspection{MIME: "video/mp4", SizeBytes: 1024}}
	service := NewAssetHandoffService(
		&assetHandoffVideoRepo{task: task}, inspector, time.Now,
		strings.NewReader(strings.Repeat("e", 64)),
	)
	issued, err := service.Issue(context.Background(), 9, task.ID, AssetHandoffVideo)
	require.NoError(t, err)
	task.Status = VideoStatusFailed

	_, err = service.Consume(context.Background(), issued.Ticket)
	require.ErrorIs(t, err, ErrAssetHandoffTaskNotSucceeded)
}

func TestAssetHandoffRemovesExpiredRecordsWhenIssuing(t *testing.T) {
	now := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)
	task := newSucceededAssetTask()
	service := NewAssetHandoffService(
		&assetHandoffVideoRepo{task: task},
		&fakeAssetInspector{inspection: AssetInspection{MIME: "video/mp4", SizeBytes: 1024}},
		func() time.Time { return now },
		bytes.NewReader(append(bytes.Repeat([]byte{'f'}, 32), bytes.Repeat([]byte{'g'}, 32)...)),
	)
	_, err := service.Issue(context.Background(), 9, task.ID, AssetHandoffVideo)
	require.NoError(t, err)
	now = now.Add(6 * time.Minute)
	_, err = service.Issue(context.Background(), 9, task.ID, AssetHandoffVideo)
	require.NoError(t, err)

	require.Len(t, service.tickets, 1)
}
