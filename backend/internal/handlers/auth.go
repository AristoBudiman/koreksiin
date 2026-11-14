package handlers

import (
	"context"
	"net/http"
	"strings"
	"time"

	"koreksiin-backend/internal/config"
	"koreksiin-backend/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	pool *pgxpool.Pool
	cfg  *config.Config
}

func NewAuthHandler(pool *pgxpool.Pool, cfg *config.Config) *AuthHandler {
	return &AuthHandler{pool: pool, cfg: cfg}
}

type registerRequest struct {
	Name        string `json:"name" binding:"required"`
	Email       string `json:"email" binding:"required,email"`
	Institution string `json:"institution" binding:"required"`
	Password    string `json:"password" binding:"required,min=6"`
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type SubscriptionSummary struct {
	PlanCode       string     `json:"plan_code"`
	PlanName       string     `json:"plan_name"`
	QuotaTotal     int        `json:"quota_total"`
	QuotaUsed      int        `json:"quota_used"`
	QuotaRemaining int        `json:"quota_remaining"`
	QuotaResetAt   *time.Time `json:"quota_reset_at"`
	ExpiresAt      *time.Time `json:"expires_at"`
}

type UserResponse struct {
	ID           int64                `json:"id"`
	Name         string               `json:"name"`
	Email        string               `json:"email"`
	Institution  string               `json:"institution"`
	Role         string               `json:"role"`
	IsActive     bool                 `json:"is_active"`
	CreatedAt    time.Time            `json:"created_at"`
	Subscription *SubscriptionSummary `json:"subscription,omitempty"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Pastikan nama lengkap, email valid, sekolah/instansi, dan kata sandi (minimal 6 karakter) telah diisi.",
		})
		return
	}

	name := strings.TrimSpace(req.Name)
	email := strings.ToLower(strings.TrimSpace(req.Email))
	institution := strings.TrimSpace(req.Institution)

	if name == "" || institution == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nama lengkap dan sekolah/instansi tidak boleh kosong."})
		return
	}

	ctx := c.Request.Context()
	var exists bool
	if err := h.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`, email).Scan(&exists); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Terjadi kesalahan saat memeriksa email."})
		return
	}
	if exists {
		c.JSON(http.StatusConflict, gin.H{"error": "Email ini sudah terdaftar. Silakan masuk atau gunakan email lain."})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengenkripsi kata sandi."})
		return
	}

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi pendaftaran."})
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Hanya role 'guru' yang dapat mendaftar melalui UI
	var user UserResponse
	err = tx.QueryRow(ctx, `
		INSERT INTO users (name, email, institution, password_hash, role, is_active)
		VALUES ($1, $2, $3, $4, 'guru', TRUE)
		RETURNING id, name, email, institution, role, is_active, created_at
	`, name, email, institution, string(hash)).Scan(
		&user.ID, &user.Name, &user.Email, &user.Institution, &user.Role, &user.IsActive, &user.CreatedAt,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan akun guru baru."})
		return
	}

	// Ambil Free Plan dan berikan otomatis dengan quota_reset_at 1 bulan ke depan
	var freePlanID int64
	var freePlanName string
	var freeQuota int
	err = tx.QueryRow(ctx, `
		SELECT id, name, quota_essays FROM plans WHERE code = 'free' LIMIT 1
	`).Scan(&freePlanID, &freePlanName, &freeQuota)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Konfigurasi Paket Gratis belum tersedia."})
		return
	}

	resetAt := time.Now().AddDate(0, 1, 0)
	_, err = tx.Exec(ctx, `
		INSERT INTO subscriptions (user_id, plan_id, quota_total, quota_used, quota_reset_at, expires_at)
		VALUES ($1, $2, $3, 0, $4, NULL)
	`, user.ID, freePlanID, freeQuota, resetAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengaktifkan Paket Gratis untuk akun baru."})
		return
	}

	if err := tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyelesaikan pendaftaran."})
		return
	}

	user.Subscription = &SubscriptionSummary{
		PlanCode:       "free",
		PlanName:       freePlanName,
		QuotaTotal:     freeQuota,
		QuotaUsed:      0,
		QuotaRemaining: freeQuota,
		QuotaResetAt:   &resetAt,
		ExpiresAt:      nil,
	}

	token, err := middleware.GenerateToken(h.cfg.JWTSecret, h.cfg.JWTExpiresHours, user.ID, user.Email, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat token sesi."})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"token": token,
		"user":  user,
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Masukkan alamat email dan kata sandi dengan benar."})
		return
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))
	ctx := c.Request.Context()

	var user UserResponse
	var passwordHash string
	err := h.pool.QueryRow(ctx, `
		SELECT id, name, email, institution, password_hash, role, is_active, created_at
		FROM users
		WHERE email = $1
	`, email).Scan(
		&user.ID, &user.Name, &user.Email, &user.Institution, &passwordHash, &user.Role, &user.IsActive, &user.CreatedAt,
	)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Email atau kata sandi salah."})
		return
	}

	if !user.IsActive {
		c.JSON(http.StatusForbidden, gin.H{"error": "Akun Anda sedang dinonaktifkan oleh Administrator."})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Email atau kata sandi salah."})
		return
	}

	if user.Role == "guru" {
		sub, _ := h.getOrRefreshSubscription(ctx, user.ID)
		user.Subscription = sub
	}

	token, err := middleware.GenerateToken(h.cfg.JWTSecret, h.cfg.JWTExpiresHours, user.ID, user.Email, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat token autentikasi."})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user":  user,
	})
}

