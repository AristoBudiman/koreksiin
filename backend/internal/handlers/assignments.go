package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"koreksiin-backend/internal/config"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/xuri/excelize/v2"
)

type AssignmentHandler struct {
	pool *pgxpool.Pool
	rdb  *redis.Client
	cfg  *config.Config
}

func NewAssignmentHandler(pool *pgxpool.Pool, rdb *redis.Client, cfg *config.Config) *AssignmentHandler {
	return &AssignmentHandler{pool: pool, rdb: rdb, cfg: cfg}
}

type CriterionInput struct {
	Name      string  `json:"name" binding:"required"`
	Weight    float64 `json:"weight" binding:"required"`
	AnswerKey string  `json:"answer_key" binding:"required"`
}

type AssignmentInput struct {
	Title         string           `json:"title" binding:"required"`
	Subject       string           `json:"subject" binding:"required"`
	QuestionText  string           `json:"question_text" binding:"required"`
	MaxScore      float64          `json:"max_score"`
	ModelType     string           `json:"model_type"`
	CustomModelID *int64           `json:"custom_model_id"`
	Criteria      []CriterionInput `json:"criteria" binding:"required"`
}

type CriterionResponse struct {
	ID        int64   `json:"id"`
	Name      string  `json:"name"`
	Weight    float64 `json:"weight"`
	AnswerKey string  `json:"answer_key"`
	SortOrder int     `json:"sort_order"`
}

type AssignmentSummary struct {
	ID                  int64     `json:"id"`
	Title               string    `json:"title"`
	Subject             string    `json:"subject"`
	QuestionText        string    `json:"question_text"`
	MaxScore            float64   `json:"max_score"`
	ModelType           string    `json:"model_type"`
	CustomModelID       *int64    `json:"custom_model_id"`
	CustomModelName     string    `json:"custom_model_name,omitempty"`
	CriteriaCount       int       `json:"criteria_count"`
	SubmissionCount     int       `json:"submission_count"`
	ScoredCount         int       `json:"scored_count"`
	SimilarityCount     int       `json:"similarity_count"`
	TrainingSampleCount int       `json:"training_sample_count"`
	AverageScore        *float64  `json:"average_score"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

func (h *AssignmentHandler) List(c *gin.Context) {
	userID := c.GetInt64("user_id")
	ctx := c.Request.Context()

	rows, err := h.pool.Query(ctx, `
		SELECT
			a.id, a.title, a.subject, a.question_text, a.max_score,
			a.model_type, a.custom_model_id, COALESCE(cm.name, ''),
			(SELECT COUNT(*) FROM rubric_criteria rc WHERE rc.assignment_id = a.id) AS criteria_count,
			(SELECT COUNT(*) FROM submissions s WHERE s.assignment_id = a.id) AS submission_count,
			(SELECT COUNT(*) FROM submissions s WHERE s.assignment_id = a.id AND s.status = 'scored') AS scored_count,
			(SELECT COUNT(*) FROM similarity_flags sf WHERE sf.assignment_id = a.id) AS similarity_count,
			(SELECT COUNT(*) FROM training_samples ts WHERE ts.assignment_id = a.id) AS training_sample_count,
			(SELECT AVG(s.final_score) FROM submissions s WHERE s.assignment_id = a.id AND s.status = 'scored') AS avg_score,
			a.created_at, a.updated_at
		FROM assignments a
		LEFT JOIN custom_models cm ON cm.id = a.custom_model_id
		WHERE a.user_id = $1
		ORDER BY a.created_at DESC
	`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memuat daftar tugas."})
		return
	}
	defer rows.Close()

	assignments := make([]AssignmentSummary, 0)
	for rows.Next() {
		var item AssignmentSummary
		if err := rows.Scan(
			&item.ID, &item.Title, &item.Subject, &item.QuestionText, &item.MaxScore,
			&item.ModelType, &item.CustomModelID, &item.CustomModelName,
			&item.CriteriaCount, &item.SubmissionCount, &item.ScoredCount,
			&item.SimilarityCount, &item.TrainingSampleCount, &item.AverageScore,
			&item.CreatedAt, &item.UpdatedAt,
		); err == nil {
			assignments = append(assignments, item)
		}
	}

	c.JSON(http.StatusOK, gin.H{"assignments": assignments})
}

func (h *AssignmentHandler) Create(c *gin.Context) {
	userID := c.GetInt64("user_id")
	var req AssignmentInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Pastikan judul, mata pelajaran, soal, dan kriteria rubrik telah diisi dengan lengkap."})
		return
	}

	if len(req.Criteria) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Minimal harus terdapat 1 kriteria rubrik."})
		return
	}

	title := strings.TrimSpace(req.Title)
	subject := strings.TrimSpace(req.Subject)
	questionText := strings.TrimSpace(req.QuestionText)
	if title == "" || subject == "" || questionText == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Judul, mata pelajaran, dan teks soal tidak boleh kosong."})
		return
	}

	maxScore := req.MaxScore
	if maxScore <= 0 {
		maxScore = 100
	}

	modelType := req.ModelType
	if modelType != "custom" {
		modelType = "base"
		req.CustomModelID = nil
	}

	for idx, crit := range req.Criteria {
		if strings.TrimSpace(crit.Name) == "" || strings.TrimSpace(crit.AnswerKey) == "" || crit.Weight <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("Kriteria ke-%d belum lengkap. Pastikan nama, bobot (> 0), dan poin kunci jawaban terisi.", idx+1),
			})
			return
		}
	}

	ctx := c.Request.Context()
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi pembuatan tugas."})
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var assignmentID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO assignments (user_id, title, subject, question_text, max_score, model_type, custom_model_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`, userID, title, subject, questionText, maxScore, modelType, req.CustomModelID).Scan(&assignmentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan tugas baru."})
		return
	}

	for idx, crit := range req.Criteria {
		_, err = tx.Exec(ctx, `
			INSERT INTO rubric_criteria (assignment_id, name, weight, answer_key, sort_order)
			VALUES ($1, $2, $3, $4, $5)
		`, assignmentID, strings.TrimSpace(crit.Name), crit.Weight, strings.TrimSpace(crit.AnswerKey), idx+1)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan kriteria rubrik."})
			return
		}
	}

	if err := tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyelesaikan pembuatan tugas."})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":      assignmentID,
		"message": "Tugas dan rubrik berhasil dibuat.",
	})
}

