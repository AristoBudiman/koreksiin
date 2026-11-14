package handlers

import (
	"testing"
	"time"
)

func testPlans() (free, monthly, yearly, addon PlanSpec) {
	free = PlanSpec{ID: 1, Code: "free", Name: "Paket Gratis", QuotaEssays: 35, DurationDays: 30, IsAddon: false, PriceIDR: 0}
	monthly = PlanSpec{ID: 2, Code: "monthly", Name: "Paket Guru Bulanan", QuotaEssays: 300, DurationDays: 30, IsAddon: false, PriceIDR: 49000}
	yearly = PlanSpec{ID: 3, Code: "yearly", Name: "Paket Sekolah Tahunan", QuotaEssays: 4000, DurationDays: 365, IsAddon: false, PriceIDR: 450000}
	addon = PlanSpec{ID: 4, Code: "addon", Name: "Add-on Kuota +150 Esai", QuotaEssays: 150, DurationDays: 0, IsAddon: true, PriceIDR: 20000}
	return
}

func TestRule1_BuySamePlanWhileActive(t *testing.T) {
	free, monthly, _, _ := testPlans()
	now := time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)
	curExp := now.AddDate(0, 0, 10)
	cur := SubscriptionState{
		PlanID: monthly.ID, PlanCode: "monthly", PlanName: monthly.Name,
		QuotaTotal: 300, QuotaUsed: 120, ExpiresAt: &curExp,
	}

	out, err := EvaluateSubscriptionPurchase(cur, monthly, free, false, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Remaining was 180, + 300 = 480
	if out.NewQuotaTotal != 480 || out.NewQuotaUsed != 0 {
		t.Fatalf("expected quota 480/0, got %d/%d", out.NewQuotaTotal, out.NewQuotaUsed)
	}
	expectedExp := curExp.AddDate(0, 0, 30)
	if out.NewExpiresAt == nil || !out.NewExpiresAt.Equal(expectedExp) {
		t.Fatalf("expected expires_at %v, got %v", expectedExp, out.NewExpiresAt)
	}
}

