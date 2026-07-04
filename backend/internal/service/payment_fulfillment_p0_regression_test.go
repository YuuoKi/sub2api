//go:build unit

package service

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/ent/paymentauditlog"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/stretchr/testify/require"
)

func TestConfirmPayment_MissingOrder_ReturnsError(t *testing.T) {
	ctx := context.Background()
	client := newOrderNotFoundTestClient(t)

	svc := &PaymentService{entClient: client, providersLoaded: true}

	err := svc.confirmPayment(ctx, 99999, "trade-1", 10, payment.TypeAlipay, nil, "")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrOrderNotFound)
}

func TestHandlePaymentNotification_LegacyFallback_RejectsOutTradeNoMismatch(t *testing.T) {
	ctx := context.Background()
	client := newOrderNotFoundTestClient(t)

	user, err := client.User.Create().
		SetEmail("legacy-mismatch@example.com").
		SetPasswordHash("hash").
		SetUsername("legacy-mismatch").
		Save(ctx)
	require.NoError(t, err)

	order, err := client.PaymentOrder.Create().
		SetUserID(user.ID).
		SetUserEmail(user.Email).
		SetUserName(user.Username).
		SetAmount(10).
		SetPayAmount(10).
		SetFeeRate(0).
		SetRechargeCode("LEGACY-MISMATCH").
		SetOutTradeNo("sub2_20260704abc").
		SetPaymentType(payment.TypeAlipay).
		SetPaymentTradeNo("").
		SetOrderType(payment.OrderTypeBalance).
		SetStatus(OrderStatusPending).
		SetExpiresAt(time.Now().Add(time.Hour)).
		SetClientIP("127.0.0.1").
		SetSrcHost("api.example.com").
		Save(ctx)
	require.NoError(t, err)

	svc := &PaymentService{entClient: client, providersLoaded: true}

	// Webhook uses legacy sub2_{pk} form that parses to this order's PK but does
	// not match the order's real out_trade_no — must not credit.
	notification := &payment.PaymentNotification{
		OrderID: "sub2_" + strconv.FormatInt(order.ID, 10),
		TradeNo: "upstream-trade",
		Status:  payment.NotificationStatusSuccess,
		Amount:  10,
	}

	err = svc.HandlePaymentNotification(ctx, notification, payment.TypeAlipay)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrOrderNotFound)
}

func TestMarkCompleted_RejectsWhenNotRecharging(t *testing.T) {
	ctx := context.Background()
	client := newOrderNotFoundTestClient(t)

	user, err := client.User.Create().
		SetEmail("markcompleted@example.com").
		SetPasswordHash("hash").
		SetUsername("markcompleted-user").
		Save(ctx)
	require.NoError(t, err)

	order, err := client.PaymentOrder.Create().
		SetUserID(user.ID).
		SetUserEmail(user.Email).
		SetUserName(user.Username).
		SetAmount(10).
		SetPayAmount(10).
		SetFeeRate(0).
		SetRechargeCode("MARK-COMPLETE").
		SetOutTradeNo("sub2_mark_complete_test").
		SetPaymentType(payment.TypeAlipay).
		SetPaymentTradeNo("").
		SetOrderType(payment.OrderTypeBalance).
		SetStatus(OrderStatusPaid).
		SetExpiresAt(time.Now().Add(time.Hour)).
		SetClientIP("127.0.0.1").
		SetSrcHost("api.example.com").
		Save(ctx)
	require.NoError(t, err)

	svc := &PaymentService{entClient: client, providersLoaded: true}
	err = svc.markCompleted(ctx, order, "RECHARGE_SUCCESS")
	require.Error(t, err)
	require.Contains(t, err.Error(), "not in RECHARGING status")

	count, err := client.PaymentAuditLog.Query().
		Where(
			paymentauditlog.OrderIDEQ(strconv.FormatInt(order.ID, 10)),
			paymentauditlog.ActionEQ("RECHARGE_SUCCESS"),
		).
		Count(ctx)
	require.NoError(t, err)
	require.Zero(t, count)
}

