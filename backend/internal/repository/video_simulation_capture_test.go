package repository

import (
	"context"
	"database/sql/driver"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

type redactedPromptArg struct {
	forbidden []string
	markers   []string
}

func (a redactedPromptArg) Match(v driver.Value) bool {
	s, ok := v.(string)
	if !ok {
		return false
	}
	for _, leak := range a.forbidden {
		if strings.Contains(s, leak) {
			return false
		}
	}
	for _, marker := range a.markers {
		if strings.Contains(s, marker) {
			return true
		}
	}
	return false
}

func TestCaptureTaskLinkedContentRedactsPromptSecrets(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	repo := &videoGatewayRepository{db: db}

	const (
		cnID   = "11010519491231002X"
		opaque = "Ab3Cd6Ef9Gh2Jk5Lm8Np1Qr4"
	)
	rawPrompt := "身份证" + cnID + " api_key=sk-live-" + opaque

	mock.ExpectExec(`(?is)INSERT\s+INTO\s+ai_generation_content`).
		WithArgs(
			"simulation-task-42",
			int64(11),
			int64(7),
			int64(3),
			int64(42),
			service.VideoModelMockVideoV1,
			redactedPromptArg{
				forbidden: []string{cnID, opaque, "sk-live-"},
				markers:   []string{"[ID]", "[已脱敏]", "***"},
			},
			"模拟视频结果",
			len(rawPrompt),
			len("模拟视频结果"),
			false,
			4, // generationRedactionVersion
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.CaptureTaskLinkedContent(context.Background(), &service.VideoTask{
		ID: 42, APIKeyID: 11, CreatedBy: 7, GroupID: 3,
		Model: service.VideoModelMockVideoV1, Prompt: rawPrompt,
	})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListSimulationTasksForOwnerAppliesLimit(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	repo := &videoGatewayRepository{db: db}
	now := time.Now().UTC()

	mock.ExpectQuery(`(?is)LIMIT\s+\$3`).
		WithArgs(int64(7), service.VideoProviderMock, service.VideoSimulationListMaxItems).
		WillReturnRows(videoTaskRows(now).AddRow(
			int64(5), int64(21), int64(22), int64(2), "mock", "mock-video-v1", "text_to_video", "prompt", "{}", "queued",
			"", "", "", 4, "720p", nil, 0, "USD", "internal_simulation", "simulation-v1", nil, nil, nil, 0, "", "", "", "list-mock-5", int64(1), "pending", int64(7),
			now, now, nil, 0, "none", nil, nil, nil, nil, 0, nil, nil, nil, nil, nil, nil, 0, nil, nil, nil, nil, nil, nil,
		))

	tasks, err := repo.ListSimulationTasksForOwner(context.Background(), 7)
	require.NoError(t, err)
	require.Len(t, tasks, 1)
	require.NoError(t, mock.ExpectationsWereMet())
}