func TestRule2_UpgradeMonthlyToYearly(t *testing.T) {
	free, monthly, yearly, _ := testPlans()
	now := time.Date(2026, 11, 14, 14, 8, 0, 0, time.UTC)
	curExp := now.AddDate(0, 0, 30)
	cur := SubscriptionState{
		PlanID: monthly.ID, PlanCode: "monthly", PlanName: monthly.Name,
		QuotaTotal: 300, QuotaUsed: 50, ExpiresAt: &curExp,
	}

	out, err := EvaluateSubscriptionPurchase(cur, yearly, free, false, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.NewPlanCode != "yearly" {
		t.Fatalf("expected yearly, got %s", out.NewPlanCode)
	}
	// Remaining 250 + 4000 = 4250
	if out.NewQuotaTotal != 4250 || out.NewQuotaUsed != 0 {
		t.Fatalf("expected quota 4250/0, got %d/%d", out.NewQuotaTotal, out.NewQuotaUsed)
	}
	expectedExp := curExp.AddDate(0, 0, 365)
	if out.NewExpiresAt == nil || !out.NewExpiresAt.Equal(expectedExp) {
		t.Fatalf("expected expires_at %v, got %v", expectedExp, out.NewExpiresAt)
	}
}

func TestRule3_DowngradeYearlyToMonthlyBlocked(t *testing.T) {
	free, monthly, yearly, _ := testPlans()
	now := time.Date(2026, 11, 14, 14, 0, 0, 0, time.UTC)
	curExp := now.AddDate(0, 0, 100)
	cur := SubscriptionState{
		PlanID: yearly.ID, PlanCode: "yearly", PlanName: yearly.Name,
		QuotaTotal: 4000, QuotaUsed: 10, ExpiresAt: &curExp,
	}

	_, err := EvaluateSubscriptionPurchase(cur, monthly, free, false, now)
	if err == nil {
		t.Fatalf("expected error blocking downgrade from yearly to monthly, got nil")
	}
}

func TestRule4_AddonRequiresActivePaidPlan(t *testing.T) {
	free, monthly, _, addon := testPlans()
	now := time.Date(2026, 11, 14, 14, 0, 0, 0, time.UTC)
	resetAt := now.AddDate(0, 1, 0)

	// Free plan -> blocked
	curFree := SubscriptionState{
		PlanID: free.ID, PlanCode: "free", PlanName: free.Name,
		QuotaTotal: 35, QuotaUsed: 5, QuotaResetAt: &resetAt,
	}
	if _, err := EvaluateSubscriptionPurchase(curFree, addon, free, false, now); err == nil {
		t.Fatalf("expected error buying add-on on free plan")
	}

	// Expired paid plan -> normalized to free -> blocked
	pastExp := now.Add(-1 * time.Hour)
	curExpired := SubscriptionState{
		PlanID: monthly.ID, PlanCode: "monthly", PlanName: monthly.Name,
		QuotaTotal: 300, QuotaUsed: 100, ExpiresAt: &pastExp,
	}
	if _, err := EvaluateSubscriptionPurchase(curExpired, addon, free, false, now); err == nil {
		t.Fatalf("expected error buying add-on on expired paid plan")
	}

	// Active paid plan -> allowed, keeps expires_at and adds quota
	futureExp := now.AddDate(0, 0, 20)
	curActive := SubscriptionState{
		PlanID: monthly.ID, PlanCode: "monthly", PlanName: monthly.Name,
		QuotaTotal: 300, QuotaUsed: 100, ExpiresAt: &futureExp,
	}
	out, err := EvaluateSubscriptionPurchase(curActive, addon, free, false, now)
	if err != nil {
		t.Fatalf("unexpected error buying add-on on active paid plan: %v", err)
	}
	if out.NewPlanCode != "monthly" || out.NewQuotaTotal != 450 || out.NewQuotaUsed != 100 {
		t.Fatalf("unexpected add-on outcome: %+v", out)
	}
	if out.NewExpiresAt == nil || !out.NewExpiresAt.Equal(futureExp) {
		t.Fatalf("expected expires_at unchanged (%v), got %v", futureExp, out.NewExpiresAt)
	}
}

func TestRule5_ExpiredPaidPlanRevertsToFreeAndForfeitsQuota(t *testing.T) {
	free, monthly, _, _ := testPlans()
	now := time.Date(2026, 11, 14, 14, 0, 0, 0, time.UTC)
	pastExp := now.Add(-24 * time.Hour)

	cur := SubscriptionState{
		PlanID: monthly.ID, PlanCode: "monthly", PlanName: monthly.Name,
		QuotaTotal: 450, QuotaUsed: 100, ExpiresAt: &pastExp,
	}
	norm, changed := NormalizeSubscriptionState(cur, free, now)
	if !changed {
		t.Fatalf("expected expired paid plan to normalize")
	}
	if norm.PlanCode != "free" || norm.QuotaTotal != 35 || norm.QuotaUsed != 0 || norm.ExpiresAt != nil || norm.QuotaResetAt == nil {
		t.Fatalf("unexpected normalized state: %+v", norm)
	}

	// Buying monthly after expiry does NOT carry over forfeited 350 quota
	out, err := EvaluateSubscriptionPurchase(cur, monthly, free, false, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.NewQuotaTotal != 300 {
		t.Fatalf("expected fresh 300 quota after expiry, got %d", out.NewQuotaTotal)
	}
}

func TestRule6_PendingTransactionBlocksNewCheckout(t *testing.T) {
	free, monthly, _, _ := testPlans()
	now := time.Date(2026, 11, 14, 14, 0, 0, 0, time.UTC)
	resetAt := now.AddDate(0, 1, 0)
	curFree := SubscriptionState{
		PlanID: free.ID, PlanCode: "free", PlanName: free.Name,
		QuotaTotal: 35, QuotaUsed: 0, QuotaResetAt: &resetAt,
	}

	if _, err := EvaluateSubscriptionPurchase(curFree, monthly, free, true, now); err == nil {
		t.Fatalf("expected error when pending transaction exists")
	}
}

