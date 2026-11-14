package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"koreksiin-backend/internal/config"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type AdminHandler struct {
	pool           *pgxpool.Pool
	rdb            *redis.Client
	cfg            *config.Config
	paymentHandler *PaymentHandler
}

func NewAdminHandler(pool *pgxpool.Pool, rdb *redis.Client, cfg *config.Config, paymentHandler *PaymentHandler) *AdminHandler {
	return &AdminHandler{pool: pool, rdb: rdb, cfg: cfg, paymentHandler: paymentHandler}
}

func (h *AdminHandler) Overview(c *gin.Context) {
	ctx := c.Request.Context()

	var totalTeachers, activeTeachers, totalAssignments, totalScoredEssays, totalCustomModels int
	var totalRevenueIDR, settledTransactions int64

	_ = h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE role = 'guru'`).Scan(&totalTeachers)
	_ = h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE role = 'guru' AND is_active = TRUE`).Scan(&activeTeachers)
	_ = h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM assignments`).Scan(&totalAssignments)
	_ = h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM submissions WHERE status = 'scored'`).Scan(&totalScoredEssays)
	_ = h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM custom_models`).Scan(&totalCustomModels)
	_ = h.pool.QueryRow(ctx, `
		SELECT COUNT(*), COALESCE(SUM(gross_amount), 0)
		FROM payment_transactions
		WHERE status = 'settlement'
	`).Scan(&settledTransactions, &totalRevenueIDR)

	type PlanDistItem struct {
		PlanCode     string `json:"plan_code"`
		PlanName     string `json:"plan_name"`
		TeacherCount int    `json:"teacher_count"`
	}
	planDist := make([]PlanDistItem, 0)
	rows, err := h.pool.Query(ctx, `
		SELECT p.code, p.name, COUNT(s.id) AS teacher_count
		FROM plans p
		LEFT JOIN subscriptions s ON s.plan_id = p.id
		WHERE p.is_addon = FALSE
		GROUP BY p.id, p.code, p.name, p.sort_order
		ORDER BY p.sort_order ASC
	`)
	if err == nil {
		for rows.Next() {
			var item PlanDistItem
			if err := rows.Scan(&item.PlanCode, &item.PlanName, &item.TeacherCount); err == nil {
				planDist = append(planDist, item)
			}
		}
		rows.Close()
	}

	scoringHeartbeat, _ := h.rdb.Get(ctx, "worker:heartbeat:scoring").Result()
	trainingHeartbeat, _ := h.rdb.Get(ctx, "worker:heartbeat:training").Result()

	notifURL := ""
	notifSource := "dashboard_default"
	if h.paymentHandler != nil {
		notifURL, notifSource = h.paymentHandler.ResolveNotificationURL(false)
	}

	c.JSON(http.StatusOK, gin.H{
		"total_teachers":          totalTeachers,
		"active_teachers":         activeTeachers,
		"inactive_teachers":       totalTeachers - activeTeachers,
		"total_assignments":       totalAssignments,
		"total_scored_essays":     totalScoredEssays,
		"total_custom_models":     totalCustomModels,
		"settled_transactions":    settledTransactions,
		"total_revenue_idr":       totalRevenueIDR,
		"plan_distribution":       planDist,
		"notification_url":        notifURL,
		"notification_url_source": notifSource,
		"workers": gin.H{
			"scoring_online":  scoringHeartbeat != "",
			"training_online": trainingHeartbeat != "",
		},
	})
}

func (h *AdminHandler) ListTeachers(c *gin.Context) {
	ctx := c.Request.Context()

	rows, err := h.pool.Query(ctx, `
		SELECT
			u.id, u.name, u.email, u.institution, u.is_active, u.created_at,
			COALESCE(p.code, 'free') AS plan_code,
			COALESCE(p.name, 'Paket Gratis') AS plan_name,
			COALESCE(s.quota_total, 0) AS quota_total,
			COALESCE(s.quota_used, 0) AS quota_used,
			s.expires_at,
			(SELECT COUNT(*) FROM assignments a WHERE a.user_id = u.id) AS assignment_count,
			(SELECT COUNT(*) FROM custom_models cm WHERE cm.user_id = u.id) AS model_count
		FROM users u
		LEFT JOIN subscriptions s ON s.user_id = u.id
		LEFT JOIN plans p ON p.id = s.plan_id
		WHERE u.role = 'guru'
		ORDER BY u.created_at DESC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memuat daftar guru."})
		return
	}
	defer rows.Close()

	type TeacherAdminItem struct {
		ID              int64      `json:"id"`
		Name            string     `json:"name"`
		Email           string     `json:"email"`
		Institution     string     `json:"institution"`
		IsActive        bool       `json:"is_active"`
		PlanCode        string     `json:"plan_code"`
		PlanName        string     `json:"plan_name"`
		QuotaTotal      int        `json:"quota_total"`
		QuotaUsed       int        `json:"quota_used"`
		QuotaRemaining  int        `json:"quota_remaining"`
		ExpiresAt       *time.Time `json:"expires_at"`
		AssignmentCount int        `json:"assignment_count"`
		ModelCount      int        `json:"model_count"`
		CreatedAt       time.Time  `json:"created_at"`
	}

	teachers := make([]TeacherAdminItem, 0)
	for rows.Next() {
		var t TeacherAdminItem
		if err := rows.Scan(
			&t.ID, &t.Name, &t.Email, &t.Institution, &t.IsActive, &t.CreatedAt,
			&t.PlanCode, &t.PlanName, &t.QuotaTotal, &t.QuotaUsed, &t.ExpiresAt,
			&t.AssignmentCount, &t.ModelCount,
		); err == nil {
			rem := t.QuotaTotal - t.QuotaUsed
			if rem < 0 {
				rem = 0
			}
			t.QuotaRemaining = rem
			teachers = append(teachers, t)
		}
	}

	c.JSON(http.StatusOK, gin.H{"teachers": teachers})
}

