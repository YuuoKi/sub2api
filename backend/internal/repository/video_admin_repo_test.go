package repository

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestCreateVideoProviderDetectsConflictOnCustomModel(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	repo := &videoAdminRepository{db: db}
	mock.ExpectQuery(`EXISTS\(SELECT 1 FROM groups[\s\S]+provider=\$2 AND default_model=\$3 AND id<>\$4`).
		WithArgs(int64(9), "seedance", "relay-model", int64(0)).
		WillReturnRows(sqlmock.NewRows([]string{"valid", "conflict"}).AddRow(true, true))
	_, err = repo.CreateVideoProvider(context.Background(), service.VideoProviderAccount{
		GroupID: 9, Provider: "seedance", DisplayName: "dup", BaseURL: "https://relay.test/v1", DefaultModel: "relay-model",
	})
	require.ErrorIs(t, err, service.ErrVideoAdminConflict)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAuthorizeTinyRealRequiresCanonicalActiveStandardProvider(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	repo := &videoAdminRepository{db: db}
	mock.ExpectQuery(`UPDATE video_provider_accounts[\s\S]+FROM groups[\s\S]+subscription_type='standard'[\s\S]+default_model=\$3[\s\S]+base_url=\$4`).
		WithArgs(int64(7), int64(3), service.SeedanceModel, service.SeedanceBaseURL).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM video_provider_accounts WHERE id=\$1\)`).WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	_, err = repo.AuthorizeTinyReal(context.Background(), 7, 3)
	require.ErrorIs(t, err, service.ErrVideoAdminAuthorizationConflict)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAuthorizeTinyRealReturnsNotFoundForUnknownProvider(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	repo := &videoAdminRepository{db: db}
	mock.ExpectQuery("WITH updated AS").WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery("SELECT EXISTS").WithArgs(int64(99)).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	_, err = repo.AuthorizeTinyReal(context.Background(), 99, 3)
	require.True(t, errors.Is(err, service.ErrVideoProviderNotFound))
	require.NoError(t, mock.ExpectationsWereMet())
}
