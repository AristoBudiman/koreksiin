package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"koreksiin-backend/internal/config"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/xuri/excelize/v2"
)

type ModelHandler struct {
	pool *pgxpool.Pool
	rdb  *redis.Client
	cfg  *config.Config
}

func NewModelHandler(pool *pgxpool.Pool, rdb *redis.Client, cfg *config.Config) *ModelHandler {
	return &ModelHandler{pool: pool, rdb: rdb, cfg: cfg}
}

type ModelInput struct {
	Name          string  `json:"name" binding:"required"`
	Description   string  `json:"description"`
	AssignmentIDs []int64 `json:"assignment_ids"`
}

type LinkedAssignmentInfo struct {
	ID          int64   `json:"id"`
	Title       string  `json:"title"`
	Subject     string  `json:"subject"`
	MaxScore    float64 `json:"max_score"`
	SampleCount int     `json:"sample_count"`
}

type ModelVersionInfo struct {
	ID            int64          `json:"id"`
	ModelID       int64          `json:"model_id"`
	VersionNum    int            `json:"version_num"`
	TrainingMode  string         `json:"training_mode"`
	SampleCount   int            `json:"sample_count"`
	TrainCount    int            `json:"train_count"`
	TestCount     int            `json:"test_count"`
	MAECustom     float64        `json:"mae_custom"`
	PearsonCustom float64        `json:"pearson_custom"`
	MAEBase       float64        `json:"mae_base"`
	PearsonBase   float64        `json:"pearson_base"`
	Metadata      map[string]any `json:"metadata"`
	IsActive      bool           `json:"is_active"`
	IsRecommended bool           `json:"is_recommended"`
	CreatedAt     time.Time      `json:"created_at"`
}

type CustomModelItem struct {
	ID                int64                  `json:"id"`
	Name              string                 `json:"name"`
	Description       string                 `json:"description"`
	ActiveVersionID   *int64                 `json:"active_version_id"`
	LinkedAssignments []LinkedAssignmentInfo `json:"linked_assignments"`
	TotalSamples      int                    `json:"total_samples"`
	VersionCount      int                    `json:"version_count"`
	ActiveVersion     *ModelVersionInfo      `json:"active_version"`
	LatestVersion     *ModelVersionInfo      `json:"latest_version"`
	Versions          []ModelVersionInfo     `json:"versions,omitempty"`
	LatestJob         map[string]any         `json:"latest_job"`
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
}

func (h *ModelHandler) getMinTrainSamples(c *gin.Context) int {
	minSamples := h.cfg.MinTrainSamples
	_ = h.pool.QueryRow(c.Request.Context(),
		`SELECT min_train_samples FROM ai_settings WHERE id = 1`,
	).Scan(&minSamples)
	if minSamples <= 0 {
		minSamples = 20
	}
	return minSamples
}

