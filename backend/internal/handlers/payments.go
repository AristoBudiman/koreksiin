package handlers

import (
	"bytes"
	"context"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"koreksiin-backend/internal/config"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PaymentHandler struct {
	pool *pgxpool.Pool
	cfg  *config.Config

	mu           sync.Mutex
	cachedURL    string
	cachedSource string
	cachedAt     time.Time
}

func NewPaymentHandler(pool *pgxpool.Pool, cfg *config.Config) *PaymentHandler {
	return &PaymentHandler{pool: pool, cfg: cfg}
}

type CheckoutInput struct {
	PlanCode string `json:"plan_code" binding:"required"`
}

type TransactionItem struct {
	ID              int64      `json:"id"`
	OrderID         string     `json:"order_id"`
	PlanID          int64      `json:"plan_id"`
	PlanCode        string     `json:"plan_code"`
	PlanName        string     `json:"plan_name"`
	QuotaEssays     int        `json:"quota_essays"`
	IsAddon         bool       `json:"is_addon"`
	GrossAmount     int        `json:"gross_amount"`
	SnapToken       string     `json:"snap_token"`
	SnapRedirectURL string     `json:"snap_redirect_url"`
	Status          string     `json:"status"`
	PaymentType     string     `json:"payment_type"`
	PaidAt          *time.Time `json:"paid_at"`
	CreatedAt       time.Time  `json:"created_at"`
}

type MidtransNotificationPayload struct {
	OrderID           string `json:"order_id"`
	StatusCode        string `json:"status_code"`
	GrossAmount       string `json:"gross_amount"`
	SignatureKey      string `json:"signature_key"`
	TransactionStatus string `json:"transaction_status"`
	FraudStatus       string `json:"fraud_status"`
	PaymentType       string `json:"payment_type"`
}

// ResolveNotificationURL menentukan URL notifikasi webhook Midtrans secara berurutan:
// 1. Jika env PUBLIC_BASE_URL diisi, gunakan <PUBLIC_BASE_URL>/api/payments/midtrans/notification.
// 2. Jika tidak, ambil hostname aktif dari http://tunnel:2000/quicktunnel (cache 60 detik).
// 3. Jika endpoint tunnel tidak dapat diakses, kembalikan string kosong dan catat peringatan di log.
func (h *PaymentHandler) ResolveNotificationURL(forceRefresh bool) (string, string) {
	if pub := strings.TrimSpace(h.cfg.PublicBaseURL); pub != "" {
		cleanBase := strings.TrimRight(pub, "/")
		if !strings.HasPrefix(cleanBase, "https://") && !strings.HasPrefix(cleanBase, "http://") {
			cleanBase = "https://" + cleanBase
		}
		return cleanBase + "/api/payments/midtrans/notification", "public_base_url"
	}

	h.mu.Lock()
	if !forceRefresh && h.cachedURL != "" && time.Since(h.cachedAt) < 60*time.Second {
		url := h.cachedURL
		src := h.cachedSource
		h.mu.Unlock()
		return url, src
	}
	h.mu.Unlock()

	metricsURL := strings.TrimSpace(h.cfg.TunnelMetricsURL)
	if metricsURL == "" {
		metricsURL = "http://tunnel:2000/quicktunnel"
	}

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(metricsURL)
	if err != nil {
		log.Printf("[Payments] Peringatan: endpoint tunnel (%s) tidak dapat diakses (%v). Header X-Override-Notification tidak dikirim (menggunakan URL dashboard Midtrans).", metricsURL, err)
		return "", "dashboard_default"
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf("[Payments] Peringatan: endpoint tunnel (%s) mengembalikan status %d. Header X-Override-Notification tidak dikirim (menggunakan URL dashboard Midtrans).", metricsURL, resp.StatusCode)
		return "", "dashboard_default"
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[Payments] Peringatan: gagal membaca respons endpoint tunnel (%s): %v. Header X-Override-Notification tidak dikirim.", metricsURL, err)
		return "", "dashboard_default"
	}

	var qt struct {
		Hostname string `json:"hostname"`
	}
	if err := json.Unmarshal(bodyBytes, &qt); err != nil {
		log.Printf("[Payments] Peringatan: respons JSON endpoint tunnel (%s) tidak valid: %v. Header X-Override-Notification tidak dikirim.", metricsURL, err)
		return "", "dashboard_default"
	}

	host := strings.TrimSpace(qt.Hostname)
	host = strings.TrimPrefix(host, "https://")
	host = strings.TrimPrefix(host, "http://")
	host = strings.Trim(host, "/")

	if host == "" || strings.Contains(host, "/") {
		log.Printf("[Payments] Peringatan: hostname tunnel kosong atau belum siap dari %s. Header X-Override-Notification tidak dikirim.", metricsURL)
		return "", "dashboard_default"
	}

	notifURL := fmt.Sprintf("https://%s/api/payments/midtrans/notification", host)

	h.mu.Lock()
	h.cachedURL = notifURL
	h.cachedSource = "tunnel"
	h.cachedAt = time.Now()
	h.mu.Unlock()

	return notifURL, "tunnel"
}

func (h *PaymentHandler) GetConfig(c *gin.Context) {
	snapURL := "https://app.sandbox.midtrans.com/snap/snap.js"
	if h.cfg.MidtransIsProduction {
		snapURL = "https://app.midtrans.com/snap/snap.js"
	}
	c.JSON(http.StatusOK, gin.H{
		"client_key":    h.cfg.MidtransClientKey,
		"is_production": h.cfg.MidtransIsProduction,
		"snap_js_url":   snapURL,
		"configured":    h.cfg.MidtransServerKey != "" && h.cfg.MidtransClientKey != "",
	})
}

func (h *PaymentHandler) ListGuruTransactions(c *gin.Context) {
	userID := c.GetInt64("user_id")
	ctx := c.Request.Context()

	reconcileParam := strings.TrimSpace(c.Query("reconcile"))
	if reconcileParam == "1" || strings.EqualFold(reconcileParam, "true") {
		_, _, _ = h.reconcilePendingForUser(ctx, userID)
	}

	rows, err := h.pool.Query(ctx, `
		SELECT
			pt.id, pt.order_id, pt.plan_id, p.code, p.name, p.quota_essays, p.is_addon,
			pt.gross_amount, pt.snap_token, pt.snap_redirect_url,
			pt.status, pt.payment_type, pt.paid_at, pt.created_at
		FROM payment_transactions pt
		JOIN plans p ON p.id = pt.plan_id
		WHERE pt.user_id = $1
		ORDER BY pt.created_at DESC
	`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memuat riwayat transaksi."})
		return
	}
	defer rows.Close()

	items := make([]TransactionItem, 0)
	for rows.Next() {
		var t TransactionItem
		if err := rows.Scan(
			&t.ID, &t.OrderID, &t.PlanID, &t.PlanCode, &t.PlanName, &t.QuotaEssays, &t.IsAddon,
			&t.GrossAmount, &t.SnapToken, &t.SnapRedirectURL,
			&t.Status, &t.PaymentType, &t.PaidAt, &t.CreatedAt,
		); err == nil {
			items = append(items, t)
		}
	}

	c.JSON(http.StatusOK, gin.H{"transactions": items})
}

func (h *PaymentHandler) ReconcileGuruTransactions(c *gin.Context) {
	userID := c.GetInt64("user_id")
	ctx := c.Request.Context()

	checkedCount, settledCount, updatedCount := h.reconcilePendingForUser(ctx, userID)

	msg := "Tidak ada transaksi pending yang berubah status."
	if checkedCount == 0 {
		msg = "Tidak ada transaksi berstatus menunggu pembayaran (pending)."
	} else if settledCount > 0 {
		msg = fmt.Sprintf("Berhasil memverifikasi %d pembayaran lunas. Kuota langganan Anda telah diperbarui!", settledCount)
	} else if updatedCount > 0 {
		msg = fmt.Sprintf("Status %d transaksi berhasil disinkronkan dari Midtrans.", updatedCount)
	} else {
		msg = fmt.Sprintf("Telah memeriksa %d transaksi pending ke Midtrans: pembayaran belum diselesaikan.", checkedCount)
	}

	c.JSON(http.StatusOK, gin.H{
		"checked_count": checkedCount,
		"settled_count": settledCount,
		"updated_count": updatedCount,
		"message":       msg,
	})
}

func (h *PaymentHandler) reconcilePendingForUser(ctx context.Context, userID int64) (int, int, int) {
	if strings.TrimSpace(h.cfg.MidtransServerKey) == "" {
		return 0, 0, 0
	}

	rows, err := h.pool.Query(ctx, `
		SELECT order_id
		FROM payment_transactions
		WHERE user_id = $1 AND status = 'pending'
		ORDER BY created_at DESC
		LIMIT 100
	`, userID)
	if err != nil {
		return 0, 0, 0
	}
	var pendingOrders []string
	for rows.Next() {
		var oid string
		if err := rows.Scan(&oid); err == nil && oid != "" {
			pendingOrders = append(pendingOrders, oid)
		}
	}
	rows.Close()

	checkedCount := len(pendingOrders)
	settledCount := 0
	updatedCount := 0

	for _, oid := range pendingOrders {
		changed, newStatus, err := h.reconcileOrderWithStatusAPI(ctx, oid)
		if err == nil && changed {
			updatedCount++
			if newStatus == "settlement" {
				settledCount++
			}
		}
	}

	return checkedCount, settledCount, updatedCount
}

func extractPaymentMethodDetail(data map[string]any, fallbackType string) string {
	pt := strings.ToLower(strings.TrimSpace(fallbackType))
	if data != nil {
		if v, ok := data["payment_type"].(string); ok && strings.TrimSpace(v) != "" {
			pt = strings.ToLower(strings.TrimSpace(v))
		}
		switch pt {
		case "bank_transfer":
			if vaArr, ok := data["va_numbers"].([]any); ok && len(vaArr) > 0 {
				if firstVA, ok := vaArr[0].(map[string]any); ok {
					if bank, ok := firstVA["bank"].(string); ok && strings.TrimSpace(bank) != "" {
						return "va_" + strings.ToLower(strings.TrimSpace(bank))
					}
				}
			}
			if pva, ok := data["permata_va_number"].(string); ok && strings.TrimSpace(pva) != "" {
				return "va_permata"
			}
			if bva, ok := data["bca_va_number"].(string); ok && strings.TrimSpace(bva) != "" {
				return "va_bca"
			}
			return "bank_transfer"
		case "echannel":
			return "echannel"
		case "cstore":
			if store, ok := data["store"].(string); ok && strings.TrimSpace(store) != "" {
				return "cstore_" + strings.ToLower(strings.TrimSpace(store))
			}
			return "cstore"
		}
	}
	return pt
}

func (h *PaymentHandler) reconcileOrderWithStatusAPI(ctx context.Context, orderID string) (bool, string, error) {
	apiData, err := h.verifyWithMidtransStatusAPI(ctx, orderID)
	if err != nil || apiData == nil {
		return false, "", err
	}

	apiStatusCode, _ := apiData["status_code"].(string)
	if apiStatusCode != "200" && apiStatusCode != "201" && apiStatusCode != "202" {
		// Misal 404 saat pengguna baru membuka popup Snap tapi belum memilih metode pembayaran
		return false, "pending", nil
	}

	effectiveStatus, _ := apiData["transaction_status"].(string)
	effectiveFraud, _ := apiData["fraud_status"].(string)
	effectiveStatus = strings.TrimSpace(effectiveStatus)
	effectiveFraud = strings.TrimSpace(effectiveFraud)
	effectivePaymentType := extractPaymentMethodDetail(apiData, "")

	if effectiveStatus == "" {
		return false, "pending", nil
	}

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		return false, "", err
	}
	defer tx.Rollback(ctx)

	var dbTxID, dbUserID, dbPlanID int64
	var dbGrossAmount, planQuotaEssays, planDurationDays int
	var dbStatus, dbPaymentType, planCode string
	var planIsAddon bool

	err = tx.QueryRow(ctx, `
		SELECT
			pt.id, pt.user_id, pt.plan_id, pt.gross_amount, pt.status, COALESCE(pt.payment_type, ''),
			p.code, p.quota_essays, p.duration_days, p.is_addon
		FROM payment_transactions pt
		JOIN plans p ON p.id = pt.plan_id
		WHERE pt.order_id = $1
		FOR UPDATE OF pt
	`, orderID).Scan(
		&dbTxID, &dbUserID, &dbPlanID, &dbGrossAmount, &dbStatus, &dbPaymentType,
		&planCode, &planQuotaEssays, &planDurationDays, &planIsAddon,
	)
	if err != nil {
		return false, "", err
	}

	// Idempoten: jika sudah settlement/capture, jangan proses ulang
	if dbStatus == "settlement" || dbStatus == "capture" {
		return false, dbStatus, nil
	}

	if gaStr, ok := apiData["gross_amount"].(string); ok && gaStr != "" {
		if apiGross, err := strconv.ParseFloat(gaStr, 64); err == nil {
			if math.Abs(apiGross-float64(dbGrossAmount)) > 0.01 {
				return false, dbStatus, fmt.Errorf("nominal gross_amount Status API tidak cocok")
			}
		}
	}

	rawBytes, _ := json.Marshal(apiData)
	isPaid := effectiveStatus == "settlement" || (effectiveStatus == "capture" && (effectiveFraud == "" || effectiveFraud == "accept"))

	if isPaid {
		if err := h.applySettlementTx(ctx, tx, dbTxID, dbUserID, dbPlanID, planQuotaEssays, planDurationDays, planIsAddon, effectivePaymentType); err != nil {
			return false, dbStatus, err
		}
		if err := tx.Commit(ctx); err != nil {
			return false, dbStatus, err
		}
		note := fmt.Sprintf("Rekonsiliasi Status API: settlement (%s), +%d kuota esai", effectivePaymentType, planQuotaEssays)
		h.logWebhookNotificationCtx(ctx, orderID, apiStatusCode, "settlement", "reconcile-status-api", true, false, note, rawBytes)
		return true, "settlement", nil
	}

	statusChanged := effectiveStatus != dbStatus
	paymentTypeChanged := effectivePaymentType != "" && effectivePaymentType != dbPaymentType

	if statusChanged || paymentTypeChanged {
		_, err = tx.Exec(ctx, `
			UPDATE payment_transactions
			SET status = $1,
			    payment_type = COALESCE(NULLIF($2, ''), payment_type),
			    updated_at = NOW()
			WHERE id = $3
		`, effectiveStatus, effectivePaymentType, dbTxID)
		if err != nil {
			return false, dbStatus, err
		}
		if err := tx.Commit(ctx); err != nil {
			return false, dbStatus, err
		}
		note := fmt.Sprintf("Rekonsiliasi Status API: status='%s', metode='%s'", effectiveStatus, effectivePaymentType)
		h.logWebhookNotificationCtx(ctx, orderID, apiStatusCode, effectiveStatus, "reconcile-status-api", true, false, note, rawBytes)
		return true, effectiveStatus, nil
	}

	return false, dbStatus, nil
}

