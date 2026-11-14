package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"koreksiin-backend/internal/config"
	"koreksiin-backend/internal/db"
	"koreksiin-backend/internal/handlers"
	"koreksiin-backend/internal/middleware"
	"koreksiin-backend/internal/seed"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func warnDefaultSecrets(cfg *config.Config) {
	if cfg.AdminPassword == "" ||
		cfg.AdminPassword == "AdminKoreksiin2026!" ||
		cfg.AdminPassword == "GANTI_DENGAN_PASSWORD_KUAT" {
		log.Printf("[Security] PERINGATAN: ADMIN_PASSWORD masih memakai nilai default/placeholder. Segera ganti dengan password kuat di file .env.")
	}
	if cfg.JWTSecret == "" ||
		cfg.JWTSecret == "koreksiin_super_secret_jwt_key_2026_change_me" ||
		cfg.JWTSecret == "koreksiin_super_secret_jwt_key_2026_local_dev" ||
		cfg.JWTSecret == "GANTI_DENGAN_STRING_ACAK_PANJANG" {
		log.Printf("[Security] PERINGATAN: JWT_SECRET masih memakai nilai default/placeholder. Segera ganti dengan string acak minimal 32 karakter di file .env.")
	}
}

func main() {
	cfg := config.Load()
	warnDefaultSecrets(cfg)

	if err := os.MkdirAll(cfg.StorageDir, 0755); err != nil {
		log.Printf("[Storage] Peringatan gagal membuat direktori storage: %v", err)
	}

	ctx := context.Background()
	pool, err := db.ConnectAndMigrate(ctx, cfg.DatabaseURL, "./migrations")
	if err != nil {
		log.Fatalf("[Fatal] Database error: %v", err)
	}
	defer pool.Close()

	if err := seed.Run(ctx, pool, cfg); err != nil {
		log.Fatalf("[Fatal] Seed error: %v", err)
	}

	opt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		log.Fatalf("[Fatal] Redis URL tidak valid: %v", err)
	}
	rdb := redis.NewClient(opt)
	defer rdb.Close()

	for attempt := 1; attempt <= 10; attempt++ {
		if err := rdb.Ping(ctx).Err(); err == nil {
			break
		}
		log.Printf("[Redis] Menunggu Redis siap (percobaan %d/10)...", attempt)
		time.Sleep(1 * time.Second)
	}

	r := gin.Default()

	authHandler := handlers.NewAuthHandler(pool, cfg)
	assignmentHandler := handlers.NewAssignmentHandler(pool, rdb, cfg)
	modelHandler := handlers.NewModelHandler(pool, rdb, cfg)
	paymentHandler := handlers.NewPaymentHandler(pool, cfg)
	adminHandler := handlers.NewAdminHandler(pool, rdb, cfg, paymentHandler)

	api := r.Group("/api")
	{
		api.GET("/health", func(c *gin.Context) {
			dbErr := pool.Ping(c.Request.Context())
			redisErr := rdb.Ping(c.Request.Context()).Err()
			status := http.StatusOK
			if dbErr != nil || redisErr != nil {
				status = http.StatusServiceUnavailable
			}
			c.JSON(status, gin.H{
				"status":   "ok",
				"service":  "koreksiin-backend",
				"postgres": dbErr == nil,
				"redis":    redisErr == nil,
			})
		})

		api.GET("/plans", authHandler.PublicPlans)
		api.GET("/templates/excel", assignmentHandler.DownloadTemplate)
		api.GET("/payments/config", paymentHandler.GetConfig)
		api.POST("/payments/midtrans/notification", paymentHandler.MidtransNotification)

		authGroup := api.Group("/auth")
		{
			authGroup.POST("/register", authHandler.Register)
			authGroup.POST("/login", authHandler.Login)
			authGroup.GET("/me", middleware.AuthRequired(cfg.JWTSecret), authHandler.Me)
		}

		guruGroup := api.Group("")
		guruGroup.Use(middleware.AuthRequired(cfg.JWTSecret), middleware.RequireRole("guru"))
		{
			guruGroup.GET("/guru/stats", assignmentHandler.GuruStats)
			guruGroup.GET("/assignments", assignmentHandler.List)
			guruGroup.POST("/assignments", assignmentHandler.Create)
			guruGroup.GET("/assignments/:id", assignmentHandler.GetDetail)
			guruGroup.PUT("/assignments/:id", assignmentHandler.Update)
			guruGroup.DELETE("/assignments/:id", assignmentHandler.Delete)

			guruGroup.POST("/assignments/:id/submissions/manual", assignmentHandler.AddManualSubmissions)
			guruGroup.POST("/assignments/:id/submissions/excel", assignmentHandler.UploadExcelSubmissions)
			guruGroup.POST("/assignments/:id/score", assignmentHandler.EnqueueScoringJob)
			guruGroup.GET("/assignments/:id/export", assignmentHandler.ExportResultsExcel)
			guruGroup.GET("/assignments/:id/training-samples", assignmentHandler.ListTrainingSamples)
			guruGroup.POST("/assignments/:id/training-samples/excel", modelHandler.UploadTrainingExcel)

			guruGroup.DELETE("/training-samples/:sampleId", modelHandler.DeleteTrainingSample)

			guruGroup.GET("/models", modelHandler.List)
			guruGroup.POST("/models", modelHandler.Create)
			guruGroup.GET("/models/:id", modelHandler.GetDetail)
			guruGroup.PUT("/models/:id", modelHandler.Update)
			guruGroup.DELETE("/models/:id", modelHandler.Delete)
			guruGroup.POST("/models/:id/train", modelHandler.StartTraining)
			guruGroup.POST("/models/:id/versions/:versionId/activate", modelHandler.ActivateVersion)
			guruGroup.POST("/models/:id/deactivate", modelHandler.DeactivateModel)

			guruGroup.DELETE("/submissions/:subId", assignmentHandler.DeleteSubmission)
			guruGroup.PUT("/submissions/:subId/revise-score", assignmentHandler.ReviseSubmissionScore)

			guruGroup.GET("/jobs/:id", assignmentHandler.GetJobStatus)

			guruGroup.POST("/payments/checkout", paymentHandler.Checkout)
			guruGroup.GET("/payments/transactions", paymentHandler.ListGuruTransactions)
			guruGroup.POST("/payments/transactions/:id/cancel", paymentHandler.CancelPendingTransaction)
			guruGroup.POST("/payments/reconcile", paymentHandler.ReconcileGuruTransactions)
		}

		adminGroup := api.Group("/admin")
		adminGroup.Use(middleware.AuthRequired(cfg.JWTSecret), middleware.RequireRole("admin"))
		{
			adminGroup.GET("/overview", adminHandler.Overview)
			adminGroup.GET("/teachers", adminHandler.ListTeachers)
			adminGroup.PUT("/teachers/:id/status", adminHandler.UpdateTeacherStatus)
			adminGroup.GET("/transactions", adminHandler.ListTransactions)
			adminGroup.GET("/plans", adminHandler.ListPlans)
			adminGroup.PUT("/plans/:id", adminHandler.UpdatePlan)
			adminGroup.GET("/ai-settings", adminHandler.GetAISettings)
			adminGroup.PUT("/ai-settings", adminHandler.UpdateAISettings)
		}
	}

	addr := ":" + cfg.BackendPort
	log.Printf("[Server] Koreksi.in Backend berjalan di %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("[Fatal] Gagal menjalankan server HTTP: %v", err)
	}
}
