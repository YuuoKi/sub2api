package repository

import (
	"context"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestPgAdvisoryLock_UnlockOnSameConnChecksResult(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectQuery("SELECT pg_try_advisory_lock\\(\\$1\\)").
		WithArgs(migrationsAdvisoryLockID).
		WillReturnRows(sqlmock.NewRows([]string{"pg_try_advisory_lock"}).AddRow(true))
	// Simulate unlock on a connection that does not hold the lock (pool mismatch).
	mock.ExpectQuery("SELECT pg_advisory_unlock\\(\\$1\\)").
		WithArgs(migrationsAdvisoryLockID).
		WillReturnRows(sqlmock.NewRows([]string{"pg_advisory_unlock"}).AddRow(false))

	conn, err := pgAdvisoryLock(context.Background(), db)
	require.NoError(t, err)
	require.NotNil(t, conn)

	err = pgAdvisoryUnlock(conn)
	require.Error(t, err)
	require.Contains(t, err.Error(), "release migrations lock")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgAdvisoryLock_UnlockSuccessClosesConn(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectQuery("SELECT pg_try_advisory_lock\\(\\$1\\)").
		WithArgs(migrationsAdvisoryLockID).
		WillReturnRows(sqlmock.NewRows([]string{"pg_try_advisory_lock"}).AddRow(true))
	mock.ExpectQuery("SELECT pg_advisory_unlock\\(\\$1\\)").
		WithArgs(migrationsAdvisoryLockID).
		WillReturnRows(sqlmock.NewRows([]string{"pg_advisory_unlock"}).AddRow(true))

	conn, err := pgAdvisoryLock(context.Background(), db)
	require.NoError(t, err)
	require.NoError(t, pgAdvisoryUnlock(conn))
	require.NoError(t, mock.ExpectationsWereMet())
}
