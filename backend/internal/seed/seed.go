package seed

import (
	"context"
	"errors"
	"log"
	"strings"

	"koreksiin-backend/internal/config"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

func Run(ctx context.Context, pool *pgxpool.Pool, cfg *config.Config) error {
	// 1. Seed Paket Langganan Default
	defaultPlans := []struct {
		code         string
		name         string
		description  string
		priceIDR     int
		quotaEssays  int
		durationDays int
		isAddon      bool
		sortOrder    int
	}{
		{
			code:         "free",
			name:         "Paket Gratis",
			description:  "Cocok untuk mencoba fitur penilaian otomatis dengan kuota esai yang direset setiap bulan.",
			priceIDR:     0,
			quotaEssays:  35,
			durationDays: 30,
			isAddon:      false,
			sortOrder:    1,
		},
		{
			code:         "monthly",
			name:         "Paket Guru Bulanan",
			description:  "Pilihan tepat untuk penilaian ulangan harian dan ujian tengah semester beserta model custom.",
			priceIDR:     49000,
			quotaEssays:  300,
			durationDays: 30,
			isAddon:      false,
			sortOrder:    2,
		},
		{
			code:         "yearly",
			name:         "Paket Sekolah Tahunan",
			description:  "Kuota besar untuk dua semester penuh dengan harga jauh lebih hemat.",
			priceIDR:     450000,
			quotaEssays:  4000,
			durationDays: 365,
			isAddon:      false,
			sortOrder:    3,
		},
		{
			code:         "addon",
			name:         "Add-on Kuota +150 Esai",
			description:  "Tambahan kuota esai instan yang berlaku mengikuti masa aktif paket Anda saat ini.",
			priceIDR:     20000,
			quotaEssays:  150,
			durationDays: 0,
			isAddon:      true,
			sortOrder:    4,
		},
	}

	for _, p := range defaultPlans {
		_, err := pool.Exec(ctx, `
			INSERT INTO plans (code, name, description, price_idr, quota_essays, duration_days, is_addon, sort_order)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (code) DO NOTHING
		`, p.code, p.name, p.description, p.priceIDR, p.quotaEssays, p.durationDays, p.isAddon, p.sortOrder)
		if err != nil {
			return err
		}
	}

	// 2. Seed AI Settings Default
	_, err := pool.Exec(ctx, `
		INSERT INTO ai_settings (id, t_high, t_low, s_sem, s_lex, min_train_samples)
		VALUES (1, $1, $2, $3, $4, $5)
		ON CONFLICT (id) DO NOTHING
	`, cfg.DefaultTHigh, cfg.DefaultTLow, cfg.DefaultSSem, cfg.DefaultSLex, cfg.MinTrainSamples)
	if err != nil {
		return err
	}

	// Sinkronkan nilai ai_settings jika masih bernilai default awal (0.72 / 0.52)
	_, _ = pool.Exec(ctx, `
		UPDATE ai_settings
		SET t_high = $1, t_low = $2, s_sem = $3, s_lex = $4, updated_at = NOW()
		WHERE id = 1 AND t_high = 0.72 AND t_low = 0.52
	`, cfg.DefaultTHigh, cfg.DefaultTLow, cfg.DefaultSSem, cfg.DefaultSLex)

	// 3. Seed & Sinkronkan Akun Admin dari .env (JANGAN membuat akun guru dummy)
	adminEmail := strings.ToLower(strings.TrimSpace(cfg.AdminEmail))
	if adminEmail != "" && cfg.AdminPassword != "" {
		var adminID int64
		var existingHash string
		err := pool.QueryRow(ctx, `SELECT id, password_hash FROM users WHERE email = $1`, adminEmail).Scan(&adminID, &existingHash)
		if errors.Is(err, pgx.ErrNoRows) {
			hash, hashErr := bcrypt.GenerateFromPassword([]byte(cfg.AdminPassword), bcrypt.DefaultCost)
			if hashErr != nil {
				return hashErr
			}
			_, execErr := pool.Exec(ctx, `
				INSERT INTO users (name, email, institution, password_hash, role, is_active)
				VALUES ($1, $2, 'Koreksi.in System', $3, 'admin', TRUE)
			`, cfg.AdminName, adminEmail, string(hash))
			if execErr != nil {
				return execErr
			}
			log.Printf("[Seed] Akun Admin berhasil dibuat untuk email: %s", adminEmail)
		} else if err != nil {
			return err
		} else if bcrypt.CompareHashAndPassword([]byte(existingHash), []byte(cfg.AdminPassword)) != nil {
			hash, hashErr := bcrypt.GenerateFromPassword([]byte(cfg.AdminPassword), bcrypt.DefaultCost)
			if hashErr != nil {
				return hashErr
			}
			_, execErr := pool.Exec(ctx, `UPDATE users SET password_hash = $1 WHERE id = $2`, string(hash), adminID)
			if execErr != nil {
				return execErr
			}
			log.Printf("[Seed] Password admin diperbarui mengikuti konfigurasi environment untuk email: %s", adminEmail)
		}
	}

	return nil
}

