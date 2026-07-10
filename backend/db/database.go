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

// AgentRun represents an observed agent execution
type AgentRun struct {
	ID               string     `json:"id"`
	WorkloadID       string     `json:"workload_id"`
	AgentName        string     `json:"agent_name"`
	Task             string     `json:"task"`
	Status           string     `json:"status"`
	FailureType      *string    `json:"failure_type,omitempty"`
	TotalToolCalls   int        `json:"total_tool_calls"`
	TotalModelCalls  int        `json:"total_model_calls"`
	TotalTokens      *int       `json:"total_tokens,omitempty"`
	TotalLatencyMS   *int       `json:"total_latency_ms,omitempty"`
	StartedAt        time.Time  `json:"started_at"`
	FinishedAt       *time.Time `json:"finished_at,omitempty"`
}

// AgentStep represents a single step in an agent execution trace
type AgentStep struct {
	ID          string     `json:"id"`
	AgentRunID  string     `json:"agent_run_id"`
	StepIndex   int        `json:"step_index"`
	StepType    string     `json:"step_type"` // TOOL_CALL, MODEL_CALL, DECISION, ERROR, FINAL_RESPONSE
	Name        string     `json:"name"`
	Input       *string    `json:"input,omitempty"`
	Output      *string    `json:"output,omitempty"`
	Status      string     `json:"status"`
	LatencyMS   *int       `json:"latency_ms,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
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

	CREATE TABLE IF NOT EXISTS agent_runs (
		id TEXT PRIMARY KEY,
		workload_id TEXT NOT NULL,
		agent_name TEXT NOT NULL,
		task TEXT,
		status TEXT NOT NULL,
		failure_type TEXT,
		total_tool_calls INTEGER DEFAULT 0,
		total_model_calls INTEGER DEFAULT 0,
		total_tokens INTEGER,
		total_latency_ms INTEGER,
		started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		finished_at TIMESTAMP,
		FOREIGN KEY (workload_id) REFERENCES workloads(id)
	);

	CREATE TABLE IF NOT EXISTS agent_steps (
		id TEXT PRIMARY KEY,
		agent_run_id TEXT NOT NULL,
		step_index INTEGER NOT NULL,
		step_type TEXT NOT NULL,
		name TEXT NOT NULL,
		input TEXT,
		output TEXT,
		status TEXT NOT NULL,
		latency_ms INTEGER,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (agent_run_id) REFERENCES agent_runs(id)
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

// Agent Run methods for observability
func (db *DB) CreateAgentRun(run *AgentRun) error {
	_, err := db.Exec(`
		INSERT INTO agent_runs (id, workload_id, agent_name, task, status, failure_type, total_tool_calls, total_model_calls, total_tokens, total_latency_ms, started_at, finished_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, run.ID, run.WorkloadID, run.AgentName, run.Task, run.Status, run.FailureType, run.TotalToolCalls, run.TotalModelCalls, run.TotalTokens, run.TotalLatencyMS, run.StartedAt, run.FinishedAt)
	return err
}

func (db *DB) UpdateAgentRun(run *AgentRun) error {
	_, err := db.Exec(`
		UPDATE agent_runs SET status = ?, failure_type = ?, total_tool_calls = ?, total_model_calls = ?, total_tokens = ?, total_latency_ms = ?, finished_at = ?
		WHERE id = ?
	`, run.Status, run.FailureType, run.TotalToolCalls, run.TotalModelCalls, run.TotalTokens, run.TotalLatencyMS, run.FinishedAt, run.ID)
	return err
}

func (db *DB) GetAgentRuns() ([]AgentRun, error) {
	rows, err := db.Query(`SELECT id, workload_id, agent_name, task, status, failure_type, total_tool_calls, total_model_calls, total_tokens, total_latency_ms, started_at, finished_at FROM agent_runs ORDER BY started_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []AgentRun
	for rows.Next() {
		var run AgentRun
		err := rows.Scan(&run.ID, &run.WorkloadID, &run.AgentName, &run.Task, &run.Status, &run.FailureType, &run.TotalToolCalls, &run.TotalModelCalls, &run.TotalTokens, &run.TotalLatencyMS, &run.StartedAt, &run.FinishedAt)
		if err != nil {
			return nil, err
		}
		runs = append(runs, run)
	}
	return runs, nil
}

func (db *DB) GetAgentRun(id string) (*AgentRun, error) {
	run := &AgentRun{}
	err := db.QueryRow(`SELECT id, workload_id, agent_name, task, status, failure_type, total_tool_calls, total_model_calls, total_tokens, total_latency_ms, started_at, finished_at FROM agent_runs WHERE id = ?`, id).
		Scan(&run.ID, &run.WorkloadID, &run.AgentName, &run.Task, &run.Status, &run.FailureType, &run.TotalToolCalls, &run.TotalModelCalls, &run.TotalTokens, &run.TotalLatencyMS, &run.StartedAt, &run.FinishedAt)
	if err != nil {
		return nil, err
	}
	return run, nil
}

func (db *DB) GetAgentRunByWorkloadID(workloadID int) (*AgentRun, error) {
	run := &AgentRun{}
	err := db.QueryRow(`SELECT id, workload_id, agent_name, task, status, failure_type, total_tool_calls, total_model_calls, total_tokens, total_latency_ms, started_at, finished_at FROM agent_runs WHERE workload_id = ?`, workloadID).
		Scan(&run.ID, &run.WorkloadID, &run.AgentName, &run.Task, &run.Status, &run.FailureType, &run.TotalToolCalls, &run.TotalModelCalls, &run.TotalTokens, &run.TotalLatencyMS, &run.StartedAt, &run.FinishedAt)
	if err != nil {
		return nil, err
	}
	return run, nil
}

// Agent Step methods
func (db *DB) CreateAgentStep(step *AgentStep) error {
	_, err := db.Exec(`
		INSERT INTO agent_steps (id, agent_run_id, step_index, step_type, name, input, output, status, latency_ms, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, step.ID, step.AgentRunID, step.StepIndex, step.StepType, step.Name, step.Input, step.Output, step.Status, step.LatencyMS, step.CreatedAt)
	return err
}

func (db *DB) GetAgentSteps(agentRunID string) ([]AgentStep, error) {
	rows, err := db.Query(`SELECT id, agent_run_id, step_index, step_type, name, input, output, status, latency_ms, created_at FROM agent_steps WHERE agent_run_id = ? ORDER BY step_index ASC`, agentRunID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var steps []AgentStep
	for rows.Next() {
		var step AgentStep
		err := rows.Scan(&step.ID, &step.AgentRunID, &step.StepIndex, &step.StepType, &step.Name, &step.Input, &step.Output, &step.Status, &step.LatencyMS, &step.CreatedAt)
		if err != nil {
			return nil, err
		}
		steps = append(steps, step)
	}
	return steps, nil
}
