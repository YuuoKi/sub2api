//go:build unit

package handler

import (
	"encoding/json"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestRefreshTokenResponseExplicitlyCarriesTemporaryCredentialRestriction(t *testing.T) {
	result := &service.TokenPairWithUser{
		TokenPair: service.TokenPair{
			AccessToken:  "new-access-token",
			RefreshToken: "new-refresh-token",
			ExpiresIn:    3600,
		},
		UserRole:           service.RoleUser,
		MustChangePassword: true,
	}

	payload, err := json.Marshal(newRefreshTokenResponse(result))
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(payload, &decoded))
	require.Equal(t, true, decoded["mustChangePassword"])
	require.Equal(t, "new-access-token", decoded["access_token"])
	require.Equal(t, "new-refresh-token", decoded["refresh_token"])
}