func (h *AssignmentHandler) Update(c *gin.Context) {
	userID := c.GetInt64("user_id")
	assignmentID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tugas tidak valid."})
		return
	}

	var req AssignmentInput
	if err := c.ShouldBindJSON(&req); err != nil || len(req.Criteria) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data pembaruan tugas atau rubrik tidak lengkap."})
		return
	}

	maxScore := req.MaxScore
	if maxScore <= 0 {
		maxScore = 100
	}
	modelType := req.ModelType
	if modelType != "custom" {
		modelType = "base"
		req.CustomModelID = nil
	}

	ctx := c.Request.Context()
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi."})
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, `
		UPDATE assignments
		SET title = $1, subject = $2, question_text = $3, max_score = $4,
		    model_type = $5, custom_model_id = $6, updated_at = NOW()
		WHERE id = $7 AND user_id = $8
	`, strings.TrimSpace(req.Title), strings.TrimSpace(req.Subject), strings.TrimSpace(req.QuestionText),
		maxScore, modelType, req.CustomModelID, assignmentID, userID)
	if err != nil || tag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tugas tidak ditemukan."})
		return
	}

	_, _ = tx.Exec(ctx, `DELETE FROM rubric_criteria WHERE assignment_id = $1`, assignmentID)
	for idx, crit := range req.Criteria {
		_, err = tx.Exec(ctx, `
			INSERT INTO rubric_criteria (assignment_id, name, weight, answer_key, sort_order)
			VALUES ($1, $2, $3, $4, $5)
		`, assignmentID, strings.TrimSpace(crit.Name), crit.Weight, strings.TrimSpace(crit.AnswerKey), idx+1)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui kriteria rubrik."})
			return
		}
	}

	if err := tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan perubahan tugas."})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Tugas dan rubrik berhasil diperbarui."})
}

func (h *AssignmentHandler) Delete(c *gin.Context) {
	userID := c.GetInt64("user_id")
	assignmentID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tugas tidak valid."})
		return
	}

	ctx := c.Request.Context()
	tag, err := h.pool.Exec(ctx, `DELETE FROM assignments WHERE id = $1 AND user_id = $2`, assignmentID, userID)
	if err != nil || tag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tugas tidak ditemukan."})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Tugas berhasil dihapus."})
}