func (h *ModelHandler) populateModelExtra(c *gin.Context, userID int64, item *CustomModelItem, includeAllVersions bool) {
	ctx := c.Request.Context()

	// 1. Tugas sumber data latih yang terhubung
	aRows, err := h.pool.Query(ctx, `
		SELECT
			a.id, a.title, a.subject, a.max_score,
			(SELECT COUNT(*) FROM training_samples ts WHERE ts.assignment_id = a.id) AS sample_count
		FROM custom_model_assignments cma
		JOIN assignments a ON a.id = cma.assignment_id
		WHERE cma.model_id = $1 AND a.user_id = $2
		ORDER BY a.created_at DESC
	`, item.ID, userID)
	item.LinkedAssignments = make([]LinkedAssignmentInfo, 0)
	totalSamples := 0
	if err == nil {
		for aRows.Next() {
			var la LinkedAssignmentInfo
			if err := aRows.Scan(&la.ID, &la.Title, &la.Subject, &la.MaxScore, &la.SampleCount); err == nil {
				item.LinkedAssignments = append(item.LinkedAssignments, la)
				totalSamples += la.SampleCount
			}
		}
		aRows.Close()
	}
	item.TotalSamples = totalSamples

	// 2. Daftar versi model custom
	vRows, err := h.pool.Query(ctx, `
		SELECT
			id, model_id, version_num, COALESCE(training_mode, 'calibration'),
			sample_count, train_count, test_count,
			mae_custom, pearson_custom, mae_base, pearson_base,
			COALESCE(metadata, '{}'::jsonb), is_active, created_at
		FROM custom_model_versions
		WHERE model_id = $1
		ORDER BY version_num DESC
	`, item.ID)
	versions := make([]ModelVersionInfo, 0)
	if err == nil {
		for vRows.Next() {
			var v ModelVersionInfo
			var metaRaw []byte
			if err := vRows.Scan(
				&v.ID, &v.ModelID, &v.VersionNum, &v.TrainingMode,
				&v.SampleCount, &v.TrainCount, &v.TestCount,
				&v.MAECustom, &v.PearsonCustom, &v.MAEBase, &v.PearsonBase,
				&metaRaw, &v.IsActive, &v.CreatedAt,
			); err == nil {
				_ = json.Unmarshal(metaRaw, &v.Metadata)
				versions = append(versions, v)
			}
		}
		vRows.Close()
	}

	// Tandai versi dengan MAE CV terkecil sebagai IsRecommended
	if len(versions) > 0 {
		bestIdx := 0
		for i := 1; i < len(versions); i++ {
			if versions[i].MAECustom < versions[bestIdx].MAECustom {
				bestIdx = i
			}
		}
		versions[bestIdx].IsRecommended = true
	}

	item.VersionCount = len(versions)
	if len(versions) > 0 {
		latestCopy := versions[0]
		item.LatestVersion = &latestCopy
		for i := range versions {
			if versions[i].IsActive || (item.ActiveVersionID != nil && versions[i].ID == *item.ActiveVersionID) {
				activeCopy := versions[i]
				item.ActiveVersion = &activeCopy
				break
			}
		}
	}
	if includeAllVersions {
		item.Versions = versions
	}

	// 3. Job training terakhir untuk model ini
	var jobID, jobStatus, jobStage, jobErr string
	var jobProgress int
	var jobMetaRaw []byte
	var jobUpdated time.Time
	err = h.pool.QueryRow(ctx, `
		SELECT id::text, status, stage, progress, error_message, metadata, updated_at
		FROM jobs
		WHERE type = 'training' AND reference_id = $1 AND user_id = $2
		ORDER BY created_at DESC
		LIMIT 1
	`, item.ID, userID).Scan(&jobID, &jobStatus, &jobStage, &jobProgress, &jobErr, &jobMetaRaw, &jobUpdated)
	if err == nil {
		var meta map[string]any
		_ = json.Unmarshal(jobMetaRaw, &meta)
		item.LatestJob = map[string]any{
			"id":            jobID,
			"status":        jobStatus,
			"stage":         jobStage,
			"progress":      jobProgress,
			"error_message": jobErr,
			"metadata":      meta,
			"updated_at":    jobUpdated,
		}
	}
}

