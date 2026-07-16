//go:build unit

package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type temporaryCredentialAdminServiceStub struct {
	service.AdminService
	createdInput *service.CreateUserInput
	resetUserID  int64
}

func (s *temporaryCredentialAdminServiceStub) CreateUser(_ context.Context, input *service.CreateUserInput) (*service.User, error) {
	s.createdInput = input
	expiresAt := time.Now().UTC().Add(24 * time.Hour)
	return &service.User{
		ID:                 91,
		Email:              input.Email,
		Role:               service.RoleUser,
		Status:             service.StatusActive,
		MustChangePassword: true,
		InitialCredential: &service.InitialCredential{
			TemporaryPassword: "one-time-value",
			ExpiresAt:         expiresAt,
		},
	}, nil
}

func (s *temporaryCredentialAdminServiceStub) ResetUserPassword(_ context.Context, id int64) (*service.InitialCredential, error) {
	s.resetUserID = id
	return &service.InitialCredential{
		TemporaryPassword: "reset-one-time-value",
		ExpiresAt:         time.Now().UTC().Add(24 * time.Hour),
	}, nil
}

func TestAdminUserCreateDoesNotRequireOrAcceptPassword(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stub := &temporaryCredentialAdminServiceStub{}
	handler := NewUserHandler(stub, nil, nil, nil)
	router := gin.New()
	router.POST("/admin/users", handler.Create)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/users", bytes.NewBufferString(`{"email":"employee@example.com"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "employee@example.com", stub.createdInput.Email)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	data := body["data"].(map[string]any)
	require.NotEmpty(t, data["initial_credential"])
}

func TestAdminUserCreateRejectsClientSuppliedPassword(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stub := &temporaryCredentialAdminServiceStub{}
	handler := NewUserHandler(stub, nil, nil, nil)
	router := gin.New()
	router.POST("/admin/users", handler.Create)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/users", bytes.NewBufferString(`{"email":"employee@example.com","password":"must-not-be-accepted"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Nil(t, stub.createdInput)
	require.NotContains(t, w.Body.String(), "must-not-be-accepted")
}

func TestAdminUserUpdateRejectsClientSuppliedPassword(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewUserHandler(&temporaryCredentialAdminServiceStub{}, nil, nil, nil)
	router := gin.New()
	router.PUT("/admin/users/:id", handler.Update)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/admin/users/91", bytes.NewBufferString(`{"password":"must-not-be-accepted"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	require.NotContains(t, w.Body.String(), "must-not-be-accepted")
}

func TestAdminUserResetPasswordReturnsCredentialOnce(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stub := &temporaryCredentialAdminServiceStub{}
	handler := NewUserHandler(stub, nil, nil, nil)
	router := gin.New()
	router.POST("/admin/users/:id/reset-password", handler.ResetPassword)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/users/92/reset-password", nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, int64(92), stub.resetUserID)
	require.Contains(t, w.Body.String(), "reset-one-time-value")
}
