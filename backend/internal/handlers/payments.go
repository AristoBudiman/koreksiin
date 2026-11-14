package handlers

import (
	"bytes"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"koreksiin-backend/internal/config"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PaymentHandler struct {
	pool *pgxpool.Pool
	cfg  *config.Config
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

	var planID int64
	var planCode, planName string
	var priceIDR, quotaEssays, durationDays int
	var isAddon bool
	err := h.pool.QueryRow(ctx, `
		SELECT id, code, name, price_idr, quota_essays, duration_days, is_addon
		FROM plans
		WHERE code = $1 AND is_active = TRUE
	`, strings.TrimSpace(req.PlanCode)).Scan(
		&planID, &planCode, &planName, &priceIDR, &quotaEssays, &durationDays, &isAddon,
	)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Paket langganan tidak ditemukan atau sedang tidak aktif."})
		return
	}

	if priceIDR <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Paket Gratis tidak memerlukan transaksi pembayaran."})
		return
	}

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
		"enabled_payments": []string{"other_qris", "gopay"},
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
		"message":           "Transaksi berhasil dibuat. Silakan selesaikan pembayaran melalui Midtrans Snap.",
		"transaction_id":    txID,
		"order_id":          orderID,
		"plan_code":         planCode,
		"plan_name":         planName,
		"gross_amount":      priceIDR,
		"snap_token":        snapResp.Token,
		"snap_redirect_url": snapResp.RedirectURL,
	})
}

func (h *PaymentHandler) verifyWithMidtransStatusAPI(ctx *gin.Context, orderID string) (map[string]any, error) {
	baseAPI := "https://api.sandbox.midtrans.com"
	if h.cfg.MidtransIsProduction {
		baseAPI = "https://api.midtrans.com"
	}
	url := fmt.Sprintf("%s/v2/%s/status", baseAPI, orderID)

	req, err := http.NewRequestWithContext(ctx.Request.Context(), http.MethodGet, url, nil)
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
	rawJSON := string(rawBytes)
	if strings.TrimSpace(rawJSON) == "" {
		rawJSON = "{}"
	}
	_, _ = h.pool.Exec(c.Request.Context(), `
		INSERT INTO payment_notification_logs (
			order_id, status_code, transaction_status, signature_key,
			signature_valid, processed_idempotently, processed_note, raw_payload
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb)
	`, orderID, statusCode, txStatus, sigKey, sigValid, idempotent, note, rawJSON)
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

	// 5. Konfirmasi ulang status ke endpoint status Midtrans
	effectiveStatus := txStatus
	effectiveFraud := strings.TrimSpace(payload.FraudStatus)
	effectivePaymentType := strings.TrimSpace(payload.PaymentType)
	verifySource := "Signature SHA512"

	apiData, apiErr := h.verifyWithMidtransStatusAPI(c, orderID)
	if apiErr == nil && apiData != nil {
		apiStatusCode, _ := apiData["status_code"].(string)
		if apiStatusCode == "200" || apiStatusCode == "201" || apiStatusCode == "202" {
			if st, ok := apiData["transaction_status"].(string); ok && st != "" {
				effectiveStatus = st
			}
			if fr, ok := apiData["fraud_status"].(string); ok {
				effectiveFraud = fr
			}
			if pt, ok := apiData["payment_type"].(string); ok && pt != "" {
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
		_, err = tx.Exec(ctx, `
			UPDATE payment_transactions
			SET status = 'settlement',
			    payment_type = $1,
			    paid_at = NOW(),
			    updated_at = NOW()
			WHERE id = $2
		`, effectivePaymentType, dbTxID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui status transaksi."})
			return
		}

		// Perbarui langganan / kuota guru HANYA saat pembayaran settlement/capture
		if planIsAddon {
			// Add-on: tambahkan ke quota_total dan masa berlaku mengikuti paket aktif saat ini
			_, err = tx.Exec(ctx, `
				UPDATE subscriptions
				SET quota_total = quota_total + $1,
				    updated_at = NOW()
				WHERE user_id = $2
			`, planQuotaEssays, dbUserID)
		} else {
			// Paket Bulanan / Tahunan: cek apakah sebelumnya paket gratis atau paket berbayar aktif
			var curPlanCode string
			var curRemaining int
			var curExpiresAt *time.Time
			_ = tx.QueryRow(ctx, `
				SELECT p.code, GREATEST(s.quota_total - s.quota_used, 0), s.expires_at
				FROM subscriptions s
				JOIN plans p ON p.id = s.plan_id
				WHERE s.user_id = $1
			`, dbUserID).Scan(&curPlanCode, &curRemaining, &curExpiresAt)

			newQuotaTotal := planQuotaEssays
			if curPlanCode != "" && curPlanCode != "free" && curRemaining > 0 {
				// Bila masih ada sisa kuota berbayar/add-on sebelumnya, akumulasikan ke paket baru
				newQuotaTotal = curRemaining + planQuotaEssays
			}

			baseTime := time.Now()
			if curExpiresAt != nil && curExpiresAt.After(baseTime) {
				baseTime = *curExpiresAt
			}
			newExpiresAt := baseTime.AddDate(0, 0, planDurationDays)

			_, err = tx.Exec(ctx, `
				UPDATE subscriptions
				SET plan_id = $1,
				    quota_total = $2,
				    quota_used = 0,
				    quota_reset_at = NULL,
				    expires_at = $3,
				    updated_at = NOW()
				WHERE user_id = $4
			`, dbPlanID, newQuotaTotal, newExpiresAt, dbUserID)
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui kuota langganan guru."})
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

