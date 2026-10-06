package db

import (
	"crashlens/classifier"
	"crashlens/metrics"
	"database/sql"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Workload struct {
	ID               int        `json:"id"`
	Name             string     `json:"name"`
	Type             string     `json:"type"`   // ML_JOB
	Status           string     `json:"status"` // pending, running, failed, succeeded
	FailureType      *string    `json:"failure_type,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	StartedAt        *time.Time `json:"started_at,omitempty"`
	FinishedAt       *time.Time `json:"finished_at,omitempty"`
	RuntimeSeconds   *float64   `json:"runtime_seconds,omitempty"`
	ExitCode         *int       `json:"exit_code,omitempty"`
	WastedGPUSeconds *float64   `json:"wasted_gpu_seconds,omitempty"`
	JobLogs          *string    `json:"job_logs,omitempty"`
	GPUMetrics       *string    `json:"gpu_metrics,omitempty"`
	CheckpointState  *string    `json:"checkpoint_state,omitempty"`
	FailureReport    *string    `json:"failure_report,omitempty"`
}

type DB struct {
	*sql.DB
}

func New(dbPath string) (*DB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(1)
	if err := initSchema(db); err != nil {
		db.Close()
		return nil, err
	}

	return &DB{db}, nil
}

func initSchema(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS workloads (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		type TEXT NOT NULL,
		status TEXT NOT NULL,
		failure_type TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		started_at TIMESTAMP,
		finished_at TIMESTAMP,
		runtime_seconds REAL,
		exit_code INTEGER,
		wasted_gpu_seconds REAL,
		job_logs TEXT,
		gpu_metrics TEXT,
		checkpoint_state TEXT,
		failure_report TEXT
	);
	`
	_, err := db.Exec(schema)
	return err
}

func (db *DB) GetWorkloads() ([]Workload, error) {
	rows, err := db.Query(`
		SELECT id, name, type, status, failure_type, created_at, started_at,
		       finished_at, runtime_seconds, exit_code, wasted_gpu_seconds
		FROM workloads
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	workloads := []Workload{}
	for rows.Next() {
		var w Workload
		err := rows.Scan(&w.ID, &w.Name, &w.Type, &w.Status, &w.FailureType,
			&w.CreatedAt, &w.StartedAt, &w.FinishedAt, &w.RuntimeSeconds,
			&w.ExitCode, &w.WastedGPUSeconds)
		if err != nil {
			return nil, err
		}
		workloads = append(workloads, w)
	}
	return workloads, rows.Err()
}

func (db *DB) GetWorkload(id string) (*Workload, error) {
	var workload Workload
	err := db.QueryRow(`
		SELECT id, name, type, status, failure_type, created_at, started_at,
		       finished_at, runtime_seconds, exit_code, wasted_gpu_seconds,
		       job_logs, gpu_metrics, checkpoint_state, failure_report
		FROM workloads WHERE id = ?
	`, id).Scan(
		&workload.ID, &workload.Name, &workload.Type, &workload.Status,
		&workload.FailureType, &workload.CreatedAt, &workload.StartedAt,
		&workload.FinishedAt, &workload.RuntimeSeconds, &workload.ExitCode,
		&workload.WastedGPUSeconds, &workload.JobLogs, &workload.GPUMetrics,
		&workload.CheckpointState, &workload.FailureReport,
	)
	return &workload, err
}

func (db *DB) CreateWorkload(name, workloadType, status string) (int64, error) {
	result, err := db.Exec(`
		INSERT INTO workloads (name, type, status, started_at)
  VALUES (?, ?, ?, ?)
	`, name, workloadType, status, startedAt(status))
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (db *DB) UpdateWorkload(id string, workload *Workload) error {
	NormalizeLifecycle(workload)
	_, err := db.Exec(`
		UPDATE workloads
		SET name = ?, type = ?, status = ?, failure_type = ?,
		    started_at = ?, finished_at = ?, runtime_seconds = ?,
		    exit_code = ?, wasted_gpu_seconds = ?,
		    job_logs = ?, gpu_metrics = ?, checkpoint_state = ?,
		    failure_report = ?
		WHERE id = ?
	`, workload.Name, workload.Type, workload.Status, workload.FailureType,
		workload.StartedAt, workload.FinishedAt, workload.RuntimeSeconds,
		workload.ExitCode, workload.WastedGPUSeconds, workload.JobLogs,
		workload.GPUMetrics, workload.CheckpointState, workload.FailureReport, id)
	return err
}

// NormalizeLifecycle applies server-owned metadata to SDK and runner updates.
func NormalizeLifecycle(w *Workload) {
	now := time.Now().UTC()
	if w.Status == "running" && w.StartedAt == nil {
		w.StartedAt = &now
	}
	if w.Status == "failed" || w.Status == "succeeded" {
		if w.FinishedAt == nil {
			w.FinishedAt = &now
		}
		if w.StartedAt == nil && w.RuntimeSeconds != nil {
			started := w.FinishedAt.Add(-time.Duration(*w.RuntimeSeconds * float64(time.Second)))
			w.StartedAt = &started
		}
	}
	if w.Status == "failed" {
		if w.FailureType == nil && w.JobLogs != nil {
			metrics := ""
			if w.GPUMetrics != nil {
				metrics = *w.GPUMetrics
			}
			failure := classifier.ClassifyFailure(*w.JobLogs, metrics)
			w.FailureType = &failure
		}
		// Single-GPU estimate; clients may supply a measured multi-GPU value.
		if w.WastedGPUSeconds == nil && w.RuntimeSeconds != nil {
			wasted := classifier.CalculateWastedGPUSeconds(*w.RuntimeSeconds, 1)
			w.WastedGPUSeconds = &wasted
		}
	} else if w.Status == "succeeded" {
		w.FailureType = nil
		w.WastedGPUSeconds = nil
		w.FailureReport = nil
	}
}

func startedAt(status string) *time.Time {
	if status != "running" {
		return nil
	}
	now := time.Now().UTC()
	return &now
}

func (db *DB) GetStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})
	var total, failed, succeeded int
	var wastedGPU float64
	err := db.QueryRow(`SELECT COUNT(*),
  COALESCE(SUM(status = 'failed'), 0), COALESCE(SUM(status = 'succeeded'), 0),
  COALESCE(SUM(CASE WHEN status = 'failed' THEN wasted_gpu_seconds ELSE 0 END), 0)
  FROM workloads`).Scan(&total, &failed, &succeeded, &wastedGPU)
	if err != nil {
		return nil, err
	}
	stats["total_workloads"] = total
	stats["failed_workloads"] = failed
	stats["succeeded_workloads"] = succeeded
	stats["wasted_gpu_seconds"] = wastedGPU
	rows, err := db.Query("SELECT failure_type, COUNT(*) FROM workloads WHERE status = 'failed' AND failure_type IS NOT NULL GROUP BY failure_type")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	failureTypes := make(map[string]int)
	for rows.Next() {
		var ftype string
		var count int
		if err := rows.Scan(&ftype, &count); err != nil {
			return nil, err
		}
		failureTypes[ftype] = count
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	stats["failure_types"] = failureTypes
	stats["gpu_platform"] = metrics.GetCollector("").Name()
	return stats, nil
}