func (h *AssignmentHandler) GetDetail(c *gin.Context) {
	userID := c.GetInt64("user_id")
	assignmentID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tugas tidak valid."})
		return
	}

	ctx := c.Request.Context()
	var assignment AssignmentSummary
	err = h.pool.QueryRow(ctx, `
		SELECT
			a.id, a.title, a.subject, a.question_text, a.max_score,
			a.model_type, a.custom_model_id, COALESCE(cm.name, ''),
			(SELECT COUNT(*) FROM training_samples ts WHERE ts.assignment_id = a.id) AS training_sample_count,
			a.created_at, a.updated_at
		FROM assignments a
		LEFT JOIN custom_models cm ON cm.id = a.custom_model_id
		WHERE a.id = $1 AND a.user_id = $2
	`, assignmentID, userID).Scan(
		&assignment.ID, &assignment.Title, &assignment.Subject, &assignment.QuestionText,
		&assignment.MaxScore, &assignment.ModelType, &assignment.CustomModelID,
		&assignment.CustomModelName, &assignment.TrainingSampleCount,
		&assignment.CreatedAt, &assignment.UpdatedAt,
	)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tugas tidak ditemukan."})
		return
	}

	// Ambil kriteria rubrik
	critRows, err := h.pool.Query(ctx, `
		SELECT id, name, weight, answer_key, sort_order
		FROM rubric_criteria
		WHERE assignment_id = $1
		ORDER BY sort_order ASC, id ASC
	`, assignmentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memuat kriteria rubrik."})
		return
	}
	defer critRows.Close()

	criteria := make([]CriterionResponse, 0)
	for critRows.Next() {
		var cr CriterionResponse
		if err := critRows.Scan(&cr.ID, &cr.Name, &cr.Weight, &cr.AnswerKey, &cr.SortOrder); err == nil {
			criteria = append(criteria, cr)
		}
	}
	assignment.CriteriaCount = len(criteria)

	// Ambil daftar jawaban siswa beserta rincian skor per kriteria
	type CriterionScoreItem struct {
		CriterionID      int64    `json:"criterion_id"`
		CriterionName    string   `json:"criterion_name"`
		CriterionWeight  float64  `json:"criterion_weight"`
		CoverageScore    float64  `json:"coverage_score"`
		Status           string   `json:"status"`
		MatchedSnippet   string   `json:"matched_snippet"`
		MissingKeyPoints []string `json:"missing_key_points"`
		FeedbackSentence string   `json:"feedback_sentence"`
	}

	type SubmissionItem struct {
		ID             int64                `json:"id"`
		StudentName    string               `json:"student_name"`
		AnswerText     string               `json:"answer_text"`
		PredictedScore *float64             `json:"predicted_score"`
		RevisedScore   *float64             `json:"revised_score"`
		FinalScore     *float64             `json:"final_score"`
		FeedbackText   *string              `json:"feedback_text"`
		Status         string               `json:"status"`
		QuotaCharged   bool                 `json:"quota_charged"`
		CriteriaScores []CriterionScoreItem `json:"criteria_scores"`
		CreatedAt      time.Time            `json:"created_at"`
		UpdatedAt      time.Time            `json:"updated_at"`
	}

	subRows, err := h.pool.Query(ctx, `
		SELECT id, student_name, answer_text, predicted_score, revised_score,
		       final_score, feedback_text, status, quota_charged, created_at, updated_at
		FROM submissions
		WHERE assignment_id = $1
		ORDER BY id ASC
	`, assignmentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memuat jawaban siswa."})
		return
	}
	defer subRows.Close()

	submissions := make([]SubmissionItem, 0)
	subIndexMap := make(map[int64]int)
	scoredCount := 0
	var scoreSum float64

	for subRows.Next() {
		var s SubmissionItem
		if err := subRows.Scan(
			&s.ID, &s.StudentName, &s.AnswerText, &s.PredictedScore, &s.RevisedScore,
			&s.FinalScore, &s.FeedbackText, &s.Status, &s.QuotaCharged, &s.CreatedAt, &s.UpdatedAt,
		); err == nil {
			s.CriteriaScores = make([]CriterionScoreItem, 0)
			if s.Status == "scored" && s.FinalScore != nil {
				scoredCount++
				scoreSum += *s.FinalScore
			}
			subIndexMap[s.ID] = len(submissions)
			submissions = append(submissions, s)
		}
	}
	assignment.SubmissionCount = len(submissions)
	assignment.ScoredCount = scoredCount
	if scoredCount > 0 {
		avg := scoreSum / float64(scoredCount)
		assignment.AverageScore = &avg
	}

	// Ambil rincian skor kriteria seluruh siswa pada tugas ini
	if len(submissions) > 0 {
		csRows, err := h.pool.Query(ctx, `
			SELECT
				scs.submission_id, scs.criterion_id, rc.name, rc.weight,
				scs.coverage_score, scs.status, scs.matched_snippet,
				scs.missing_key_points, scs.feedback_sentence
			FROM submission_criteria_scores scs
			JOIN rubric_criteria rc ON rc.id = scs.criterion_id
			WHERE rc.assignment_id = $1
			ORDER BY rc.sort_order ASC, rc.id ASC
		`, assignmentID)
		if err == nil {
			defer csRows.Close()
			for csRows.Next() {
				var subID int64
				var item CriterionScoreItem
				var missingRaw []byte
				if err := csRows.Scan(
					&subID, &item.CriterionID, &item.CriterionName, &item.CriterionWeight,
					&item.CoverageScore, &item.Status, &item.MatchedSnippet,
					&missingRaw, &item.FeedbackSentence,
				); err == nil {
					_ = json.Unmarshal(missingRaw, &item.MissingKeyPoints)
					if item.MissingKeyPoints == nil {
						item.MissingKeyPoints = []string{}
					}
					if idx, ok := subIndexMap[subID]; ok {
						submissions[idx].CriteriaScores = append(submissions[idx].CriteriaScores, item)
					}
				}
			}
		}
	}

	// Ambil pasangan siswa yang terindikasi mirip
	type SimilarityFlagItem struct {
		ID             int64   `json:"id"`
		SubmissionAID  int64   `json:"submission_a_id"`
		StudentAName   string  `json:"student_a_name"`
		StudentAText   string  `json:"student_a_text"`
		SubmissionBID  int64   `json:"submission_b_id"`
		StudentBName   string  `json:"student_b_name"`
		StudentBText   string  `json:"student_b_text"`
		SemanticScore  float64 `json:"semantic_score"`
		LexicalScore   float64 `json:"lexical_score"`
		MatchedExcerpt string  `json:"matched_excerpt"`
	}

	simRows, err := h.pool.Query(ctx, `
		SELECT
			sf.id, sf.submission_a_id, sa.student_name, sa.answer_text,
			sf.submission_b_id, sb.student_name, sb.answer_text,
			sf.semantic_score, sf.lexical_score, sf.matched_excerpt
		FROM similarity_flags sf
		JOIN submissions sa ON sa.id = sf.submission_a_id
		JOIN submissions sb ON sb.id = sf.submission_b_id
		WHERE sf.assignment_id = $1
		ORDER BY sf.semantic_score DESC, sf.lexical_score DESC
	`, assignmentID)
	similarityFlags := make([]SimilarityFlagItem, 0)
	if err == nil {
		defer simRows.Close()
		for simRows.Next() {
			var sf SimilarityFlagItem
			if err := simRows.Scan(
				&sf.ID, &sf.SubmissionAID, &sf.StudentAName, &sf.StudentAText,
				&sf.SubmissionBID, &sf.StudentBName, &sf.StudentBText,
				&sf.SemanticScore, &sf.LexicalScore, &sf.MatchedExcerpt,
			); err == nil {
				similarityFlags = append(similarityFlags, sf)
			}
		}
	}
	assignment.SimilarityCount = len(similarityFlags)

	// Ambil job penilaian terakhir (bila ada)
	var latestJob map[string]interface{}
	var jobID, jobStatus, jobStage, jobErr string
	var jobProgress int
	var jobUpdated time.Time
	err = h.pool.QueryRow(ctx, `
		SELECT id::text, status, stage, progress, error_message, updated_at
		FROM jobs
		WHERE type = 'scoring' AND reference_id = $1 AND user_id = $2
		ORDER BY created_at DESC
		LIMIT 1
	`, assignmentID, userID).Scan(&jobID, &jobStatus, &jobStage, &jobProgress, &jobErr, &jobUpdated)
	if err == nil {
		latestJob = map[string]interface{}{
			"id":            jobID,
			"status":        jobStatus,
			"stage":         jobStage,
			"progress":      jobProgress,
			"error_message": jobErr,
			"updated_at":    jobUpdated,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"assignment":       assignment,
		"criteria":         criteria,
		"submissions":      submissions,
		"similarity_flags": similarityFlags,
		"latest_job":       latestJob,
	})
}

