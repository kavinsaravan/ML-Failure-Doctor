package db

import (
	"crashlens/classifier"
	"crashlens/metrics"
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Workload struct {
	OwnerID          string     `json:"-"`
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
		failure_report TEXT,
  runner_managed INTEGER NOT NULL DEFAULT 0
	);
	`
	if _, err := db.Exec(schema); err != nil {
		return err
	}
	for _, column := range []struct{ name, definition string }{
		{"runner_managed", "INTEGER NOT NULL DEFAULT 0"},
		{"owner_id", "TEXT NOT NULL DEFAULT 'legacy'"},
	} {
		var count int
		if err := db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('workloads') WHERE name = ?", column.name).Scan(&count); err != nil {
			return err
		}
		if count == 0 {
			if _, err := db.Exec("ALTER TABLE workloads ADD COLUMN " + column.name + " " + column.definition); err != nil {
				return err
			}
		}
	}
	_, err := db.Exec(`CREATE INDEX IF NOT EXISTS workloads_owner ON workloads(owner_id);
 CREATE TABLE IF NOT EXISTS api_keys (
 id TEXT PRIMARY KEY, owner_id TEXT NOT NULL, name TEXT NOT NULL,
 key_hash TEXT NOT NULL UNIQUE, created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
 revoked_at TIMESTAMP);
 `)
	return err
}

func (db *DB) GetWorkloads(owner ...string) ([]Workload, error) {
	rows, err := db.Query(`
		SELECT id, name, type, status, failure_type, created_at, started_at,
		       finished_at, runtime_seconds, exit_code, wasted_gpu_seconds
		FROM workloads WHERE (? IS NULL OR owner_id = ?)
		ORDER BY created_at DESC
	`, ownerFilter(owner), ownerFilter(owner))
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

func (db *DB) GetWorkload(id string, owner ...string) (*Workload, error) {
	var workload Workload
	err := db.QueryRow(`
		SELECT id, name, type, status, failure_type, created_at, started_at,
		       finished_at, runtime_seconds, exit_code, wasted_gpu_seconds,
		       job_logs, gpu_metrics, checkpoint_state, failure_report, owner_id
		FROM workloads WHERE id = ? AND (? IS NULL OR owner_id = ?)
	`, id, ownerFilter(owner), ownerFilter(owner)).Scan(
		&workload.ID, &workload.Name, &workload.Type, &workload.Status,
		&workload.FailureType, &workload.CreatedAt, &workload.StartedAt,
		&workload.FinishedAt, &workload.RuntimeSeconds, &workload.ExitCode,
		&workload.WastedGPUSeconds, &workload.JobLogs, &workload.GPUMetrics,
		&workload.CheckpointState, &workload.FailureReport, &workload.OwnerID,
	)
	return &workload, err
}

func (db *DB) CreateWorkload(name, workloadType, status string, owner ...string) (int64, error) {
	result, err := db.Exec(`
		INSERT INTO workloads (name, type, status, started_at, owner_id)
  VALUES (?, ?, ?, ?, ?)
	`, name, workloadType, status, startedAt(status), creationOwner(owner))
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

var ErrWorkloadChanged = errors.New("workload changed during update")

func (db *DB) UpdateWorkload(id string, workload *Workload) error {
	return db.updateWorkload(id, workload, nil)
}
func (db *DB) UpdateWorkloadIfStatus(id string, workload *Workload, status string) error {
	return db.updateWorkload(id, workload, &status)
}
func (db *DB) updateWorkload(id string, workload *Workload, expectedStatus *string) error {
	NormalizeLifecycle(workload)
	result, err := db.Exec(`
		UPDATE workloads
		SET name = ?, type = ?, status = ?, failure_type = ?,
		    started_at = ?, finished_at = ?, runtime_seconds = ?,
		    exit_code = ?, wasted_gpu_seconds = ?,
		    job_logs = ?, gpu_metrics = ?, checkpoint_state = ?,
		    failure_report = ?
  WHERE id = ? AND (? IS NULL OR status = ?) AND owner_id = ?
 `, workload.Name, workload.Type, workload.Status, workload.FailureType,
		workload.StartedAt, workload.FinishedAt, workload.RuntimeSeconds,
		workload.ExitCode, workload.WastedGPUSeconds, workload.JobLogs,
		workload.GPUMetrics, workload.CheckpointState, workload.FailureReport, id, expectedStatus, expectedStatus, creationOwner([]string{workload.OwnerID}))
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return ErrWorkloadChanged
	}
	return nil
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

func (db *DB) GetStats(owner ...string) (map[string]interface{}, error) {
	stats := make(map[string]interface{})
	var total, failed, succeeded int
	var wastedGPU float64
	err := db.QueryRow(`SELECT COUNT(*),
  COALESCE(SUM(status = 'failed'), 0), COALESCE(SUM(status = 'succeeded'), 0),
  COALESCE(SUM(CASE WHEN status = 'failed' THEN wasted_gpu_seconds ELSE 0 END), 0)
  FROM workloads WHERE (? IS NULL OR owner_id = ?)`, ownerFilter(owner), ownerFilter(owner)).Scan(&total, &failed, &succeeded, &wastedGPU)
	if err != nil {
		return nil, err
	}
	stats["total_workloads"] = total
	stats["failed_workloads"] = failed
	stats["succeeded_workloads"] = succeeded
	stats["wasted_gpu_seconds"] = wastedGPU
	rows, err := db.Query("SELECT failure_type, COUNT(*) FROM workloads WHERE status = 'failed' AND failure_type IS NOT NULL AND (? IS NULL OR owner_id = ?) GROUP BY failure_type", ownerFilter(owner), ownerFilter(owner))
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
	stats["backend_gpu_platform"] = metrics.GetCollector("").Name()
	return stats, nil
}

func (db *DB) CreateManagedWorkload(name, kind string, owner ...string) (int64, error) {
	result, err := db.Exec("INSERT INTO workloads (name,type,status,runner_managed,owner_id) VALUES (?,?,'pending',1,?)", name, kind, creationOwner(owner))
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}
func (db *DB) UpdateTelemetry(id int, logs, metrics string, runtime float64) error {
	_, err := db.Exec("UPDATE workloads SET job_logs=?,gpu_metrics=?,runtime_seconds=? WHERE id=? AND status='running'", logs, metrics, runtime, id)
	return err
}

// Only server-managed processes were lost. SDK jobs can keep running on external hosts.
func (db *DB) RecoverManagedJobs(reason string) error {
	rows, err := db.Query("SELECT id FROM workloads WHERE runner_managed=1 AND status IN ('pending','running')")
	if err != nil {
		return err
	}
	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, id := range ids {
		w, err := db.GetWorkload(fmt.Sprint(id))
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		runtime := 0.0
		if w.StartedAt != nil {
			runtime = now.Sub(*w.StartedAt).Seconds()
		}
		logs := ""
		if w.JobLogs != nil {
			logs = *w.JobLogs
		}
		logs += "\nJob interrupted: " + reason
		code := 1
		failure := classifier.UnknownError
		w.Status = "failed"
		w.FinishedAt = &now
		if w.StartedAt != nil {
			w.RuntimeSeconds = &runtime
		} else {
			w.RuntimeSeconds = nil
		}
		w.ExitCode = &code
		w.JobLogs = &logs
		w.FailureType = &failure
		if err := db.UpdateWorkload(fmt.Sprint(id), w); err != nil {
			return err
		}
	}
	return nil
}

func ownerFilter(owner []string) interface{} {
	if len(owner) == 0 {
		return nil
	}
	return owner[0]
}
func creationOwner(owner []string) string {
	if len(owner) == 0 || owner[0] == "" {
		return "legacy"
	}
	return owner[0]
}