func TestAlreadyProcessed_ExpiredBeyondGrace_ReturnsSentinel(t *testing.T) {
	ctx := context.Background()
	client := newOrderNotFoundTestClient(t)

	user, err := client.User.Create().
		SetEmail("expired@example.com").
		SetPasswordHash("hash").
		SetUsername("expired-user").
		Save(ctx)
	require.NoError(t, err)

	expiredAt := time.Now().Add(-2 * time.Hour)
	order, err := client.PaymentOrder.Create().
		SetUserID(user.ID).
		SetUserEmail(user.Email).
		SetUserName(user.Username).
		SetAmount(10).
		SetPayAmount(10).
		SetFeeRate(0).
		SetRechargeCode("EXPIRED-ORDER").
		SetOutTradeNo("sub2_expired_beyond_grace").
		SetPaymentType(payment.TypeAlipay).
		SetPaymentTradeNo("").
		SetOrderType(payment.OrderTypeBalance).
		SetStatus(OrderStatusExpired).
		SetExpiresAt(expiredAt).
		SetClientIP("127.0.0.1").
		SetSrcHost("api.example.com").
		Save(ctx)
	require.NoError(t, err)

	_, err = client.PaymentOrder.UpdateOneID(order.ID).
		SetUpdatedAt(expiredAt).
		Save(ctx)
	require.NoError(t, err)

	reloaded, err := client.PaymentOrder.Get(ctx, order.ID)
	require.NoError(t, err)

	svc := &PaymentService{entClient: client, providersLoaded: true}
	err = svc.alreadyProcessed(ctx, reloaded)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrPaymentAfterExpiry)

	auditCount, err := client.PaymentAuditLog.Query().
		Where(
			paymentauditlog.OrderIDEQ(strconv.FormatInt(order.ID, 10)),
			paymentauditlog.ActionEQ("PAYMENT_AFTER_EXPIRY"),
		).
		Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, auditCount)
}

func TestAlreadyProcessed_CancelledOrder_ReturnsRejected(t *testing.T) {
	ctx := context.Background()
	client := newOrderNotFoundTestClient(t)

	user, err := client.User.Create().
		SetEmail("cancelled@example.com").
		SetPasswordHash("hash").
		SetUsername("cancelled-user").
		Save(ctx)
	require.NoError(t, err)

	order, err := client.PaymentOrder.Create().
		SetUserID(user.ID).
		SetUserEmail(user.Email).
		SetUserName(user.Username).
		SetAmount(10).
		SetPayAmount(10).
		SetFeeRate(0).
		SetRechargeCode("CANCELLED-ORDER").
		SetOutTradeNo("sub2_cancelled_order").
		SetPaymentType(payment.TypeAlipay).
		SetPaymentTradeNo("").
		SetOrderType(payment.OrderTypeBalance).
		SetStatus(OrderStatusCancelled).
		SetExpiresAt(time.Now().Add(time.Hour)).
		SetClientIP("127.0.0.1").
		SetSrcHost("api.example.com").
		Save(ctx)
	require.NoError(t, err)

	svc := &PaymentService{entClient: client, providersLoaded: true}
	err = svc.alreadyProcessed(ctx, order)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrPaymentRejected)

	auditCount, err := client.PaymentAuditLog.Query().
		Where(
			paymentauditlog.OrderIDEQ(strconv.FormatInt(order.ID, 10)),
			paymentauditlog.ActionEQ("PAYMENT_ON_CANCELLED_ORDER"),
		).
		Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, auditCount)
}

func TestAlreadyProcessed_RecentRechargingOrderStillAcksDuplicate(t *testing.T) {
	ctx := context.Background()
	client := newOrderNotFoundTestClient(t)

	user, err := client.User.Create().
		SetEmail("recent-recharging@example.com").
		SetPasswordHash("hash").
		SetUsername("recent-recharging-user").
		Save(ctx)
	require.NoError(t, err)

	order, err := client.PaymentOrder.Create().
		SetUserID(user.ID).
		SetUserEmail(user.Email).
		SetUserName(user.Username).
		SetAmount(10).
		SetPayAmount(10).
		SetFeeRate(0).
		SetRechargeCode("RECENT-RECHARGING").
		SetOutTradeNo("sub2_recent_recharging").
		SetPaymentType(payment.TypeAlipay).
		SetPaymentTradeNo("trade-recent").
		SetOrderType(payment.OrderTypeBalance).
		SetStatus(OrderStatusRecharging).
		SetExpiresAt(time.Now().Add(time.Hour)).
		SetClientIP("127.0.0.1").
		SetSrcHost("api.example.com").
		Save(ctx)
	require.NoError(t, err)

	svc := &PaymentService{entClient: client, providersLoaded: true}
	require.NoError(t, svc.alreadyProcessed(ctx, order))
}