func (h *AssignmentHandler) AddManualSubmissions(c *gin.Context) {
	userID := c.GetInt64("user_id")
	assignmentID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tugas tidak valid."})
		return
	}

	type manualEntry struct {
		StudentName string `json:"student_name"`
		AnswerText  string `json:"answer_text"`
	}
	var req struct {
		Submissions []manualEntry `json:"submissions"`
		StudentName string        `json:"student_name"`
		AnswerText  string        `json:"answer_text"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format input jawaban siswa tidak valid."})
		return
	}

	entries := req.Submissions
	if len(entries) == 0 && (req.StudentName != "" || req.AnswerText != "") {
		entries = append(entries, manualEntry{
			StudentName: req.StudentName,
			AnswerText:  req.AnswerText,
		})
	}
	if len(entries) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Masukkan minimal 1 jawaban siswa."})
		return
	}

	ctx := c.Request.Context()
	var ownerExists bool
	_ = h.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM assignments WHERE id = $1 AND user_id = $2)`, assignmentID, userID).Scan(&ownerExists)
	if !ownerExists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tugas tidak ditemukan."})
		return
	}

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan jawaban siswa."})
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	added := 0
	for idx, entry := range entries {
		name := strings.TrimSpace(entry.StudentName)
		ans := strings.TrimSpace(entry.AnswerText)
		if name == "" || ans == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("Baris ke-%d: nama siswa dan teks jawaban wajib diisi.", idx+1),
			})
			return
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO submissions (assignment_id, student_name, answer_text, status, quota_charged)
			VALUES ($1, $2, $3, 'pending', FALSE)
		`, assignmentID, name, ans)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menambahkan jawaban siswa."})
			return
		}
		added++
	}

	if err := tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan data jawaban."})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"added_count": added,
		"message":     fmt.Sprintf("%d jawaban siswa berhasil ditambahkan.", added),
	})
}

type RowValidationError struct {
	Row     int    `json:"row"`
	Message string `json:"message"`
}

func (h *AssignmentHandler) UploadExcelSubmissions(c *gin.Context) {
	userID := c.GetInt64("user_id")
	assignmentID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tugas tidak valid."})
		return
	}

	ctx := c.Request.Context()
	var ownerExists bool
	_ = h.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM assignments WHERE id = $1 AND user_id = $2)`, assignmentID, userID).Scan(&ownerExists)
	if !ownerExists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tugas tidak ditemukan."})
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Pilih file Excel (.xlsx) terlebih dahulu."})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Gagal membaca isi file Excel."})
		return
	}

	xl, err := excelize.OpenReader(bytes.NewReader(buf))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File bukan merupakan dokumen Excel (.xlsx) yang valid."})
		return
	}
	defer func() { _ = xl.Close() }()

	sheetName := xl.GetSheetName(0)
	if sheetName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File Excel tidak memiliki lembar kerja (sheet)."})
		return
	}

	rows, err := xl.GetRows(sheetName)
	if err != nil || len(rows) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File Excel kosong atau belum memiliki baris data di bawah header."})
		return
	}

	// Cari indeks kolom nama_siswa dan jawaban pada baris header
	header := rows[0]
	colNama := -1
	colJawaban := -1
	for i, hName := range header {
		norm := strings.ToLower(strings.TrimSpace(hName))
		if norm == "nama_siswa" || norm == "nama siswa" || norm == "nama" {
			colNama = i
		} else if norm == "jawaban" || norm == "jawaban_siswa" || norm == "jawaban siswa" {
			colJawaban = i
		}
	}

	if colNama == -1 || colJawaban == -1 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Header kolom Excel wajib memuat 'nama_siswa' dan 'jawaban'. Gunakan template yang tersedia.",
		})
		return
	}

	type validRow struct {
		RowNum      int
		StudentName string
		AnswerText  string
	}
	validRows := make([]validRow, 0)
	rowErrors := make([]RowValidationError, 0)

	for rIdx := 1; rIdx < len(rows); rIdx++ {
		excelRowNum := rIdx + 1
		row := rows[rIdx]

		var nameVal, ansVal string
		if colNama < len(row) {
			nameVal = strings.TrimSpace(row[colNama])
		}
		if colJawaban < len(row) {
			ansVal = strings.TrimSpace(row[colJawaban])
		}

		// Abaikan baris yang benar-benar kosong di akhir sheet
		if nameVal == "" && ansVal == "" {
			continue
		}

		if nameVal == "" {
			rowErrors = append(rowErrors, RowValidationError{
				Row:     excelRowNum,
				Message: "Kolom nama_siswa kosong.",
			})
			continue
		}
		if ansVal == "" {
			rowErrors = append(rowErrors, RowValidationError{
				Row:     excelRowNum,
				Message: fmt.Sprintf("Jawaban untuk siswa '%s' kosong.", nameVal),
			})
			continue
		}

		validRows = append(validRows, validRow{
			RowNum:      excelRowNum,
			StudentName: nameVal,
			AnswerText:  ansVal,
		})
	}

	dryRun := c.Query("dry_run") == "true"
	if dryRun || len(rowErrors) > 0 {
		status := http.StatusOK
		if len(rowErrors) > 0 && !dryRun {
			status = http.StatusUnprocessableEntity
		}
		c.JSON(status, gin.H{
			"valid_count": len(validRows),
			"error_count": len(rowErrors),
			"row_errors":  rowErrors,
			"saved":       false,
		})
		return
	}

	if len(validRows) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tidak ditemukan baris jawaban siswa yang valid di dalam file Excel."})
		return
	}

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai penyimpanan jawaban."})
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for _, vr := range validRows {
		_, err := tx.Exec(ctx, `
			INSERT INTO submissions (assignment_id, student_name, answer_text, status, quota_charged)
			VALUES ($1, $2, $3, 'pending', FALSE)
		`, assignmentID, vr.StudentName, vr.AnswerText)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan baris jawaban dari Excel."})
			return
		}
	}

	if err := tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan seluruh jawaban dari Excel."})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"valid_count": len(validRows),
		"error_count": 0,
		"row_errors":  []RowValidationError{},
		"saved":       true,
		"message":     fmt.Sprintf("%d jawaban siswa berhasil diunggah dari Excel.", len(validRows)),
	})
}

