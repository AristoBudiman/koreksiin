package handlers

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// dbQueryExec is satisfied by both *pgxpool.Pool and pgx.Tx.
type dbQueryExec interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

type SubscriptionState struct {
	PlanID       int64
	PlanCode     string
	PlanName     string
	QuotaTotal   int
	QuotaUsed    int
	QuotaResetAt *time.Time
	ExpiresAt    *time.Time
}

type PlanSpec struct {
	ID           int64
	Code         string
	Name         string
	QuotaEssays  int
	DurationDays int
	IsAddon      bool
	PriceIDR     int
}

type PurchaseOutcome struct {
	NewPlanID     int64
	NewPlanCode   string
	NewPlanName   string
	NewQuotaTotal int
	NewQuotaUsed  int
	NewExpiresAt  *time.Time
}

// NormalizeSubscriptionState reverts an expired paid plan to the free plan
// and resets monthly free quota when quota_reset_at has passed.
func NormalizeSubscriptionState(cur SubscriptionState, freePlan PlanSpec, now time.Time) (SubscriptionState, bool) {
	if cur.PlanCode != "" && cur.PlanCode != "free" && cur.ExpiresAt != nil && now.After(*cur.ExpiresAt) {
		nextReset := now.AddDate(0, 1, 0)
		return SubscriptionState{
			PlanID:       freePlan.ID,
			PlanCode:     "free",
			PlanName:     freePlan.Name,
			QuotaTotal:   freePlan.QuotaEssays,
			QuotaUsed:    0,
			QuotaResetAt: &nextReset,
			ExpiresAt:    nil,
		}, true
	}

	if cur.PlanCode == "free" && cur.QuotaResetAt != nil && now.After(*cur.QuotaResetAt) {
		nextReset := now.AddDate(0, 1, 0)
		cur.QuotaUsed = 0
		cur.QuotaTotal = freePlan.QuotaEssays
		cur.QuotaResetAt = &nextReset
		return cur, true
	}

	return cur, false
}

// EvaluateSubscriptionPurchase validates all subscription rules and computes
// the resulting subscription state for a purchase.
func EvaluateSubscriptionPurchase(
	cur SubscriptionState,
	target PlanSpec,
	freePlan PlanSpec,
	hasPendingTx bool,
	now time.Time,
) (PurchaseOutcome, error) {
	if hasPendingTx {
		return PurchaseOutcome{}, errors.New("Anda masih memiliki transaksi yang menunggu pembayaran. Selesaikan atau batalkan transaksi tersebut terlebih dahulu.")
	}
	if target.PriceIDR <= 0 {
		return PurchaseOutcome{}, errors.New("Paket Gratis tidak memerlukan transaksi pembayaran.")
	}

	normalized, _ := NormalizeSubscriptionState(cur, freePlan, now)
	remaining := normalized.QuotaTotal - normalized.QuotaUsed
	if remaining < 0 {
		remaining = 0
	}

	isPaidActive := normalized.PlanCode != "" &&
		normalized.PlanCode != "free" &&
		(normalized.ExpiresAt == nil || !now.After(*normalized.ExpiresAt))

	if target.IsAddon {
		if !isPaidActive {
			return PurchaseOutcome{}, errors.New("Add-on Kuota hanya dapat dibeli saat paket berbayar (Bulanan atau Tahunan) sedang aktif.")
		}
		return PurchaseOutcome{
			NewPlanID:     normalized.PlanID,
			NewPlanCode:   normalized.PlanCode,
			NewPlanName:   normalized.PlanName,
			NewQuotaTotal: normalized.QuotaTotal + target.QuotaEssays,
			NewQuotaUsed:  normalized.QuotaUsed,
			NewExpiresAt:  normalized.ExpiresAt,
		}, nil
	}

	if isPaidActive && normalized.PlanCode == "yearly" && target.Code == "monthly" {
		return PurchaseOutcome{}, errors.New("Paket Sekolah Tahunan Anda masih aktif. Jika kuota esai kurang, silakan beli Add-on Kuota.")
	}

	baseTime := now
	if isPaidActive && normalized.ExpiresAt != nil && normalized.ExpiresAt.After(now) {
		baseTime = *normalized.ExpiresAt
	}
	exp := baseTime.AddDate(0, 0, target.DurationDays)

	newQuotaTotal := target.QuotaEssays
	if isPaidActive && remaining > 0 {
		newQuotaTotal = remaining + target.QuotaEssays
	}

	return PurchaseOutcome{
		NewPlanID:     target.ID,
		NewPlanCode:   target.Code,
		NewPlanName:   target.Name,
		NewQuotaTotal: newQuotaTotal,
		NewQuotaUsed:  0,
		NewExpiresAt:  &exp,
	}, nil
}

func loadFreePlan(ctx context.Context, q dbQueryExec) (PlanSpec, error) {
	var freePlan PlanSpec
	err := q.QueryRow(ctx, `
		SELECT id, code, name, quota_essays, duration_days, is_addon, price_idr
		FROM plans
		WHERE code = 'free'
		LIMIT 1
	`).Scan(
		&freePlan.ID, &freePlan.Code, &freePlan.Name,
		&freePlan.QuotaEssays, &freePlan.DurationDays, &freePlan.IsAddon, &freePlan.PriceIDR,
	)
	return freePlan, err
}

func refreshOrExpireSubscription(ctx context.Context, q dbQueryExec, userID int64, now time.Time) (SubscriptionState, PlanSpec, error) {
	freePlan, err := loadFreePlan(ctx, q)
	if err != nil {
		return SubscriptionState{}, PlanSpec{}, err
	}

	var cur SubscriptionState
	err = q.QueryRow(ctx, `
		SELECT s.plan_id, p.code, p.name, s.quota_total, s.quota_used, s.quota_reset_at, s.expires_at
		FROM subscriptions s
		JOIN plans p ON p.id = s.plan_id
		WHERE s.user_id = $1
	`, userID).Scan(
		&cur.PlanID, &cur.PlanCode, &cur.PlanName,
		&cur.QuotaTotal, &cur.QuotaUsed, &cur.QuotaResetAt, &cur.ExpiresAt,
	)
	if err != nil {
		return SubscriptionState{}, freePlan, err
	}

	normalized, changed := NormalizeSubscriptionState(cur, freePlan, now)
	if changed {
		_, err = q.Exec(ctx, `
			UPDATE subscriptions
			SET plan_id = $1,
			    quota_total = $2,
			    quota_used = $3,
			    quota_reset_at = $4,
			    expires_at = $5,
			    updated_at = NOW()
			WHERE user_id = $6
		`, normalized.PlanID, normalized.QuotaTotal, normalized.QuotaUsed, normalized.QuotaResetAt, normalized.ExpiresAt, userID)
		if err != nil {
			return normalized, freePlan, err
		}
	}

	return normalized, freePlan, nil
}