func (h *PaymentHandler) applySettlementTx(
	ctx context.Context,
	tx pgx.Tx,
	dbTxID int64,
	dbUserID int64,
	dbPlanID int64,
	planQuotaEssays int,
	planDurationDays int,
	planIsAddon bool,
	effectivePaymentType string,
) error {
	now := time.Now()
	curState, freePlan, err := refreshOrExpireSubscription(ctx, tx, dbUserID, now)
	if err != nil {
		return err
	}

	var targetPlan PlanSpec
	err = tx.QueryRow(ctx, `
		SELECT id, code, name, quota_essays, duration_days, is_addon, price_idr
		FROM plans
		WHERE id = $1
	`, dbPlanID).Scan(
		&targetPlan.ID, &targetPlan.Code, &targetPlan.Name,
		&targetPlan.QuotaEssays, &targetPlan.DurationDays, &targetPlan.IsAddon, &targetPlan.PriceIDR,
	)
	if err != nil {
		return err
	}

	outcome, err := EvaluateSubscriptionPurchase(curState, targetPlan, freePlan, false, now)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		UPDATE payment_transactions
		SET status = 'settlement',
		    payment_type = $1,
		    paid_at = NOW(),
		    updated_at = NOW()
		WHERE id = $2
	`, effectivePaymentType, dbTxID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		UPDATE subscriptions
		SET plan_id = $1,
		    quota_total = $2,
		    quota_used = $3,
		    quota_reset_at = NULL,
		    expires_at = $4,
		    updated_at = NOW()
		WHERE user_id = $5
	`, outcome.NewPlanID, outcome.NewQuotaTotal, outcome.NewQuotaUsed, outcome.NewExpiresAt, dbUserID)
	return err
}