func (h *AssignmentHandler) DeleteSubmission(c *gin.Context) {
	userID := c.GetInt64("user_id")
	subID, err := strconv.ParseInt(c.Param("subId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID jawaban tidak valid."})
		return
	}

	ctx := c.Request.Context()
	tag, err := h.pool.Exec(ctx, `
		DELETE FROM submissions s
		USING assignments a
		WHERE s.id = $1 AND s.assignment_id = a.id AND a.user_id = $2
	`, subID, userID)
	if err != nil || tag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Jawaban siswa tidak ditemukan."})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Jawaban siswa berhasil dihapus."})
}

func (h *AssignmentHandler) ReviseSubmissionScore(c *gin.Context) {
	userID := c.GetInt64("user_id")
	subID, err := strconv.ParseInt(c.Param("subId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID jawaban tidak valid."})
		return
	}

	var req struct {
		RevisedScore float64 `json:"revised_score"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Masukkan nilai revisi yang valid."})
		return
	}

	ctx := c.Request.Context()
	var assignmentID int64
	var maxScore float64
	var studentName, answerText string

	err = h.pool.QueryRow(ctx, `
		SELECT s.assignment_id, a.max_score, s.student_name, s.answer_text
		FROM submissions s
		JOIN assignments a ON a.id = s.assignment_id
		WHERE s.id = $1 AND a.user_id = $2
	`, subID, userID).Scan(&assignmentID, &maxScore, &studentName, &answerText)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Jawaban siswa tidak ditemukan."})
		return
	}

	if req.RevisedScore < 0 || req.RevisedScore > maxScore {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Nilai revisi harus berada di antara 0 dan %.0f.", maxScore),
		})
		return
	}

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi revisi nilai."})
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = tx.Exec(ctx, `
		UPDATE submissions
		SET revised_score = $1, final_score = $1, updated_at = NOW()
		WHERE id = $2
	`, req.RevisedScore, subID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui nilai siswa."})
		return
	}

	// Upsert ke training_samples (terhubung ke assignment_id dan submission_id, dengan target_score mentah)
	_, err = tx.Exec(ctx, `
		INSERT INTO training_samples (assignment_id, submission_id, student_name, answer_text, target_score, source)
		VALUES ($1, $2, $3, $4, $5, 'revision')
		ON CONFLICT (submission_id) DO UPDATE SET
			student_name = EXCLUDED.student_name,
			answer_text = EXCLUDED.answer_text,
			target_score = EXCLUDED.target_score,
			updated_at = NOW()
	`, assignmentID, subID, studentName, answerText, req.RevisedScore)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan hasil revisi ke data latih."})
		return
	}

	if err := tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan revisi nilai."})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "Nilai akhir berhasil direvisi dan disimpan sebagai sampel data latih.",
		"revised_score": req.RevisedScore,
		"final_score":   req.RevisedScore,
	})
}

func (h *AssignmentHandler) EnqueueScoringJob(c *gin.Context) {
	userID := c.GetInt64("user_id")
	assignmentID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tugas tidak valid."})
		return
	}

	ctx := c.Request.Context()
	var exists bool
	_ = h.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM assignments WHERE id = $1 AND user_id = $2)`, assignmentID, userID).Scan(&exists)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tugas tidak ditemukan."})
		return
	}

	// Hitung total jawaban siswa dan berapa jawaban yang BELUM pernah memotong kuota (quota_charged = false)
	var totalSubmissions, unchargedCount int
	err = h.pool.QueryRow(ctx, `
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE quota_charged = FALSE)
		FROM submissions
		WHERE assignment_id = $1
	`, assignmentID).Scan(&totalSubmissions, &unchargedCount)
	if err != nil || totalSubmissions == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Belum ada jawaban siswa pada tugas ini untuk dinilai."})
		return
	}

	// Cek langganan & reset bulanan bila Free Plan
	var planCode string
	var planQuota, quotaTotal, quotaUsed int
	var quotaResetAt, expiresAt *time.Time
	err = h.pool.QueryRow(ctx, `
		SELECT p.code, p.quota_essays, s.quota_total, s.quota_used, s.quota_reset_at, s.expires_at
		FROM subscriptions s
		JOIN plans p ON p.id = s.plan_id
		WHERE s.user_id = $1
	`, userID).Scan(&planCode, &planQuota, &quotaTotal, &quotaUsed, &quotaResetAt, &expiresAt)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{
			"error":          "Data langganan tidak ditemukan.",
			"quota_exceeded": true,
		})
		return
	}

	now := time.Now()
	if planCode == "free" && quotaResetAt != nil && now.After(*quotaResetAt) {
		nextReset := now.AddDate(0, 1, 0)
		_, _ = h.pool.Exec(ctx, `
			UPDATE subscriptions
			SET quota_used = 0, quota_total = $1, quota_reset_at = $2, updated_at = NOW()
			WHERE user_id = $3
		`, planQuota, nextReset, userID)
		quotaUsed = 0
		quotaTotal = planQuota
	}

	if planCode != "free" && expiresAt != nil && now.After(*expiresAt) {
		c.JSON(http.StatusPaymentRequired, gin.H{
			"error":          "Masa aktif paket langganan Anda telah berakhir. Silakan perpanjang paket di halaman Langganan.",
			"quota_exceeded": true,
		})
		return
	}

	remaining := quotaTotal - quotaUsed
	if unchargedCount > remaining {
		c.JSON(http.StatusPaymentRequired, gin.H{
			"error": fmt.Sprintf(
				"Kuota penilaian tidak mencukupi. Anda membutuhkan %d kuota baru, sedangkan sisa kuota Anda adalah %d esai. Silakan tingkatkan paket atau beli Add-on Kuota.",
				unchargedCount, maxInt(remaining, 0),
			),
			"quota_exceeded":  true,
			"quota_needed":    unchargedCount,
			"quota_remaining": maxInt(remaining, 0),
		})
		return
	}

	// Buat baris job di tabel jobs dan masukkan ke Redis queue:scoring
	var jobID string
	err = h.pool.QueryRow(ctx, `
		INSERT INTO jobs (user_id, type, queue_name, reference_id, status, stage, progress)
		VALUES ($1, 'scoring', $2, $3, 'queued', 'antri', 0)
		RETURNING id::text
	`, userID, h.cfg.ScoringQueue, assignmentID).Scan(&jobID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat antrian penilaian."})
		return
	}

	payloadBytes, _ := json.Marshal(map[string]interface{}{
		"job_id":        jobID,
		"type":          "scoring",
		"assignment_id": assignmentID,
		"user_id":       userID,
	})

	if err := h.rdb.RPush(ctx, h.cfg.ScoringQueue, string(payloadBytes)).Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengirim job ke antrian Redis."})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"job_id":  jobID,
		"status":  "queued",
		"stage":   "antri",
		"message": "Job penilaian berhasil dimasukkan ke antrian.",
	})
}