func (h *ModelHandler) List(c *gin.Context) {
	userID := c.GetInt64("user_id")
	ctx := c.Request.Context()

	rows, err := h.pool.Query(ctx, `
		SELECT id, name, description, active_version_id, created_at, updated_at
		FROM custom_models
		WHERE user_id = $1
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memuat daftar model custom."})
		return
	}
	defer rows.Close()

	models := make([]CustomModelItem, 0)
	for rows.Next() {
		var item CustomModelItem
		if err := rows.Scan(
			&item.ID, &item.Name, &item.Description, &item.ActiveVersionID,
			&item.CreatedAt, &item.UpdatedAt,
		); err == nil {
			models = append(models, item)
		}
	}
	rows.Close()

	for i := range models {
		h.populateModelExtra(c, userID, &models[i], true)
	}

	c.JSON(http.StatusOK, gin.H{
		"models":            models,
		"min_train_samples": h.getMinTrainSamples(c),
		"base_model_name":   "LazarusNLP/all-indobert-base-v4",
	})
}

func (h *ModelHandler) GetDetail(c *gin.Context) {
	userID := c.GetInt64("user_id")
	modelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID model tidak valid."})
		return
	}

	ctx := c.Request.Context()
	var item CustomModelItem
	err = h.pool.QueryRow(ctx, `
		SELECT id, name, description, active_version_id, created_at, updated_at
		FROM custom_models
		WHERE id = $1 AND user_id = $2
	`, modelID, userID).Scan(
		&item.ID, &item.Name, &item.Description, &item.ActiveVersionID,
		&item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Model custom tidak ditemukan."})
		return
	}

	h.populateModelExtra(c, userID, &item, true)
	c.JSON(http.StatusOK, gin.H{
		"model":             item,
		"min_train_samples": h.getMinTrainSamples(c),
	})
}

func (h *ModelHandler) Create(c *gin.Context) {
	userID := c.GetInt64("user_id")
	var req ModelInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nama model custom wajib diisi."})
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nama model custom tidak boleh kosong."})
		return
	}

	ctx := c.Request.Context()
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi database."})
		return
	}
	defer tx.Rollback(ctx)

	var modelID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO custom_models (user_id, name, description)
		VALUES ($1, $2, $3)
		RETURNING id
	`, userID, name, strings.TrimSpace(req.Description)).Scan(&modelID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan model custom."})
		return
	}

	for _, aid := range req.AssignmentIDs {
		var exists bool
		_ = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM assignments WHERE id = $1 AND user_id = $2)`, aid, userID).Scan(&exists)
		if exists {
			_, _ = tx.Exec(ctx, `
				INSERT INTO custom_model_assignments (model_id, assignment_id)
				VALUES ($1, $2)
				ON CONFLICT DO NOTHING
			`, modelID, aid)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan relasi tugas model custom."})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":  "Model custom berhasil dibuat.",
		"model_id": modelID,
	})
}

func (h *ModelHandler) Update(c *gin.Context) {
	userID := c.GetInt64("user_id")
	modelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID model tidak valid."})
		return
	}

	var req ModelInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data pembaruan model tidak valid."})
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nama model custom tidak boleh kosong."})
		return
	}

	ctx := c.Request.Context()
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi."})
		return
	}
	defer tx.Rollback(ctx)

	cmdTag, err := tx.Exec(ctx, `
		UPDATE custom_models
		SET name = $1, description = $2, updated_at = NOW()
		WHERE id = $3 AND user_id = $4
	`, name, strings.TrimSpace(req.Description), modelID, userID)
	if err != nil || cmdTag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Model custom tidak ditemukan."})
		return
	}

	_, _ = tx.Exec(ctx, `DELETE FROM custom_model_assignments WHERE model_id = $1`, modelID)
	for _, aid := range req.AssignmentIDs {
		var exists bool
		_ = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM assignments WHERE id = $1 AND user_id = $2)`, aid, userID).Scan(&exists)
		if exists {
			_, _ = tx.Exec(ctx, `
				INSERT INTO custom_model_assignments (model_id, assignment_id)
				VALUES ($1, $2)
				ON CONFLICT DO NOTHING
			`, modelID, aid)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui model custom."})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Model custom berhasil diperbarui."})
}

func (h *ModelHandler) Delete(c *gin.Context) {
	userID := c.GetInt64("user_id")
	modelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID model tidak valid."})
		return
	}

	ctx := c.Request.Context()

	// Pastikan jika ada tugas yang memakai model ini, kembalikan ke model_type = 'base'
	_, _ = h.pool.Exec(ctx, `
		UPDATE assignments
		SET model_type = 'base', custom_model_id = NULL, updated_at = NOW()
		WHERE custom_model_id = $1 AND user_id = $2
	`, modelID, userID)

	cmdTag, err := h.pool.Exec(ctx, `
		DELETE FROM custom_models WHERE id = $1 AND user_id = $2
	`, modelID, userID)
	if err != nil || cmdTag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Model custom tidak ditemukan."})
		return
	}

	modelDir := filepath.Join(h.cfg.StorageDir, "models", fmt.Sprintf("user_%d", userID), fmt.Sprintf("model_%d", modelID))
	_ = os.RemoveAll(modelDir)

	c.JSON(http.StatusOK, gin.H{"message": "Model custom beserta seluruh versinya berhasil dihapus."})
}

