package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/shopspring/decimal"
)

const providerRealAccessPolicyColumns = `
	id, name, enabled, global_kill_switch, allow_member, allow_group,
	image_daily_cny::text, video_daily_cny::text, monthly_cny::text,
	enabled_at, disabled_at, audit_actor_id, audit_actor_email, created_at, updated_at`

type providerRealAccessPolicyRepository struct {
	db *sql.DB
}

func NewProviderRealAccessPolicyRepository(db *sql.DB) service.ProviderRealAccessPolicyRepository {
	return &providerRealAccessPolicyRepository{db: db}
}

func (r *providerRealAccessPolicyRepository) GetPolicy(ctx context.Context, name string) (*service.ProviderRealAccessPolicy, error) {
	name = normalizePolicyName(name)
	policy, err := scanProviderRealAccessPolicy(r.db.QueryRowContext(ctx, `
		SELECT `+providerRealAccessPolicyColumns+`
		FROM provider_real_access_policies
		WHERE name = $1
	`, name))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("policy not found")
	}
	return policy, err
}

func (r *providerRealAccessPolicyRepository) SavePolicy(ctx context.Context, policy *service.ProviderRealAccessPolicy) error {
	if policy == nil {
		return fmt.Errorf("policy required")
	}
	name := normalizePolicyName(policy.Name)
	now := time.Now().UTC()
	var enabledAt, disabledAt any
	if policy.Enabled {
		if policy.EnabledAt != nil {
			enabledAt = *policy.EnabledAt
		} else {
			enabledAt = now
		}
		disabledAt = nil
	} else {
		enabledAt = nil
		if policy.DisabledAt != nil {
			disabledAt = *policy.DisabledAt
		} else {
			disabledAt = now
		}
	}
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO provider_real_access_policies (
			name, enabled, global_kill_switch, allow_member, allow_group,
			image_daily_cny, video_daily_cny, monthly_cny,
			enabled_at, disabled_at, audit_actor_id, audit_actor_email, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		ON CONFLICT (name) DO UPDATE SET
			enabled = EXCLUDED.enabled,
			global_kill_switch = EXCLUDED.global_kill_switch,
			allow_member = EXCLUDED.allow_member,
			allow_group = EXCLUDED.allow_group,
			image_daily_cny = EXCLUDED.image_daily_cny,
			video_daily_cny = EXCLUDED.video_daily_cny,
			monthly_cny = EXCLUDED.monthly_cny,
			enabled_at = EXCLUDED.enabled_at,
			disabled_at = EXCLUDED.disabled_at,
			audit_actor_id = EXCLUDED.audit_actor_id,
			audit_actor_email = EXCLUDED.audit_actor_email,
			updated_at = EXCLUDED.updated_at
		RETURNING `+providerRealAccessPolicyColumns,
		name,
		policy.Enabled,
		policy.GlobalKillSwitch,
		policy.AllowMember,
		policy.AllowGroup,
		policy.ImageDailyCNY.String(),
		policy.VideoDailyCNY.String(),
		policy.MonthlyCNY.String(),
		enabledAt,
		disabledAt,
		policy.AuditActorID,
		strings.TrimSpace(policy.AuditActorEmail),
		now,
	)
	saved, err := scanProviderRealAccessPolicy(row)
	if err != nil {
		return err
	}
	*policy = *saved
	return nil
}

func (r *providerRealAccessPolicyRepository) ReserveInTx(ctx context.Context, reservation service.ProviderRealAccessReservation) error {
	opID := strings.TrimSpace(reservation.OperationID)
	if opID == "" {
		return fmt.Errorf("operation id required")
	}
	kind := strings.TrimSpace(reservation.Kind)
	if kind != "image" && kind != "video" {
		return fmt.Errorf("kind must be image or video")
	}
	if reservation.UserID <= 0 {
		return service.ErrInternalRealPolicyDenied
	}
	if !reservation.ReservedCNY.IsPositive() {
		return service.ErrInternalRealPolicyDenied
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	policy, err := scanProviderRealAccessPolicy(tx.QueryRowContext(ctx, `
		SELECT `+providerRealAccessPolicyColumns+`
		FROM provider_real_access_policies
		WHERE name = $1
		FOR UPDATE
	`, "default"))
	if errors.Is(err, sql.ErrNoRows) {
		return service.ErrInternalRealPolicyDenied
	}
	if err != nil {
		return err
	}
	if !policy.Enabled || policy.GlobalKillSwitch || !policy.AllowMember {
		return service.ErrInternalRealPolicyDenied
	}

	existing, err := getProviderRealAccessReservationByOp(ctx, tx, opID)
	if err == nil {
		if existing.ReservedCNY.Equal(reservation.ReservedCNY) && existing.Kind == kind && existing.UserID == reservation.UserID {
			return tx.Commit()
		}
		return fmt.Errorf("INTERNAL_REAL_IDEMPOTENCY_MISMATCH")
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	limit := policy.MonthlyCNY
	switch kind {
	case "image":
		if policy.ImageDailyCNY.IsPositive() && (limit.IsZero() || policy.ImageDailyCNY.LessThan(limit)) {
			limit = policy.ImageDailyCNY
		}
	case "video":
		if policy.VideoDailyCNY.IsPositive() && (limit.IsZero() || policy.VideoDailyCNY.LessThan(limit)) {
			limit = policy.VideoDailyCNY
		}
	}
	if !limit.IsPositive() {
		return service.ErrInternalRealBudgetExceeded
	}

	var usedRaw string
	if err := tx.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(reserved_cny), 0)::text
		FROM provider_real_access_reservations
		WHERE user_id = $1 AND status = 'reserved' AND kind = $2
	`, reservation.UserID, kind).Scan(&usedRaw); err != nil {
		return err
	}
	used, err := decimal.NewFromString(usedRaw)
	if err != nil {
		return err
	}
	if used.Add(reservation.ReservedCNY).GreaterThan(limit) {
		return service.ErrInternalRealBudgetExceeded
	}

	policyID := policy.ID
	row := tx.QueryRowContext(ctx, `
		INSERT INTO provider_real_access_reservations (
			operation_id, user_id, kind, reserved_cny, status, policy_id
		) VALUES ($1, $2, $3, $4, 'reserved', $5)
		ON CONFLICT (operation_id) DO NOTHING
		RETURNING operation_id, user_id, kind, reserved_cny::text, status, settled_cny::text, policy_id
	`, opID, reservation.UserID, kind, reservation.ReservedCNY.String(), policyID)
	inserted, err := scanProviderRealAccessReservation(row)
	if err == nil {
		_ = inserted
		return tx.Commit()
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	existing, err = getProviderRealAccessReservationByOp(ctx, tx, opID)
	if err != nil {
		return err
	}
	if existing.ReservedCNY.Equal(reservation.ReservedCNY) && existing.Kind == kind && existing.UserID == reservation.UserID {
		return tx.Commit()
	}
	return fmt.Errorf("INTERNAL_REAL_IDEMPOTENCY_MISMATCH")
}

func (r *providerRealAccessPolicyRepository) Settle(ctx context.Context, operationID string, settledCNY decimal.Decimal) error {
	operationID = strings.TrimSpace(operationID)
	if operationID == "" {
		return nil
	}
	_, err := r.db.ExecContext(ctx, `
		UPDATE provider_real_access_reservations
		SET status = 'settled',
		    settled_cny = $2,
		    updated_at = NOW()
		WHERE operation_id = $1 AND status = 'reserved'
	`, operationID, settledCNY.String())
	return err
}

func (r *providerRealAccessPolicyRepository) Release(ctx context.Context, operationID string) error {
	operationID = strings.TrimSpace(operationID)
	if operationID == "" {
		return nil
	}
	_, err := r.db.ExecContext(ctx, `
		UPDATE provider_real_access_reservations
		SET status = 'released',
		    updated_at = NOW()
		WHERE operation_id = $1 AND status = 'reserved'
	`, operationID)
	return err
}

func (r *providerRealAccessPolicyRepository) SumReservedCNY(ctx context.Context, userID int64, kind string, since time.Time) (decimal.Decimal, error) {
	query := `
		SELECT COALESCE(SUM(reserved_cny), 0)::text
		FROM provider_real_access_reservations
		WHERE user_id = $1 AND status = 'reserved'`
	args := []any{userID}
	argN := 2
	if strings.TrimSpace(kind) != "" {
		query += fmt.Sprintf(" AND kind = $%d", argN)
		args = append(args, strings.TrimSpace(kind))
		argN++
	}
	if !since.IsZero() {
		query += fmt.Sprintf(" AND created_at >= $%d", argN)
		args = append(args, since.UTC())
	}
	var raw string
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&raw); err != nil {
		return decimal.Zero, err
	}
	return decimal.NewFromString(raw)
}