func (h *AssignmentHandler) GetJobStatus(c *gin.Context) {
	userID := c.GetInt64("user_id")
	jobID := c.Param("id")
	ctx := c.Request.Context()

	var status, stage, errMsg, jobType string
	var referenceID int64
	var progress int
	var metaRaw []byte
	var createdAt, updatedAt time.Time

	err := h.pool.QueryRow(ctx, `
		SELECT type, reference_id, status, stage, progress, error_message, metadata, created_at, updated_at
		FROM jobs
		WHERE id = $1::uuid AND user_id = $2
	`, jobID, userID).Scan(
		&jobType, &referenceID, &status, &stage, &progress, &errMsg, &metaRaw, &createdAt, &updatedAt,
	)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job tidak ditemukan."})
		return
	}

	var metadata map[string]interface{}
	_ = json.Unmarshal(metaRaw, &metadata)

	c.JSON(http.StatusOK, gin.H{
		"job": gin.H{
			"id":            jobID,
			"type":          jobType,
			"reference_id":  referenceID,
			"status":        status,
			"stage":         stage,
			"progress":      progress,
			"error_message": errMsg,
			"metadata":      metadata,
			"created_at":    createdAt,
			"updated_at":    updatedAt,
		},
	})
}

func (h *AssignmentHandler) DownloadTemplate(c *gin.Context) {
	templateType := c.Query("type")
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()

	sheet := "Template"
	f.SetSheetName("Sheet1", sheet)

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "#FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#FF5347"}, Pattern: 1},
	})

	if templateType == "training" {
		_ = f.SetCellValue(sheet, "A1", "nama_siswa")
		_ = f.SetCellValue(sheet, "B1", "jawaban")
		_ = f.SetCellValue(sheet, "C1", "nilai")
		_ = f.SetCellStyle(sheet, "A1", "C1", headerStyle)
		_ = f.SetColWidth(sheet, "A", "A", 28)
		_ = f.SetColWidth(sheet, "B", "B", 85)
		_ = f.SetColWidth(sheet, "C", "C", 14)
		_ = f.SetCellValue(sheet, "A2", "Andi Pratama")
		_ = f.SetCellValue(sheet, "B2", "Tuliskan teks jawaban esai siswa di kolom ini...")
		_ = f.SetCellValue(sheet, "C2", 85)
		c.Header("Content-Disposition", "attachment; filename=template_data_latih.xlsx")
	} else {
		_ = f.SetCellValue(sheet, "A1", "nama_siswa")
		_ = f.SetCellValue(sheet, "B1", "jawaban")
		_ = f.SetCellStyle(sheet, "A1", "B1", headerStyle)
		_ = f.SetColWidth(sheet, "A", "A", 28)
		_ = f.SetColWidth(sheet, "B", "B", 85)
		_ = f.SetCellValue(sheet, "A2", "Andi Pratama")
		_ = f.SetCellValue(sheet, "B2", "Tuliskan teks jawaban esai siswa di kolom ini...")
		c.Header("Content-Disposition", "attachment; filename=template_jawaban_siswa.xlsx")
	}

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	_ = f.Write(c.Writer)
}