func (h *AuthHandler) Me(c *gin.Context) {
	userID := c.GetInt64("user_id")
	ctx := c.Request.Context()

	var user UserResponse
	err := h.pool.QueryRow(ctx, `
		SELECT id, name, email, institution, role, is_active, created_at
		FROM users
		WHERE id = $1
	`, userID).Scan(
		&user.ID, &user.Name, &user.Email, &user.Institution, &user.Role, &user.IsActive, &user.CreatedAt,
	)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pengguna tidak ditemukan."})
		return
	}

	if !user.IsActive {
		c.JSON(http.StatusForbidden, gin.H{"error": "Akun Anda telah dinonaktifkan."})
		return
	}

	if user.Role == "guru" {
		sub, _ := h.getOrRefreshSubscription(ctx, user.ID)
		user.Subscription = sub
	}

	c.JSON(http.StatusOK, gin.H{"user": user})
}

func (h *AuthHandler) PublicPlans(c *gin.Context) {
	ctx := c.Request.Context()
	rows, err := h.pool.Query(ctx, `
		SELECT id, code, name, description, price_idr, quota_essays, duration_days, is_addon
		FROM plans
		WHERE is_active = TRUE
		ORDER BY sort_order ASC, id ASC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memuat daftar paket."})
		return
	}
	defer rows.Close()

	type planItem struct {
		ID           int64  `json:"id"`
		Code         string `json:"code"`
		Name         string `json:"name"`
		Description  string `json:"description"`
		PriceIDR     int    `json:"price_idr"`
		QuotaEssays  int    `json:"quota_essays"`
		DurationDays int    `json:"duration_days"`
		IsAddon      bool   `json:"is_addon"`
	}

	var plans []planItem
	for rows.Next() {
		var p planItem
		if err := rows.Scan(&p.ID, &p.Code, &p.Name, &p.Description, &p.PriceIDR, &p.QuotaEssays, &p.DurationDays, &p.IsAddon); err == nil {
			plans = append(plans, p)
		}
	}
	c.JSON(http.StatusOK, gin.H{"plans": plans})
}

func (h *AuthHandler) getOrRefreshSubscription(ctx context.Context, userID int64) (*SubscriptionSummary, error) {
	var sub SubscriptionSummary
	var planQuota int
	err := h.pool.QueryRow(ctx, `
		SELECT p.code, p.name, p.quota_essays, s.quota_total, s.quota_used, s.quota_reset_at, s.expires_at
		FROM subscriptions s
		JOIN plans p ON p.id = s.plan_id
		WHERE s.user_id = $1
	`, userID).Scan(
		&sub.PlanCode, &sub.PlanName, &planQuota, &sub.QuotaTotal, &sub.QuotaUsed, &sub.QuotaResetAt, &sub.ExpiresAt,
	)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	// Jika paket Free dan sudah melewati quota_reset_at, reset bulanan otomatis
	if sub.PlanCode == "free" && sub.QuotaResetAt != nil && now.After(*sub.QuotaResetAt) {
		nextReset := now.AddDate(0, 1, 0)
		_, _ = h.pool.Exec(ctx, `
			UPDATE subscriptions
			SET quota_used = 0, quota_total = $1, quota_reset_at = $2, updated_at = NOW()
			WHERE user_id = $3
		`, planQuota, nextReset, userID)
		sub.QuotaUsed = 0
		sub.QuotaTotal = planQuota
		sub.QuotaResetAt = &nextReset
	}

	rem := sub.QuotaTotal - sub.QuotaUsed
	if rem < 0 {
		rem = 0
	}
	sub.QuotaRemaining = rem
	return &sub, nil
}