func (h *AdminHandler) UpdateTeacherStatus(c *gin.Context) {
	teacherID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID guru tidak valid."})
		return
	}

	var req struct {
		IsActive bool `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Field is_active wajib dikirim."})
		return
	}

	ctx := c.Request.Context()
	cmdTag, err := h.pool.Exec(ctx, `
		UPDATE users
		SET is_active = $1, updated_at = NOW()
		WHERE id = $2 AND role = 'guru'
	`, req.IsActive, teacherID)
	if err != nil || cmdTag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Akun guru tidak ditemukan."})
		return
	}

	statusLabel := "diaktifkan"
	if !req.IsActive {
		statusLabel = "dinonaktifkan"
	}
	c.JSON(http.StatusOK, gin.H{
		"message":   "Akun guru berhasil " + statusLabel + ".",
		"is_active": req.IsActive,
	})
}

func (h *AdminHandler) ListTransactions(c *gin.Context) {
	ctx := c.Request.Context()

	rows, err := h.pool.Query(ctx, `
		SELECT
			pt.id, pt.order_id, u.name, u.email, p.code, p.name,
			pt.gross_amount, pt.status, pt.payment_type, pt.paid_at, pt.created_at
		FROM payment_transactions pt
		JOIN users u ON u.id = pt.user_id
		JOIN plans p ON p.id = pt.plan_id
		ORDER BY pt.created_at DESC
		LIMIT 200
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memuat daftar transaksi."})
		return
	}
	defer rows.Close()

	type AdminTxItem struct {
		ID           int64      `json:"id"`
		OrderID      string     `json:"order_id"`
		TeacherName  string     `json:"teacher_name"`
		TeacherEmail string     `json:"teacher_email"`
		PlanCode     string     `json:"plan_code"`
		PlanName     string     `json:"plan_name"`
		GrossAmount  int        `json:"gross_amount"`
		Status       string     `json:"status"`
		PaymentType  string     `json:"payment_type"`
		PaidAt       *time.Time `json:"paid_at"`
		CreatedAt    time.Time  `json:"created_at"`
	}

	transactions := make([]AdminTxItem, 0)
	for rows.Next() {
		var item AdminTxItem
		if err := rows.Scan(
			&item.ID, &item.OrderID, &item.TeacherName, &item.TeacherEmail,
			&item.PlanCode, &item.PlanName, &item.GrossAmount, &item.Status,
			&item.PaymentType, &item.PaidAt, &item.CreatedAt,
		); err == nil {
			transactions = append(transactions, item)
		}
	}
	rows.Close()

	type WebhookLogItem struct {
		ID                    int64     `json:"id"`
		OrderID               string    `json:"order_id"`
		StatusCode            string    `json:"status_code"`
		TransactionStatus     string    `json:"transaction_status"`
		SignatureValid        bool      `json:"signature_valid"`
		ProcessedIdempotently bool      `json:"processed_idempotently"`
		ProcessedNote         string    `json:"processed_note"`
		CreatedAt             time.Time `json:"created_at"`
	}

	logs := make([]WebhookLogItem, 0)
	lRows, err := h.pool.Query(ctx, `
		SELECT
			id, order_id, status_code, transaction_status,
			COALESCE(signature_valid, FALSE), processed_idempotently,
			COALESCE(processed_note, ''), created_at
		FROM payment_notification_logs
		ORDER BY created_at DESC
		LIMIT 50
	`)
	if err == nil {
		for lRows.Next() {
			var l WebhookLogItem
			if err := lRows.Scan(
				&l.ID, &l.OrderID, &l.StatusCode, &l.TransactionStatus,
				&l.SignatureValid, &l.ProcessedIdempotently, &l.ProcessedNote, &l.CreatedAt,
			); err == nil {
				logs = append(logs, l)
			}
		}
		lRows.Close()
	}

	c.JSON(http.StatusOK, gin.H{
		"transactions":  transactions,
		"webhook_logs":  logs,
	})
}