func (h *PaymentHandler) CancelPendingTransaction(c *gin.Context) {
	userID := c.GetInt64("user_id")
	txID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || txID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID transaksi tidak valid."})
		return
	}

	ctx := c.Request.Context()
	tag, err := h.pool.Exec(ctx, `
		UPDATE payment_transactions
		SET status = 'cancel',
		    updated_at = NOW()
		WHERE id = $1 AND user_id = $2 AND status = 'pending'
	`, txID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membatalkan transaksi."})
		return
	}
	if tag.RowsAffected() == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Transaksi pending tidak ditemukan atau sudah diproses."})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Transaksi pending berhasil dibatalkan."})
}

func (h *PaymentHandler) Checkout(c *gin.Context) {
	userID := c.GetInt64("user_id")
	var req CheckoutInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Kode paket (plan_code) wajib dipilih."})
		return
	}

	if strings.TrimSpace(h.cfg.MidtransServerKey) == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Konfigurasi Midtrans Sandbox belum diatur di server.",
		})
		return
	}

	ctx := c.Request.Context()

	var targetPlan PlanSpec
	err := h.pool.QueryRow(ctx, `
		SELECT id, code, name, price_idr, quota_essays, duration_days, is_addon
		FROM plans
		WHERE code = $1 AND is_active = TRUE
	`, strings.TrimSpace(req.PlanCode)).Scan(
		&targetPlan.ID, &targetPlan.Code, &targetPlan.Name, &targetPlan.PriceIDR, &targetPlan.QuotaEssays, &targetPlan.DurationDays, &targetPlan.IsAddon,
	)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Paket langganan tidak ditemukan atau sedang tidak aktif."})
		return
	}

	// Cek transaksi pending yang masih aktif
	var pendingTx TransactionItem
	pendingErr := h.pool.QueryRow(ctx, `
		SELECT
			pt.id, pt.order_id, pt.plan_id, p.code, p.name, p.quota_essays, p.is_addon,
			pt.gross_amount, pt.snap_token, pt.snap_redirect_url,
			pt.status, COALESCE(pt.payment_type, ''), pt.paid_at, pt.created_at
		FROM payment_transactions pt
		JOIN plans p ON p.id = pt.plan_id
		WHERE pt.user_id = $1 AND pt.status = 'pending'
		ORDER BY pt.created_at DESC
		LIMIT 1
	`, userID).Scan(
		&pendingTx.ID, &pendingTx.OrderID, &pendingTx.PlanID, &pendingTx.PlanCode, &pendingTx.PlanName, &pendingTx.QuotaEssays, &pendingTx.IsAddon,
		&pendingTx.GrossAmount, &pendingTx.SnapToken, &pendingTx.SnapRedirectURL,
		&pendingTx.Status, &pendingTx.PaymentType, &pendingTx.PaidAt, &pendingTx.CreatedAt,
	)
	if pendingErr == nil {
		c.JSON(http.StatusConflict, gin.H{
			"error":               "Anda masih memiliki transaksi yang menunggu pembayaran. Silakan lanjutkan atau batalkan transaksi tersebut terlebih dahulu.",
			"pending_transaction": pendingTx,
		})
		return
	}

	now := time.Now()
	curState, freePlan, err := refreshOrExpireSubscription(ctx, h.pool, userID, now)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memuat data langganan Anda."})
		return
	}

	if _, err := EvaluateSubscriptionPurchase(curState, targetPlan, freePlan, false, now); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	planID := targetPlan.ID
	planCode := targetPlan.Code
	planName := targetPlan.Name
	priceIDR := targetPlan.PriceIDR

	var userName, userEmail string
	err = h.pool.QueryRow(ctx, `
		SELECT name, email FROM users WHERE id = $1
	`, userID).Scan(&userName, &userEmail)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Data pengguna tidak ditemukan."})
		return
	}

	orderID := fmt.Sprintf("KOR-%d-%d-%04d", time.Now().Unix(), userID, time.Now().Nanosecond()%10000)

	snapEndpoint := "https://app.sandbox.midtrans.com/snap/v1/transactions"
	if h.cfg.MidtransIsProduction {
		snapEndpoint = "https://app.midtrans.com/snap/v1/transactions"
	}

	snapReqBody := map[string]any{
		"transaction_details": map[string]any{
			"order_id":     orderID,
			"gross_amount": priceIDR,
		},
		"customer_details": map[string]any{
			"first_name": userName,
			"email":      userEmail,
		},
		"item_details": []map[string]any{
			{
				"id":       planCode,
				"price":    priceIDR,
				"quantity": 1,
				"name":     planName,
			},
		},
	}

	rawBody, _ := json.Marshal(snapReqBody)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, snapEndpoint, bytes.NewReader(rawBody))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyiapkan permintaan ke Midtrans."})
		return
	}

	basicAuth := base64.StdEncoding.EncodeToString([]byte(h.cfg.MidtransServerKey + ":"))
	httpReq.Header.Set("Authorization", "Basic "+basicAuth)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	notifURL, notifSource := h.ResolveNotificationURL(false)
	if notifURL != "" {
		httpReq.Header.Set("X-Override-Notification", notifURL)
		log.Printf("[Payments] Mengirim X-Override-Notification (%s) untuk order %s: %s", notifSource, orderID, notifURL)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Gagal menghubungi server Midtrans Sandbox."})
		return
	}
	defer resp.Body.Close()

	respBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		c.JSON(http.StatusBadGateway, gin.H{
			"error": fmt.Sprintf("Midtrans mengembalikan status %d saat membuat transaksi Snap.", resp.StatusCode),
		})
		return
	}

	var snapResp struct {
		Token       string `json:"token"`
		RedirectURL string `json:"redirect_url"`
	}
	if err := json.Unmarshal(respBytes, &snapResp); err != nil || snapResp.Token == "" {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Respons token Snap Midtrans tidak valid."})
		return
	}

	var txID int64
	err = h.pool.QueryRow(ctx, `
		INSERT INTO payment_transactions (
			order_id, user_id, plan_id, gross_amount, snap_token, snap_redirect_url, status
		)
		VALUES ($1, $2, $3, $4, $5, $6, 'pending')
		RETURNING id
	`, orderID, userID, planID, priceIDR, snapResp.Token, snapResp.RedirectURL).Scan(&txID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan transaksi pembayaran."})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":             "Transaksi berhasil dibuat. Silakan selesaikan pembayaran melalui Midtrans Snap.",
		"transaction_id":      txID,
		"order_id":            orderID,
		"plan_code":           planCode,
		"plan_name":           planName,
		"gross_amount":        priceIDR,
		"snap_token":          snapResp.Token,
		"snap_redirect_url":   snapResp.RedirectURL,
		"notification_url":    notifURL,
		"notification_source": notifSource,
	})
}