func (h *AssignmentHandler) ExportResultsExcel(c *gin.Context) {
	userID := c.GetInt64("user_id")
	assignmentID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tugas tidak valid."})
		return
	}

	ctx := c.Request.Context()
	var title, subject string
	var maxScore float64
	err = h.pool.QueryRow(ctx, `
		SELECT title, subject, max_score
		FROM assignments
		WHERE id = $1 AND user_id = $2
	`, assignmentID, userID).Scan(&title, &subject, &maxScore)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tugas tidak ditemukan."})
		return
	}

	// Ambil kriteria rubrik
	critRows, err := h.pool.Query(ctx, `
		SELECT id, name, weight
		FROM rubric_criteria
		WHERE assignment_id = $1
		ORDER BY sort_order ASC, id ASC
	`, assignmentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memuat kriteria rubrik."})
		return
	}
	defer critRows.Close()

	type critCol struct {
		ID     int64
		Name   string
		Weight float64
	}
	var criteria []critCol
	for critRows.Next() {
		var cc critCol
		if err := critRows.Scan(&cc.ID, &cc.Name, &cc.Weight); err == nil {
			criteria = append(criteria, cc)
		}
	}

	// Ambil pasangan siswa yang terkena flag kemiripan
	simMap := make(map[int64][]string)
	simRows, err := h.pool.Query(ctx, `
		SELECT sf.submission_a_id, sa.student_name, sf.submission_b_id, sb.student_name, sf.semantic_score, sf.lexical_score
		FROM similarity_flags sf
		JOIN submissions sa ON sa.id = sf.submission_a_id
		JOIN submissions sb ON sb.id = sf.submission_b_id
		WHERE sf.assignment_id = $1
	`, assignmentID)
	if err == nil {
		defer simRows.Close()
		for simRows.Next() {
			var idA, idB int64
			var nameA, nameB string
			var sSem, sLex float64
			if err := simRows.Scan(&idA, &nameA, &idB, &nameB, &sSem, &sLex); err == nil {
				simMap[idA] = append(simMap[idA], fmt.Sprintf("Mirip dengan %s (Semantik %.0f%%, Leksikal %.0f%%)", nameB, sSem*100, sLex*100))
				simMap[idB] = append(simMap[idB], fmt.Sprintf("Mirip dengan %s (Semantik %.0f%%, Leksikal %.0f%%)", nameA, sSem*100, sLex*100))
			}
		}
	}

	// Ambil skor per kriteria untuk tiap submission
	statusMap := make(map[string]string)
	csRows, err := h.pool.Query(ctx, `
		SELECT scs.submission_id, scs.criterion_id, scs.status
		FROM submission_criteria_scores scs
		JOIN rubric_criteria rc ON rc.id = scs.criterion_id
		WHERE rc.assignment_id = $1
	`, assignmentID)
	if err == nil {
		defer csRows.Close()
		for csRows.Next() {
			var sID, cID int64
			var st string
			if err := csRows.Scan(&sID, &cID, &st); err == nil {
				key := fmt.Sprintf("%d_%d", sID, cID)
				statusMap[key] = formatCriterionStatusText(st)
			}
		}
	}

	f := excelize.NewFile()
	defer func() { _ = f.Close() }()

	sheet := "Hasil Penilaian"
	f.SetSheetName("Sheet1", sheet)

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "#FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#FF5347"}, Pattern: 1},
	})

	headers := []string{"No", "Nama Siswa", "Nilai AI", "Nilai Revisi Guru", "Nilai Akhir"}
	for _, cItem := range criteria {
		headers = append(headers, fmt.Sprintf("Status: %s", cItem.Name))
	}
	headers = append(headers, "Umpan Balik (Feedback)", "Indikasi Kemiripan", "Jawaban Siswa")

	for colIdx, hTitle := range headers {
		cell, _ := excelize.CoordinatesToCellName(colIdx+1, 1)
		_ = f.SetCellValue(sheet, cell, hTitle)
		_ = f.SetCellStyle(sheet, cell, cell, headerStyle)
	}

	subRows, err := h.pool.Query(ctx, `
		SELECT id, student_name, answer_text, predicted_score, revised_score, final_score, COALESCE(feedback_text, '')
		FROM submissions
		WHERE assignment_id = $1
		ORDER BY id ASC
	`, assignmentID)
	if err == nil {
		defer subRows.Close()
		rowNum := 2
		no := 1
		for subRows.Next() {
			var sID int64
			var sName, sAns, sFeedback string
			var predScore, revScore, finScore *float64
			if err := subRows.Scan(&sID, &sName, &sAns, &predScore, &revScore, &finScore, &sFeedback); err == nil {
				vals := []interface{}{no, sName}
				if predScore != nil {
					vals = append(vals, *predScore)
				} else {
					vals = append(vals, "-")
				}
				if revScore != nil {
					vals = append(vals, *revScore)
				} else {
					vals = append(vals, "-")
				}
				if finScore != nil {
					vals = append(vals, *finScore)
				} else {
					vals = append(vals, "-")
				}

				for _, cItem := range criteria {
					key := fmt.Sprintf("%d_%d", sID, cItem.ID)
					stText, ok := statusMap[key]
					if !ok {
						stText = "Tidak ada"
					}
					vals = append(vals, stText)
				}

				simNotes := "Tidak ada"
				if notes, ok := simMap[sID]; ok && len(notes) > 0 {
					simNotes = strings.Join(notes, "; ")
				}
				vals = append(vals, sFeedback, simNotes, sAns)

				for cIdx, v := range vals {
					cell, _ := excelize.CoordinatesToCellName(cIdx+1, rowNum)
					_ = f.SetCellValue(sheet, cell, v)
				}
				rowNum++
				no++
			}
		}
	}

	_ = f.SetColWidth(sheet, "A", "A", 6)
	_ = f.SetColWidth(sheet, "B", "B", 26)
	_ = f.SetColWidth(sheet, "C", "E", 16)
	lastCol, _ := excelize.ColumnNumberToName(len(headers))
	_ = f.SetColWidth(sheet, "F", lastCol, 32)

	filename := fmt.Sprintf("hasil_koreksi_tugas_%d.xlsx", assignmentID)
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	_ = f.Write(c.Writer)
}