func normalizePolicyName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "default"
	}
	return name
}

func getProviderRealAccessReservationByOp(ctx context.Context, queryer sqlQueryRower, operationID string) (*service.ProviderRealAccessReservation, error) {
	return scanProviderRealAccessReservation(queryer.QueryRowContext(ctx, `
		SELECT operation_id, user_id, kind, reserved_cny::text, status, settled_cny::text, policy_id
		FROM provider_real_access_reservations
		WHERE operation_id = $1
	`, operationID))
}

func scanProviderRealAccessPolicy(scanner sqlRowScanner) (*service.ProviderRealAccessPolicy, error) {
	var (
		item                                   service.ProviderRealAccessPolicy
		imageRaw, videoRaw, monthlyRaw         string
		enabledAt, disabledAt, createdAt, updatedAt sql.NullTime
		auditActorID                           sql.NullInt64
	)
	if err := scanner.Scan(
		&item.ID,
		&item.Name,
		&item.Enabled,
		&item.GlobalKillSwitch,
		&item.AllowMember,
		&item.AllowGroup,
		&imageRaw,
		&videoRaw,
		&monthlyRaw,
		&enabledAt,
		&disabledAt,
		&auditActorID,
		&item.AuditActorEmail,
		&createdAt,
		&updatedAt,
	); err != nil {
		return nil, err
	}
	var err error
	if item.ImageDailyCNY, err = decimal.NewFromString(imageRaw); err != nil {
		return nil, fmt.Errorf("decode image_daily_cny: %w", err)
	}
	if item.VideoDailyCNY, err = decimal.NewFromString(videoRaw); err != nil {
		return nil, fmt.Errorf("decode video_daily_cny: %w", err)
	}
	if item.MonthlyCNY, err = decimal.NewFromString(monthlyRaw); err != nil {
		return nil, fmt.Errorf("decode monthly_cny: %w", err)
	}
	item.EnabledAt = nullableTime(enabledAt)
	item.DisabledAt = nullableTime(disabledAt)
	item.AuditActorID = nullableInt64(auditActorID)
	if updatedAt.Valid {
		item.UpdatedAt = updatedAt.Time
	}
	return &item, nil
}

func scanProviderRealAccessReservation(scanner sqlRowScanner) (*service.ProviderRealAccessReservation, error) {
	var (
		item                      service.ProviderRealAccessReservation
		reservedRaw, settledRaw   sql.NullString
		policyID                  sql.NullInt64
	)
	if err := scanner.Scan(
		&item.OperationID,
		&item.UserID,
		&item.Kind,
		&reservedRaw,
		&item.Status,
		&settledRaw,
		&policyID,
	); err != nil {
		return nil, err
	}
	if reservedRaw.Valid {
		reserved, err := decimal.NewFromString(reservedRaw.String)
		if err != nil {
			return nil, fmt.Errorf("decode reserved_cny: %w", err)
		}
		item.ReservedCNY = reserved
	}
	if settledRaw.Valid && settledRaw.String != "" {
		settled, err := decimal.NewFromString(settledRaw.String)
		if err != nil {
			return nil, fmt.Errorf("decode settled_cny: %w", err)
		}
		item.SettledCNY = &settled
	}
	item.PolicyID = nullableInt64(policyID)
	return &item, nil
}