func (h *ModelHandler) UploadTrainingExcel(c *gin.Context) {
	userID := c.GetInt64("user_id")
	assignmentID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tugas tidak valid."})
		return
	}

	dryRun := c.Query("dry_run") == "true"
	ctx := c.Request.Context()

	var maxScore float64
	var assignmentTitle string
	err = h.pool.QueryRow(ctx, `
		SELECT title, max_score FROM assignments WHERE id = $1 AND user_id = $2
	`, assignmentID, userID).Scan(&assignmentTitle, &maxScore)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tugas tidak ditemukan."})
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File Excel (.xlsx) wajib diunggah."})
		return
	}
	if !strings.HasSuffix(strings.ToLower(fileHeader.Filename), ".xlsx") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format file harus .xlsx (Microsoft Excel)."})
		return
	}

	f, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Gagal membuka file yang diunggah."})
		return
	}
	defer f.Close()

	buf, err := io.ReadAll(f)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Gagal membaca isi file."})
		return
	}

	xl, err := excelize.OpenReader(bytes.NewReader(buf))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File Excel rusak atau tidak dapat dibaca."})
		return
	}
	defer xl.Close()

	sheets := xl.GetSheetList()
	if len(sheets) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File Excel tidak memiliki sheet."})
		return
	}

	rows, err := xl.GetRows(sheets[0])
	if err != nil || len(rows) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Sheet pertama pada file Excel kosong."})
		return
	}

	header := rows[0]
	colName := -1
	colAnswer := -1
	colScore := -1
	for idx, col := range header {
		norm := strings.ToLower(strings.TrimSpace(col))
		if norm == "nama_siswa" {
			colName = idx
		} else if norm == "jawaban" {
			colAnswer = idx
		} else if norm == "nilai" {
			colScore = idx
		}
	}

	if colName == -1 || colAnswer == -1 || colScore == -1 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Header kolom wajib memuat 'nama_siswa', 'jawaban', dan 'nilai' pada baris pertama.",
		})
		return
	}

	type ValidTrainRow struct {
		RowNumber   int     `json:"row_number"`
		StudentName string  `json:"student_name"`
		AnswerText  string  `json:"answer_text"`
		TargetScore float64 `json:"target_score"`
	}
	type InvalidTrainRow struct {
		RowNumber   int    `json:"row_number"`
		StudentName string `json:"student_name"`
		Reason      string `json:"reason"`
	}

	validRows := make([]ValidTrainRow, 0)
	errorRows := make([]InvalidTrainRow, 0)

	for rIdx := 1; rIdx < len(rows); rIdx++ {
		r := rows[rIdx]
		rowNum := rIdx + 1

		getCol := func(cIdx int) string {
			if cIdx >= 0 && cIdx < len(r) {
				return strings.TrimSpace(r[cIdx])
			}
			return ""
		}

		sName := getCol(colName)
		sAns := getCol(colAnswer)
		sScoreRaw := getCol(colScore)

		if sName == "" && sAns == "" && sScoreRaw == "" {
			continue
		}

		if sName == "" {
			errorRows = append(errorRows, InvalidTrainRow{
				RowNumber:   rowNum,
				StudentName: "(Kosong)",
				Reason:      "Kolom nama_siswa kosong",
			})
			continue
		}
		if sAns == "" {
			errorRows = append(errorRows, InvalidTrainRow{
				RowNumber:   rowNum,
				StudentName: sName,
				Reason:      "Kolom jawaban kosong",
			})
			continue
		}
		if sScoreRaw == "" {
			errorRows = append(errorRows, InvalidTrainRow{
				RowNumber:   rowNum,
				StudentName: sName,
				Reason:      "Kolom nilai kosong",
			})
			continue
		}

		scoreVal, parseErr := strconv.ParseFloat(strings.ReplaceAll(sScoreRaw, ",", "."), 64)
		if parseErr != nil {
			errorRows = append(errorRows, InvalidTrainRow{
				RowNumber:   rowNum,
				StudentName: sName,
				Reason:      fmt.Sprintf("Nilai '%s' bukan angka yang valid", sScoreRaw),
			})
			continue
		}
		if scoreVal < 0 || scoreVal > maxScore {
			errorRows = append(errorRows, InvalidTrainRow{
				RowNumber:   rowNum,
				StudentName: sName,
				Reason:      fmt.Sprintf("Nilai %.2f di luar rentang tugas (0 - %.0f)", scoreVal, maxScore),
			})
			continue
		}

		validRows = append(validRows, ValidTrainRow{
			RowNumber:   rowNum,
			StudentName: sName,
			AnswerText:  sAns,
			TargetScore: scoreVal,
		})
	}

	if dryRun {
		c.JSON(http.StatusOK, gin.H{
			"dry_run":          true,
			"assignment_id":    assignmentID,
			"assignment_title": assignmentTitle,
			"max_score":        maxScore,
			"valid_count":      len(validRows),
			"error_count":      len(errorRows),
			"valid_rows":       validRows,
			"error_rows":       errorRows,
		})
		return
	}

	if len(validRows) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":       "Tidak ada baris data latih yang valid untuk disimpan.",
			"error_count": len(errorRows),
			"error_rows":  errorRows,
		})
		return
	}

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi database."})
		return
	}
	defer tx.Rollback(ctx)

	inserted := 0
	for _, vr := range validRows {
		_, err := tx.Exec(ctx, `
			INSERT INTO training_samples (assignment_id, student_name, answer_text, target_score, source)
			VALUES ($1, $2, $3, $4, 'excel')
		`, assignmentID, vr.StudentName, vr.AnswerText, vr.TargetScore)
		if err == nil {
			inserted++
		}
	}

	if err := tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan sampel data latih."})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":        fmt.Sprintf("%d sampel data latih berhasil ditambahkan ke tugas '%s'.", inserted, assignmentTitle),
		"inserted_count": inserted,
		"error_count":    len(errorRows),
		"error_rows":     errorRows,
	})
}