func formatCriterionStatusText(st string) string {
	switch st {
	case "terpenuhi":
		return "Terpenuhi"
	case "sebagian":
		return "Sebagian"
	default:
		return "Tidak ada"
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (h *AssignmentHandler) ListTrainingSamples(c *gin.Context) {
	userID := c.GetInt64("user_id")
	assignmentID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tugas tidak valid."})
		return
	}

	ctx := c.Request.Context()
	var exists bool
	_ = h.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM assignments WHERE id = $1 AND user_id = $2)`, assignmentID, userID).Scan(&exists)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tugas tidak ditemukan."})
		return
	}

	type sampleItem struct {
		ID           int64     `json:"id"`
		AssignmentID int64     `json:"assignment_id"`
		SubmissionID *int64    `json:"submission_id"`
		StudentName  string    `json:"student_name"`
		AnswerText   string    `json:"answer_text"`
		TargetScore  float64   `json:"target_score"`
		Source       string    `json:"source"`
		UpdatedAt    time.Time `json:"updated_at"`
	}

	rows, err := h.pool.Query(ctx, `
		SELECT id, assignment_id, submission_id, student_name, answer_text, target_score, source, updated_at
		FROM training_samples
		WHERE assignment_id = $1
		ORDER BY updated_at DESC, id DESC
	`, assignmentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memuat sampel data latih."})
		return
	}
	defer rows.Close()

	samples := make([]sampleItem, 0)
	for rows.Next() {
		var s sampleItem
		if err := rows.Scan(&s.ID, &s.AssignmentID, &s.SubmissionID, &s.StudentName, &s.AnswerText, &s.TargetScore, &s.Source, &s.UpdatedAt); err == nil {
			samples = append(samples, s)
		}
	}

	c.JSON(http.StatusOK, gin.H{"samples": samples})
}

func (h *AssignmentHandler) GuruStats(c *gin.Context) {
	userID := c.GetInt64("user_id")
	ctx := c.Request.Context()

	var totalAssignments, totalSubmissions, totalScored, totalModels int
	_ = h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM assignments WHERE user_id = $1`, userID).Scan(&totalAssignments)
	_ = h.pool.QueryRow(ctx, `
		SELECT COUNT(*), COUNT(*) FILTER (WHERE s.status = 'scored')
		FROM submissions s
		JOIN assignments a ON a.id = s.assignment_id
		WHERE a.user_id = $1
	`, userID).Scan(&totalSubmissions, &totalScored)
	_ = h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM custom_models WHERE user_id = $1`, userID).Scan(&totalModels)

	c.JSON(http.StatusOK, gin.H{
		"total_assignments": totalAssignments,
		"total_submissions": totalSubmissions,
		"total_scored":      totalScored,
		"total_models":      totalModels,
	})
}