func (h *PaymentHandler) verifyWithMidtransStatusAPI(ctx context.Context, orderID string) (map[string]any, error) {
	baseAPI := "https://api.sandbox.midtrans.com"
	if h.cfg.MidtransIsProduction {
		baseAPI = "https://api.midtrans.com"
	}
	url := fmt.Sprintf("%s/v2/%s/status", baseAPI, orderID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	basicAuth := base64.StdEncoding.EncodeToString([]byte(h.cfg.MidtransServerKey + ":"))
	req.Header.Set("Authorization", "Basic "+basicAuth)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data map[string]any
	if err := json.Unmarshal(bodyBytes, &data); err != nil {
		return nil, err
	}
	return data, nil
}

func (h *PaymentHandler) logWebhookNotificationCtx(
	ctx context.Context,
	orderID string,
	statusCode string,
	txStatus string,
	sigKey string,
	sigValid bool,
	idempotent bool,
	note string,
	rawBytes []byte,
) {
	rawJSON := string(rawBytes)
	if strings.TrimSpace(rawJSON) == "" {
		rawJSON = "{}"
	}
	_, _ = h.pool.Exec(ctx, `
		INSERT INTO payment_notification_logs (
			order_id, status_code, transaction_status, signature_key,
			signature_valid, processed_idempotently, processed_note, raw_payload
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb)
	`, orderID, statusCode, txStatus, sigKey, sigValid, idempotent, note, rawJSON)
}

func (h *PaymentHandler) logWebhookNotification(
	c *gin.Context,
	orderID string,
	statusCode string,
	txStatus string,
	sigKey string,
	sigValid bool,
	idempotent bool,
	note string,
	rawBytes []byte,
) {
	h.logWebhookNotificationCtx(c.Request.Context(), orderID, statusCode, txStatus, sigKey, sigValid, idempotent, note, rawBytes)
}

func (h *PaymentHandler) MidtransNotification(c *gin.Context) {
	rawBytes, err := io.ReadAll(c.Request.Body)
	if err != nil || len(rawBytes) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Payload notifikasi kosong."})
		return
	}

	var payload MidtransNotificationPayload
	if err := json.Unmarshal(rawBytes, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format JSON notifikasi tidak valid."})
		return
	}

	orderID := strings.TrimSpace(payload.OrderID)
	statusCode := strings.TrimSpace(payload.StatusCode)
	grossAmountRaw := payload.GrossAmount // Pertahankan string asli persis seperti di payload (mis. "49000.00")
	sigKey := strings.TrimSpace(payload.SignatureKey)
	txStatus := strings.TrimSpace(payload.TransactionStatus)

	if orderID == "" || statusCode == "" || grossAmountRaw == "" || sigKey == "" {
		h.logWebhookNotification(c, orderID, statusCode, txStatus, sigKey, false, false, "Ditolak: field wajib tidak lengkap", rawBytes)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Field wajib notifikasi tidak lengkap."})
		return
	}

	// 1. Verifikasi Signature SHA512 menggunakan string gross_amount persis seperti di payload
	sigRaw := orderID + statusCode + grossAmountRaw + h.cfg.MidtransServerKey
	sum := sha512.Sum512([]byte(sigRaw))
	expectedSig := hex.EncodeToString(sum[:])

	if !strings.EqualFold(expectedSig, sigKey) {
		h.logWebhookNotification(c, orderID, statusCode, txStatus, sigKey, false, false, "Ditolak: signature SHA512 tidak cocok", rawBytes)
		c.JSON(http.StatusForbidden, gin.H{"error": "Signature notifikasi tidak valid."})
		return
	}

	// 2. Normalisasi gross_amount untuk dibandingkan dengan nominal di database
	parsedGross, parseErr := strconv.ParseFloat(strings.TrimSpace(grossAmountRaw), 64)
	if parseErr != nil {
		h.logWebhookNotification(c, orderID, statusCode, txStatus, sigKey, true, false, "Ditolak: format gross_amount tidak valid", rawBytes)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nilai gross_amount tidak valid."})
		return
	}

	ctx := c.Request.Context()
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi database."})
		return
	}
	defer tx.Rollback(ctx)

	var dbTxID, dbUserID, dbPlanID int64
	var dbGrossAmount, planQuotaEssays, planDurationDays int
	var dbStatus, planCode string
	var planIsAddon bool

	err = tx.QueryRow(ctx, `
		SELECT
			pt.id, pt.user_id, pt.plan_id, pt.gross_amount, pt.status,
			p.code, p.quota_essays, p.duration_days, p.is_addon
		FROM payment_transactions pt
		JOIN plans p ON p.id = pt.plan_id
		WHERE pt.order_id = $1
		FOR UPDATE OF pt
	`, orderID).Scan(
		&dbTxID, &dbUserID, &dbPlanID, &dbGrossAmount, &dbStatus,
		&planCode, &planQuotaEssays, &planDurationDays, &planIsAddon,
	)
	if err != nil {
		tx.Rollback(ctx)
		h.logWebhookNotification(c, orderID, statusCode, txStatus, sigKey, true, false, "Ditolak: order_id tidak ditemukan di database", rawBytes)
		c.JSON(http.StatusNotFound, gin.H{"error": "Transaksi tidak ditemukan."})
		return
	}

	// 3. Bandingkan nominal setelah dinormalisasi
	if math.Abs(parsedGross-float64(dbGrossAmount)) > 0.01 {
		tx.Rollback(ctx)
		note := fmt.Sprintf("Ditolak: nominal tidak cocok (payload=%.2f vs db=%d)", parsedGross, dbGrossAmount)
		h.logWebhookNotification(c, orderID, statusCode, txStatus, sigKey, true, false, note, rawBytes)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nominal gross_amount tidak cocok dengan transaksi di database."})
		return
	}

	// 4. Penanganan Idempoten: jika transaksi sudah pernah settlement/capture, abaikan tanpa menambah kuota lagi
	if dbStatus == "settlement" || dbStatus == "capture" {
		tx.Rollback(ctx)
		h.logWebhookNotification(c, orderID, statusCode, txStatus, sigKey, true, true, "Idempoten: notifikasi duplikat, transaksi sudah settlement sebelumnya", rawBytes)
		c.JSON(http.StatusOK, gin.H{
			"status":     "ok",
			"idempotent": true,
			"message":    "Notifikasi duplikat diabaikan secara idempoten.",
		})
		return
	}

	// 5. Konfirmasi ulang status dan metode pembayaran ke endpoint status Midtrans
	var rawMap map[string]any
	_ = json.Unmarshal(rawBytes, &rawMap)

	effectiveStatus := txStatus
	effectiveFraud := strings.TrimSpace(payload.FraudStatus)
	effectivePaymentType := extractPaymentMethodDetail(rawMap, payload.PaymentType)
	verifySource := "Signature SHA512"

	apiData, apiErr := h.verifyWithMidtransStatusAPI(ctx, orderID)
	if apiErr == nil && apiData != nil {
		apiStatusCode, _ := apiData["status_code"].(string)
		if apiStatusCode == "200" || apiStatusCode == "201" || apiStatusCode == "202" {
			if st, ok := apiData["transaction_status"].(string); ok && st != "" {
				effectiveStatus = st
			}
			if fr, ok := apiData["fraud_status"].(string); ok {
				effectiveFraud = fr
			}
			if pt := extractPaymentMethodDetail(apiData, effectivePaymentType); pt != "" {
				effectivePaymentType = pt
			}
			if gaStr, ok := apiData["gross_amount"].(string); ok && gaStr != "" {
				if apiGross, err := strconv.ParseFloat(gaStr, 64); err == nil {
					if math.Abs(apiGross-float64(dbGrossAmount)) > 0.01 {
						tx.Rollback(ctx)
						h.logWebhookNotification(c, orderID, statusCode, txStatus, sigKey, true, false, "Ditolak: gross_amount dari Midtrans Status API tidak cocok", rawBytes)
						c.JSON(http.StatusBadRequest, gin.H{"error": "Nominal dari Midtrans Status API tidak cocok."})
						return
					}
				}
			}
			verifySource = "Midtrans Status API + Signature SHA512"
		}
	}

	isPaid := effectiveStatus == "settlement" || (effectiveStatus == "capture" && (effectiveFraud == "" || effectiveFraud == "accept"))

	if isPaid {
		if err := h.applySettlementTx(ctx, tx, dbTxID, dbUserID, dbPlanID, planQuotaEssays, planDurationDays, planIsAddon, effectivePaymentType); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui status transaksi dan kuota langganan guru."})
			return
		}

		if err := tx.Commit(ctx); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan perubahan pembayaran."})
			return
		}

		note := fmt.Sprintf("Sukses: settlement (%s), +%d kuota esai", verifySource, planQuotaEssays)
		h.logWebhookNotification(c, orderID, statusCode, effectiveStatus, sigKey, true, false, note, rawBytes)

		c.JSON(http.StatusOK, gin.H{
			"status":     "ok",
			"idempotent": false,
			"order_id":   orderID,
			"new_status": "settlement",
		})
		return
	}

	// Jika status pending / expire / cancel / deny
	mappedStatus := effectiveStatus
	if mappedStatus == "" {
		mappedStatus = "pending"
	}
	_, _ = tx.Exec(ctx, `
		UPDATE payment_transactions
		SET status = $1,
		    payment_type = COALESCE(NULLIF($2, ''), payment_type),
		    updated_at = NOW()
		WHERE id = $3
	`, mappedStatus, effectivePaymentType, dbTxID)

	_ = tx.Commit(ctx)
	note := fmt.Sprintf("Status diperbarui ke '%s' (%s)", mappedStatus, verifySource)
	h.logWebhookNotification(c, orderID, statusCode, mappedStatus, sigKey, true, false, note, rawBytes)

	c.JSON(http.StatusOK, gin.H{
		"status":     "ok",
		"order_id":   orderID,
		"new_status": mappedStatus,
	})
}
