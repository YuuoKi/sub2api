//go:build unit

package admin

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type memberTypeAdminServiceStub struct {
	service.AdminService
	created *service.CreateUserInput
	updated *service.UpdateUserInput
}

func (s *memberTypeAdminServiceStub) CreateUser(_ context.Context, input *service.CreateUserInput) (*service.User, error) {
	s.created = input
	return &service.User{ID: 42, Email: input.Email, Notes: input.Notes, Role: service.RoleUser, Status: service.StatusActive}, nil
}

func (s *memberTypeAdminServiceStub) UpdateUser(_ context.Context, id int64, input *service.UpdateUserInput) (*service.User, error) {
	s.updated = input
	return &service.User{ID: id, Email: "member@example.com", Role: service.RoleUser, Status: service.StatusActive}, nil
}

func TestAdminUserCreateValidatesAndForwardsMemberType(t *testing.T) {
	for _, tt := range []struct {
		name       string
		body       string
		wantStatus int
		wantType   string
		wantCall   bool
	}{
		{name: "tool", body: `{"email":"tool@example.com","member_type":"tool"}`, wantStatus: http.StatusOK, wantType: service.UserMemberTypeTool, wantCall: true},
		{name: "omitted", body: `{"email":"human@example.com"}`, wantStatus: http.StatusOK, wantCall: true},
		{name: "invalid", body: `{"email":"robot@example.com","member_type":"robot"}`, wantStatus: http.StatusBadRequest},
	} {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			stub := &memberTypeAdminServiceStub{}
			handler := NewUserHandler(stub, nil, nil, nil)
			router := gin.New()
			router.POST("/admin/users", handler.Create)

			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/admin/users", bytes.NewBufferString(tt.body))
			request.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(response, request)

			require.Equal(t, tt.wantStatus, response.Code)
			if tt.wantCall {
				require.NotNil(t, stub.created)
				require.Equal(t, tt.wantType, stub.created.MemberType)
			} else {
				require.Nil(t, stub.created)
			}
		})
	}
}

func TestAdminUserUpdateValidatesAndForwardsMemberType(t *testing.T) {
	for _, tt := range []struct {
		name       string
		body       string
		wantStatus int
		wantType   *string
		wantCall   bool
	}{
		{name: "human", body: `{"member_type":"human"}`, wantStatus: http.StatusOK, wantType: memberTypeTestStringPointer(service.UserMemberTypeHuman), wantCall: true},
		{name: "omitted", body: `{"notes":"renamed"}`, wantStatus: http.StatusOK, wantCall: true},
		{name: "invalid", body: `{"member_type":"robot"}`, wantStatus: http.StatusBadRequest},
	} {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			stub := &memberTypeAdminServiceStub{}
			handler := NewUserHandler(stub, nil, nil, nil)
			router := gin.New()
			router.PUT("/admin/users/:id", handler.Update)

			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPut, "/admin/users/42", bytes.NewBufferString(tt.body))
			request.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(response, request)

			require.Equal(t, tt.wantStatus, response.Code)
			if tt.wantCall {
				require.NotNil(t, stub.updated)
				require.Equal(t, tt.wantType, stub.updated.MemberType)
			} else {
				require.Nil(t, stub.updated)
			}
		})
	}
}

func memberTypeTestStringPointer(value string) *string { return &value }