func (h *AdminHandler) ListPlans(c *gin.Context) {
	ctx := c.Request.Context()
	rows, err := h.pool.Query(ctx, `
		SELECT id, code, name, description, price_idr, quota_essays, duration_days, is_addon, is_active, sort_order
		FROM plans
		ORDER BY sort_order ASC, id ASC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memuat daftar paket."})
		return
	}
	defer rows.Close()

	plans := make([]map[string]any, 0)
	for rows.Next() {
		var id int64
		var code, name, desc string
		var price, quota, duration, sortOrder int
		var isAddon, isActive bool
		if err := rows.Scan(&id, &code, &name, &desc, &price, &quota, &duration, &isAddon, &isActive, &sortOrder); err == nil {
			plans = append(plans, map[string]any{
				"id":            id,
				"code":          code,
				"name":          name,
				"description":   desc,
				"price_idr":     price,
				"quota_essays":  quota,
				"duration_days": duration,
				"is_addon":      isAddon,
				"is_active":     isActive,
				"sort_order":    sortOrder,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{"plans": plans})
}

func (h *AdminHandler) UpdatePlan(c *gin.Context) {
	planID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID paket tidak valid."})
		return
	}

	var req struct {
		Name         string `json:"name" binding:"required"`
		Description  string `json:"description"`
		PriceIDR     int    `json:"price_idr"`
		QuotaEssays  int    `json:"quota_essays" binding:"required"`
		DurationDays int    `json:"duration_days"`
		IsActive     bool   `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data paket tidak valid."})
		return
	}

	if req.PriceIDR < 0 || req.QuotaEssays <= 0 || req.DurationDays < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Harga, kuota esai, dan durasi hari harus bernilai positif."})
		return
	}

	ctx := c.Request.Context()
	cmdTag, err := h.pool.Exec(ctx, `
		UPDATE plans
		SET name = $1,
		    description = $2,
		    price_idr = $3,
		    quota_essays = $4,
		    duration_days = $5,
		    is_active = $6,
		    updated_at = NOW()
		WHERE id = $7
	`, strings.TrimSpace(req.Name), strings.TrimSpace(req.Description), req.PriceIDR, req.QuotaEssays, req.DurationDays, req.IsActive, planID)
	if err != nil || cmdTag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Paket tidak ditemukan."})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Pengaturan paket berhasil diperbarui."})
}

func (h *AdminHandler) GetAISettings(c *gin.Context) {
	ctx := c.Request.Context()
	var tHigh, tLow, sSem, sLex float64
	var minTrainSamples int
	var updatedAt time.Time

	err := h.pool.QueryRow(ctx, `
		SELECT t_high, t_low, s_sem, s_lex, min_train_samples, updated_at
		FROM ai_settings
		WHERE id = 1
	`).Scan(&tHigh, &tLow, &sSem, &sLex, &minTrainSamples, &updatedAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memuat pengaturan ambang AI."})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ai_settings": gin.H{
			"t_high":            tHigh,
			"t_low":             tLow,
			"s_sem":             sSem,
			"s_lex":             sLex,
			"min_train_samples": minTrainSamples,
			"base_model_name":   "LazarusNLP/all-indobert-base-v4",
			"updated_at":        updatedAt,
		},
	})
}

func (h *AdminHandler) UpdateAISettings(c *gin.Context) {
	var req struct {
		THigh           float64 `json:"t_high"`
		TLow            float64 `json:"t_low"`
		SSem            float64 `json:"s_sem"`
		SLex            float64 `json:"s_lex"`
		MinTrainSamples int     `json:"min_train_samples"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format pengaturan AI tidak valid."})
		return
	}

	if req.TLow <= 0 || req.THigh <= req.TLow || req.THigh > 1.0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Ambang cakupan harus memenuhi 0 < T_LOW < T_HIGH <= 1.0.",
		})
		return
	}
	if req.SSem <= 0 || req.SSem > 1.0 || req.SLex <= 0 || req.SLex > 1.0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Ambang kemiripan S_SEM dan S_LEX harus berada di antara 0 dan 1.0.",
		})
		return
	}
	if req.MinTrainSamples < 5 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Minimal sampel pelatihan tidak boleh kurang dari 5.",
		})
		return
	}

	ctx := c.Request.Context()
	_, err := h.pool.Exec(ctx, `
		UPDATE ai_settings
		SET t_high = $1,
		    t_low = $2,
		    s_sem = $3,
		    s_lex = $4,
		    min_train_samples = $5,
		    updated_at = NOW()
		WHERE id = 1
	`, req.THigh, req.TLow, req.SSem, req.SLex, req.MinTrainSamples)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan pengaturan ambang AI."})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Pengaturan ambang AI berhasil disimpan."})
}

