package repository

import (
	"context"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func TestSettleVideoFinalizationBilling_ReviewRequiredKeepsHungWithoutCharge(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT id, user_id, api_key_id, reserved_amount_usd::text, status").
		WithArgs(int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "api_key_id", "reserved_amount_usd", "status"}).
			AddRow(int64(42), int64(7), nil, "3.0000000000", service.BillingReservationStatusReviewRequired))
	// Must not settle/release/charge when reservation is already review_required.
	mock.ExpectCommit()

	tx, err := db.Begin()
	require.NoError(t, err)
	reservationID := int64(42)
	task := videoFinalizationTaskRow{
		provider:      service.VideoProviderSeedance,
		userID:        7,
		reservationID: &reservationID,
	}
	input := service.VideoTaskFinalizationInput{
		TaskID:         1001,
		TerminalStatus: service.VideoStatusSucceeded,
		ActualCostUSD:  service.MustUSD("2.5"),
		CompletedAt:    time.Now().UTC(),
	}

	txID, overrun, chargeApplied, reviewRequired, err := settleVideoFinalizationBilling(context.Background(), tx, task, input)
	require.NoError(t, err, "finalize must not fail when reservation is already review_required")
	require.True(t, reviewRequired)
	require.False(t, chargeApplied)
	require.False(t, overrun)
	require.Equal(t, int64(0), txID)
	require.NoError(t, tx.Commit())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSettleVideoFinalizationBilling_NonActiveNonReviewStillFails(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT id, user_id, api_key_id, reserved_amount_usd::text, status").
		WithArgs(int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "api_key_id", "reserved_amount_usd", "status"}).
			AddRow(int64(42), int64(7), nil, "3.0000000000", service.BillingReservationStatusSettled))
	mock.ExpectRollback()

	tx, err := db.Begin()
	require.NoError(t, err)
	reservationID := int64(42)
	task := videoFinalizationTaskRow{
		provider:      service.VideoProviderSeedance,
		userID:        7,
		reservationID: &reservationID,
	}
	_, _, _, _, err = settleVideoFinalizationBilling(context.Background(), tx, task, service.VideoTaskFinalizationInput{
		TaskID:         1001,
		TerminalStatus: service.VideoStatusSucceeded,
		ActualCostUSD:  service.MustUSD("2.5"),
		CompletedAt:    time.Now().UTC(),
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "not active")
	_ = tx.Rollback()
	require.NoError(t, mock.ExpectationsWereMet())
}