func (h *ModelHandler) DeleteTrainingSample(c *gin.Context) {
	userID := c.GetInt64("user_id")
	sampleID, err := strconv.ParseInt(c.Param("sampleId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID sampel tidak valid."})
		return
	}

	ctx := c.Request.Context()
	cmdTag, err := h.pool.Exec(ctx, `
		DELETE FROM training_samples ts
		USING assignments a
		WHERE ts.id = $1 AND ts.assignment_id = a.id AND a.user_id = $2
	`, sampleID, userID)
	if err != nil || cmdTag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Sampel data latih tidak ditemukan."})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Sampel data latih berhasil dihapus."})
}

func (h *ModelHandler) StartTraining(c *gin.Context) {
	userID := c.GetInt64("user_id")
	modelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID model tidak valid."})
		return
	}

	var req struct {
		UseLoRA bool `json:"use_lora"`
	}
	_ = c.ShouldBindJSON(&req)

	ctx := c.Request.Context()
	var modelName string
	err = h.pool.QueryRow(ctx, `
		SELECT name FROM custom_models WHERE id = $1 AND user_id = $2
	`, modelID, userID).Scan(&modelName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Model custom tidak ditemukan."})
		return
	}

	// Cek apakah sedang ada job training yang berjalan untuk model ini
	var activeJobID string
	err = h.pool.QueryRow(ctx, `
		SELECT id::text FROM jobs
		WHERE type = 'training' AND reference_id = $1 AND status IN ('queued', 'processing')
		ORDER BY created_at DESC LIMIT 1
	`, modelID).Scan(&activeJobID)
	if err == nil && activeJobID != "" {
		c.JSON(http.StatusConflict, gin.H{
			"error":  "Pelatihan model sedang berjalan.",
			"job_id": activeJobID,
		})
		return
	}

	// Hitung jumlah sampel di semua tugas yang terhubung
	var sampleCount int
	_ = h.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM training_samples ts
		JOIN custom_model_assignments cma ON cma.assignment_id = ts.assignment_id
		WHERE cma.model_id = $1
	`, modelID).Scan(&sampleCount)

	minSamples := h.getMinTrainSamples(c)
	if sampleCount < minSamples {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":        fmt.Sprintf("Jumlah data latih belum mencukupi (%d dari minimal %d sampel). Tambahkan data latih dari revisi nilai atau unggah Excel terlebih dahulu.", sampleCount, minSamples),
			"sample_count": sampleCount,
			"min_samples":  minSamples,
		})
		return
	}

	var jobID string
	err = h.pool.QueryRow(ctx, `
		INSERT INTO jobs (user_id, type, queue_name, reference_id, status, stage, progress)
		VALUES ($1, 'training', $2, $3, 'queued', 'antri', 0)
		RETURNING id::text
	`, userID, h.cfg.TrainingQueue, modelID).Scan(&jobID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat antrian pelatihan model."})
		return
	}

	payload, _ := json.Marshal(map[string]any{
		"job_id":   jobID,
		"type":     "training",
		"model_id": modelID,
		"use_lora": req.UseLoRA,
	})
	if err := h.rdb.RPush(ctx, h.cfg.TrainingQueue, string(payload)).Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengirim job ke antrian Redis."})
		return
	}

	modeText := "Mode Kalibrasi (5-Fold CV)"
	if req.UseLoRA {
		modeText = "Mode Kalibrasi + Fine-tune Mendalam LoRA (5-Fold CV)"
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message":      fmt.Sprintf("Pelatihan model '%s' (%s) dimulai di latar belakang.", modelName, modeText),
		"job_id":       jobID,
		"sample_count": sampleCount,
		"use_lora":     req.UseLoRA,
	})
}

func (h *ModelHandler) ActivateVersion(c *gin.Context) {
	userID := c.GetInt64("user_id")
	modelID, err1 := strconv.ParseInt(c.Param("id"), 10, 64)
	versionID, err2 := strconv.ParseInt(c.Param("versionId"), 10, 64)
	if err1 != nil || err2 != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID model atau versi tidak valid."})
		return
	}

	ctx := c.Request.Context()
	var vNum int
	err := h.pool.QueryRow(ctx, `
		SELECT v.version_num
		FROM custom_model_versions v
		JOIN custom_models m ON m.id = v.model_id
		WHERE v.id = $1 AND m.id = $2 AND m.user_id = $3
	`, versionID, modelID, userID).Scan(&vNum)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Versi model custom tidak ditemukan."})
		return
	}

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi."})
		return
	}
	defer tx.Rollback(ctx)

	_, _ = tx.Exec(ctx, `UPDATE custom_model_versions SET is_active = FALSE WHERE model_id = $1`, modelID)
	_, _ = tx.Exec(ctx, `UPDATE custom_model_versions SET is_active = TRUE WHERE id = $1 AND model_id = $2`, versionID, modelID)
	_, _ = tx.Exec(ctx, `UPDATE custom_models SET active_version_id = $1, updated_at = NOW() WHERE id = $2`, versionID, modelID)

	if err := tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengaktifkan versi model custom."})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":           fmt.Sprintf("Versi v%d berhasil diaktifkan.", vNum),
		"active_version_id": versionID,
	})
}

func (h *ModelHandler) DeactivateModel(c *gin.Context) {
	userID := c.GetInt64("user_id")
	modelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID model tidak valid."})
		return
	}

	ctx := c.Request.Context()
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi."})
		return
	}
	defer tx.Rollback(ctx)

	cmdTag, err := tx.Exec(ctx, `
		UPDATE custom_models SET active_version_id = NULL, updated_at = NOW()
		WHERE id = $1 AND user_id = $2
	`, modelID, userID)
	if err != nil || cmdTag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Model custom tidak ditemukan."})
		return
	}

	_, _ = tx.Exec(ctx, `UPDATE custom_model_versions SET is_active = FALSE WHERE model_id = $1`, modelID)

	if err := tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menonaktifkan model custom."})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Model custom dinonaktifkan."})
}

