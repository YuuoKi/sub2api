package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func seedDramaTaskWithContext(t *testing.T, repo *memoryVideoGatewayRepo, createdBy int64, employeeAlias string) int64 {
	t.Helper()
	providerID := repo.seedMockProvider()
	taskID := repo.nextTaskID
	repo.nextTaskID++
	now := time.Now().UTC().Add(time.Duration(taskID) * time.Second)
	task := &VideoTask{
		ID:                taskID,
		ProviderAccountID: providerID,
		Provider:          VideoProviderMock,
		Model:             "mock-video-v1",
		TaskType:          VideoTaskTypeTextToVideo,
		Status:            VideoStatusQueued,
		CreatedBy:         createdBy,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	repo.tasks[taskID] = cloneVideoTask(task)
	require.NoError(t, repo.AddTaskEvent(context.Background(), &VideoTaskEvent{
		VideoTaskID: taskID,
		EventType:   "drama_context",
		Message:     "test drama context",
		Payload: map[string]any{
			"employee_alias":    employeeAlias,
			"selected_provider": "safe_demo_provider",
			"selected_model":    "mock-video-v1",
			"selected_mode":     "text-to-video",
		},
	}))
	return taskID
}

func TestListDramaTasks_FilteredPagination(t *testing.T) {
	repo := newMemoryVideoGatewayRepo()
	svc := NewVideoGatewayService(repo, noopVideoKeyEncryptor{}, nil)
	ctx := context.Background()

	for range 5 {
		seedDramaTaskWithContext(t, repo, 101, "E001")
	}
	for range 3 {
		seedDramaTaskWithContext(t, repo, 101, "E002")
	}

	filters := map[string]string{"employee_alias": "E001"}
	baseParams := VideoTaskListParams{PageSize: 2, CreatedBy: 101, IsAdmin: true}

	page1, total, err := svc.ListDramaTasks(ctx, mergeVideoListPage(baseParams, 1), filters)
	require.NoError(t, err)
	require.Equal(t, int64(5), total, "total must count all filter matches, not just current page")
	require.Len(t, page1, 2)

	page2, total2, err := svc.ListDramaTasks(ctx, mergeVideoListPage(baseParams, 2), filters)
	require.NoError(t, err)
	require.Equal(t, int64(5), total2)
	require.Len(t, page2, 2)

	page3, total3, err := svc.ListDramaTasks(ctx, mergeVideoListPage(baseParams, 3), filters)
	require.NoError(t, err)
	require.Equal(t, int64(5), total3)
	require.Len(t, page3, 1)
}

func mergeVideoListPage(p VideoTaskListParams, page int) VideoTaskListParams {
	p.Page = page
	return p
}

func seedAuthorizedKlingProvider(t *testing.T, svc *VideoGatewayService) *VideoProviderAccount {
	t.Helper()
	account, err := svc.CreateProviderAccount(context.Background(), VideoProviderCreateParams{
		Provider:     VideoProviderKling,
		DisplayName:  "Kling Production",
		Enabled:      true,
		AccessKey:    "ak-drama-real-111122223333",
		SecretKey:    "sk-drama-real-444455556666",
		DefaultModel: "kling-2.6-pro",
		Metadata: map[string]any{
			"production_authorized": true,
		},
	})
	require.NoError(t, err)
	require.True(t, account.RouteAvailable, "authorized kling account must be route-available")
	return account
}

func TestRecommendDramaProvider_UnauthorizedFallsBackToKlingSafeDemo(t *testing.T) {
	repo := newMemoryVideoGatewayRepo()
	repo.seedMockProvider()
	svc := NewVideoGatewayService(repo, noopVideoKeyEncryptor{}, nil)

	rec, err := svc.RecommendDramaProvider(context.Background(), DramaProviderRecommendParams{
		DramaType: "真人短剧",
		SceneType: "情绪爆发",
		RequestedEngineCapabilities: DramaEngineCapabilityRequest{
			SupportsRealPerson:   true,
			SupportsImageToVideo: true,
		},
	})
	require.NoError(t, err)
	require.Equal(t, "kling_safe_demo", rec.SelectedProvider)
	require.Equal(t, "kling_safe_demo", rec.EngineProfileID)
	require.True(t, rec.SafeDemoMode)
	require.True(t, rec.NeedsRealValidation)
}

func TestRecommendDramaProvider_AuthorizedUsesKlingRealProfile(t *testing.T) {
	repo := newMemoryVideoGatewayRepo()
	repo.seedMockProvider()
	svc := NewVideoGatewayService(repo, noopVideoKeyEncryptor{}, nil)
	seedAuthorizedKlingProvider(t, svc)

	rec, err := svc.RecommendDramaProvider(context.Background(), DramaProviderRecommendParams{
		DramaType: "真人短剧",
		SceneType: "情绪爆发",
		RequestedEngineCapabilities: DramaEngineCapabilityRequest{
			SupportsRealPerson:   true,
			SupportsImageToVideo: true,
		},
	})
	require.NoError(t, err)
	require.Equal(t, "kling_real", rec.SelectedProvider)
	require.Equal(t, "kling_real_image_to_video", rec.EngineProfileID)
	require.Equal(t, "kling-2.6-pro", rec.SelectedModel)
	require.False(t, rec.SafeDemoMode)
	require.False(t, rec.NeedsRealValidation)
}

func TestRecommendDramaProvider_AuthorizedTextToVideoKeepsOfficialMode(t *testing.T) {
	repo := newMemoryVideoGatewayRepo()
	repo.seedMockProvider()
	svc := NewVideoGatewayService(repo, noopVideoKeyEncryptor{}, nil)
	seedAuthorizedKlingProvider(t, svc)

	rec, err := svc.RecommendDramaProvider(context.Background(), DramaProviderRecommendParams{
		DramaType: "真人短剧",
		SceneType: "情绪爆发",
		RequestedEngineCapabilities: DramaEngineCapabilityRequest{
			SupportsRealPerson: true,
		},
	})
	require.NoError(t, err)
	require.Equal(t, "kling_real", rec.SelectedProvider)
	require.Equal(t, "kling_real_text_to_video", rec.EngineProfileID)
	require.Equal(t, "text-to-video", rec.SelectedMode)
	require.False(t, rec.SafeDemoMode)
}

func TestCreateDramaTask_AuthorizedRoutesToRealKlingAccount(t *testing.T) {
	repo := newMemoryVideoGatewayRepo()
	repo.seedMockProvider()
	svc := NewVideoGatewayService(repo, noopVideoKeyEncryptor{}, nil)
	kling := seedAuthorizedKlingProvider(t, svc)

	task, err := svc.CreateDramaTask(context.Background(), DramaTaskCreateParams{
		EmployeeAlias: "E001",
		APIClientID:   "internal_tool_001",
		ProjectID:     "drama_project_real_001",
		DramaType:     "真人短剧",
		SceneType:     "情绪爆发",
		ShotRole:      "女主特写反应",
		Prompt:        "女主在雨夜街边听到男主离开的消息，缓慢抬头。",
		RequestedEngineCapabilities: DramaEngineCapabilityRequest{
			SupportsRealPerson:   true,
			SupportsImageToVideo: true,
		},
		DurationSeconds: 5,
		AspectRatio:     "9:16",
		CreatedBy:       101,
	})
	require.NoError(t, err)
	require.Equal(t, "kling_real", task.SelectedProvider)
	require.Equal(t, "kling-2.6-pro", task.SelectedModel)
	require.Equal(t, "image-to-video", task.SelectedMode)

	stored, err := repo.GetTask(context.Background(), 1)
	require.NoError(t, err)
	require.Equal(t, VideoProviderKling, stored.Provider)
	require.Equal(t, kling.ID, stored.ProviderAccountID)
	require.Equal(t, VideoTaskTypeImageToVideo, stored.TaskType)
}

func TestCreateDramaTask_UnauthorizedStillUsesSafeDemo(t *testing.T) {
	repo := newMemoryVideoGatewayRepo()
	repo.seedMockProvider()
	svc := NewVideoGatewayService(repo, noopVideoKeyEncryptor{}, nil)

	task, err := svc.CreateDramaTask(context.Background(), DramaTaskCreateParams{
		EmployeeAlias: "E001",
		APIClientID:   "internal_tool_001",
		ProjectID:     "drama_project_demo_001",
		DramaType:     "真人短剧",
		SceneType:     "情绪爆发",
		Prompt:        "女主在雨夜街边听到男主离开的消息，缓慢抬头。",
		RequestedEngineCapabilities: DramaEngineCapabilityRequest{
			SupportsRealPerson:   true,
			SupportsImageToVideo: true,
		},
		DurationSeconds: 5,
		AspectRatio:     "9:16",
		CreatedBy:       101,
	})
	require.NoError(t, err)
	require.Equal(t, "kling_safe_demo", task.SelectedProvider)

	stored, err := repo.GetTask(context.Background(), 1)
	require.NoError(t, err)
	require.Equal(t, VideoProviderMock, stored.Provider)
}

func TestCreateDramaTask_AuthorizedTextToVideoDispatchesOfficialT2V(t *testing.T) {
	repo := newMemoryVideoGatewayRepo()
	repo.seedMockProvider()
	svc := NewVideoGatewayService(repo, noopVideoKeyEncryptor{}, nil)
	kling := seedAuthorizedKlingProvider(t, svc)

	task, err := svc.CreateDramaTask(context.Background(), DramaTaskCreateParams{
		EmployeeAlias: "E001",
		APIClientID:   "internal_tool_001",
		ProjectID:     "drama_project_t2v_001",
		DramaType:     "真人短剧",
		SceneType:     "情绪爆发",
		Prompt:        "女主在雨夜街边听到男主离开的消息，缓慢抬头。",
		RequestedEngineCapabilities: DramaEngineCapabilityRequest{
			SupportsRealPerson: true,
		},
		DurationSeconds: 5,
		AspectRatio:     "9:16",
		CreatedBy:       101,
	})
	require.NoError(t, err)
	require.Equal(t, "kling_real", task.SelectedProvider)
	require.Equal(t, "text-to-video", task.SelectedMode)

	stored, err := repo.GetTask(context.Background(), 1)
	require.NoError(t, err)
	require.Equal(t, VideoProviderKling, stored.Provider)
	require.Equal(t, kling.ID, stored.ProviderAccountID)
	require.Equal(t, VideoTaskTypeTextToVideo, stored.TaskType)
}

func TestDramaEngineCapabilityMatrix_MarksKlingRealVerifiedWhenAuthorized(t *testing.T) {
	repo := newMemoryVideoGatewayRepo()
	repo.seedMockProvider()
	svc := NewVideoGatewayService(repo, noopVideoKeyEncryptor{}, nil)
	seedAuthorizedKlingProvider(t, svc)

	profiles := svc.DramaEngineCapabilityMatrix(context.Background())
	var realI2V, realT2V, safeDemo *DramaEngineProfile
	for i := range profiles {
		switch profiles[i].ID {
		case "kling_real_image_to_video":
			realI2V = &profiles[i]
		case "kling_real_text_to_video":
			realT2V = &profiles[i]
		case "kling_safe_demo":
			safeDemo = &profiles[i]
		}
	}
	require.NotNil(t, realI2V)
	require.NotNil(t, realT2V)
	require.NotNil(t, safeDemo)
	require.True(t, realI2V.RealProviderVerified)
	require.True(t, realT2V.RealProviderVerified)
	require.False(t, safeDemo.RealProviderVerified)
	require.True(t, safeDemo.InternalSafeModeOnly)
}