func TestAlreadyProcessed_StaleRechargingOrderReturnsRetryableError(t *testing.T) {
	ctx := context.Background()
	client := newOrderNotFoundTestClient(t)

	user, err := client.User.Create().
		SetEmail("stale-recharging@example.com").
		SetPasswordHash("hash").
		SetUsername("stale-recharging-user").
		Save(ctx)
	require.NoError(t, err)

	order, err := client.PaymentOrder.Create().
		SetUserID(user.ID).
		SetUserEmail(user.Email).
		SetUserName(user.Username).
		SetAmount(10).
		SetPayAmount(10).
		SetFeeRate(0).
		SetRechargeCode("STALE-RECHARGING").
		SetOutTradeNo("sub2_stale_recharging").
		SetPaymentType(payment.TypeAlipay).
		SetPaymentTradeNo("trade-stale").
		SetOrderType(payment.OrderTypeBalance).
		SetStatus(OrderStatusRecharging).
		SetExpiresAt(time.Now().Add(time.Hour)).
		SetClientIP("127.0.0.1").
		SetSrcHost("api.example.com").
		Save(ctx)
	require.NoError(t, err)

	staleUpdatedAt := time.Now().Add(-paymentInProgressRecoveryAfter - time.Minute)
	_, err = client.PaymentOrder.UpdateOneID(order.ID).SetUpdatedAt(staleUpdatedAt).Save(ctx)
	require.NoError(t, err)
	reloaded, err := client.PaymentOrder.Get(ctx, order.ID)
	require.NoError(t, err)

	svc := &PaymentService{entClient: client, providersLoaded: true}
	err = svc.alreadyProcessed(ctx, reloaded)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrPaymentFulfillmentStale)

	auditCount, err := client.PaymentAuditLog.Query().
		Where(
			paymentauditlog.OrderIDEQ(strconv.FormatInt(order.ID, 10)),
			paymentauditlog.ActionEQ("PAYMENT_FULFILLMENT_STALE"),
		).
		Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, auditCount)
}

func TestToPaid_DoesNotAcceptCancelledOrders(t *testing.T) {
	ctx := context.Background()
	client := newOrderNotFoundTestClient(t)

	user, err := client.User.Create().
		SetEmail("topaid-cancelled@example.com").
		SetPasswordHash("hash").
		SetUsername("topaid-cancelled-user").
		Save(ctx)
	require.NoError(t, err)

	order, err := client.PaymentOrder.Create().
		SetUserID(user.ID).
		SetUserEmail(user.Email).
		SetUserName(user.Username).
		SetAmount(10).
		SetPayAmount(10).
		SetFeeRate(0).
		SetRechargeCode("TOPAID-CANCELLED").
		SetOutTradeNo("sub2_topaid_cancelled").
		SetPaymentType(payment.TypeAlipay).
		SetPaymentTradeNo("").
		SetOrderType(payment.OrderTypeBalance).
		SetStatus(OrderStatusCancelled).
		SetExpiresAt(time.Now().Add(time.Hour)).
		SetClientIP("127.0.0.1").
		SetSrcHost("api.example.com").
		Save(ctx)
	require.NoError(t, err)

	svc := &PaymentService{entClient: client, providersLoaded: true}
	err = svc.toPaid(ctx, order, "trade-cancelled", 10, payment.TypeAlipay)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrPaymentRejected)

	reloaded, err := client.PaymentOrder.Get(ctx, order.ID)
	require.NoError(t, err)
	require.Equal(t, OrderStatusCancelled, reloaded.Status)
}
