package db

import (
	"database/sql"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Workload struct {
	ID                 int        `json:"id"`
	Name               string     `json:"name"`
	Type               string     `json:"type"` // ML_JOB or AGENT_RUN
	Status             string     `json:"status"` // pending, running, failed, succeeded
	FailureType        *string    `json:"failure_type,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	StartedAt          *time.Time `json:"started_at,omitempty"`
	FinishedAt         *time.Time `json:"finished_at,omitempty"`
	RuntimeSeconds     *float64   `json:"runtime_seconds,omitempty"`
	ExitCode           *int       `json:"exit_code,omitempty"`
	WastedGPUSeconds   *float64   `json:"wasted_gpu_seconds,omitempty"`
	JobLogs            *string    `json:"job_logs,omitempty"`
	GPUMetrics         *string    `json:"gpu_metrics,omitempty"`
	CheckpointState    *string    `json:"checkpoint_state,omitempty"`
	FailureReport      *string    `json:"failure_report,omitempty"`
	AgentSteps         *string    `json:"agent_steps,omitempty"`
	ToolCalls          *string    `json:"tool_calls,omitempty"`
	ModelCalls         *string    `json:"model_calls,omitempty"`
	TraceEvents        *string    `json:"trace_events,omitempty"`
}

type DB struct {
	*sql.DB
}

func New(dbPath string) (*DB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	if err := initSchema(db); err != nil {
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
		agent_steps TEXT,
		tool_calls TEXT,
		model_calls TEXT,
		trace_events TEXT
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
			continue
		}
		workloads = append(workloads, w)
	}
	return workloads, nil
}

func (db *DB) GetWorkload(id string) (*Workload, error) {
	var workload Workload
	err := db.QueryRow(`
		SELECT id, name, type, status, failure_type, created_at, started_at,
		       finished_at, runtime_seconds, exit_code, wasted_gpu_seconds,
		       job_logs, gpu_metrics, checkpoint_state, failure_report,
		       agent_steps, tool_calls, model_calls, trace_events
		FROM workloads WHERE id = ?
	`, id).Scan(
		&workload.ID, &workload.Name, &workload.Type, &workload.Status,
		&workload.FailureType, &workload.CreatedAt, &workload.StartedAt,
		&workload.FinishedAt, &workload.RuntimeSeconds, &workload.ExitCode,
		&workload.WastedGPUSeconds, &workload.JobLogs, &workload.GPUMetrics,
		&workload.CheckpointState, &workload.FailureReport, &workload.AgentSteps,
		&workload.ToolCalls, &workload.ModelCalls, &workload.TraceEvents,
	)
	return &workload, err
}

func (db *DB) CreateWorkload(name, workloadType, status string) (int64, error) {
	result, err := db.Exec(`
		INSERT INTO workloads (name, type, status)
		VALUES (?, ?, ?)
	`, name, workloadType, status)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (db *DB) UpdateWorkload(id string, workload *Workload) error {
	_, err := db.Exec(`
		UPDATE workloads
		SET name = ?, type = ?, status = ?, failure_type = ?,
		    started_at = ?, finished_at = ?, runtime_seconds = ?,
		    exit_code = ?, wasted_gpu_seconds = ?,
		    job_logs = ?, gpu_metrics = ?, checkpoint_state = ?,
		    failure_report = ?, agent_steps = ?, tool_calls = ?,
		    model_calls = ?, trace_events = ?
		WHERE id = ?
	`, workload.Name, workload.Type, workload.Status, workload.FailureType,
		workload.StartedAt, workload.FinishedAt, workload.RuntimeSeconds,
		workload.ExitCode, workload.WastedGPUSeconds, workload.JobLogs,
		workload.GPUMetrics, workload.CheckpointState, workload.FailureReport,
		workload.AgentSteps, workload.ToolCalls, workload.ModelCalls,
		workload.TraceEvents, id)
	return err
}

func (db *DB) GetStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	var total int
	db.QueryRow("SELECT COUNT(*) FROM workloads").Scan(&total)
	stats["total_workloads"] = total

	var failed int
	db.QueryRow("SELECT COUNT(*) FROM workloads WHERE status = 'failed'").Scan(&failed)
	stats["failed_workloads"] = failed

	var succeeded int
	db.QueryRow("SELECT COUNT(*) FROM workloads WHERE status = 'succeeded'").Scan(&succeeded)
	stats["succeeded_workloads"] = succeeded

	var wastedGPU float64
	db.QueryRow("SELECT COALESCE(SUM(wasted_gpu_seconds), 0) FROM workloads WHERE status = 'failed'").Scan(&wastedGPU)
	stats["wasted_gpu_seconds"] = wastedGPU

	rows, _ := db.Query("SELECT failure_type, COUNT(*) FROM workloads WHERE failure_type IS NOT NULL GROUP BY failure_type")
	defer rows.Close()

	failureTypes := make(map[string]int)
	for rows.Next() {
		var ftype string
		var count int
		rows.Scan(&ftype, &count)
		failureTypes[ftype] = count
	}
	stats["failure_types"] = failureTypes

	return stats, nil
}
