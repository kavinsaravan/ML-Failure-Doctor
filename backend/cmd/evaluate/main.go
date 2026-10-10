// Evaluate the production diagnosis path without HTTP, stored labels, or cached reports.
package main

import (
	"crypto/sha256"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"crashlens/db"
	"crashlens/diagnosis"
	"crashlens/fireworks"
)

type Case struct {
	Hardware   json.RawMessage `json:"hardware,omitempty"`
	ID         string          `json:"id"`
	Provenance string          `json:"provenance"`
	Expected   string          `json:"expected_failure_type"`
	Logs       string          `json:"logs"`
	Metrics    string          `json:"gpu_metrics"`
	Cause      string          `json:"expected_root_cause"`
	Fix        string          `json:"acceptable_remediation"`
}
type Dataset struct {
	Schema      int    `json:"schema_version"`
	Description string `json:"description"`
	Cases       []Case `json:"cases"`
}
type Prediction struct {
	Case
	Correct bool             `json:"classification_correct"`
	Report  diagnosis.Report `json:"report"`
}
type Score struct {
	Total    int     `json:"total"`
	Correct  int     `json:"correct"`
	Accuracy float64 `json:"accuracy_percent"`
}

func score(total, correct int) Score {
	s := Score{Total: total, Correct: correct}
	if total > 0 {
		s.Accuracy = 100 * float64(correct) / float64(total)
	}
	return s
}
func wilson(n, k int) []float64 {
	p, z := float64(k)/float64(n), 1.96
	d := 1 + z*z/float64(n)
	center := (p + z*z/(2*float64(n))) / d
	half := z * math.Sqrt(p*(1-p)/float64(n)+z*z/(4*float64(n*n))) / d
	return []float64{100 * (center - half), 100 * (center + half)}
}
func run() error {
	path := flag.String("cases", "../evaluation/cases.json", "Dataset file")
	output := flag.String("out", "../evaluation/results/rules", "Output directory")
	mode := flag.String("mode", "rules", "rules or fireworks (paid; requires environment key and model)")
	flag.Parse()
	raw, err := os.ReadFile(*path)
	if err != nil {
		return err
	}
	var data Dataset
	if err = json.Unmarshal(raw, &data); err != nil {
		return err
	}
	if data.Schema != 1 || len(data.Cases) == 0 {
		return fmt.Errorf("invalid dataset")
	}
	var client *fireworks.Client
	if *mode == "fireworks" {
		client = fireworks.NewClient()
		if client == nil {
			return fmt.Errorf("set both FIREWORKS_API_KEY and FIREWORKS_MODEL; do not put credentials in arguments")
		}
	} else if *mode != "rules" {
		return fmt.Errorf("mode must be rules or fireworks")
	}
	seen := map[string]bool{}
	logsSeen := map[string]bool{}
	for _, c := range data.Cases {
		if c.ID == "" || c.Logs == "" || c.Expected == "" || seen[c.ID] || logsSeen[c.Logs] {
			return fmt.Errorf("empty/duplicate case or duplicate log: %s", c.ID)
		}
		seen[c.ID] = true
		logsSeen[c.Logs] = true
	}
	if err = os.MkdirAll(*output, 0755); err != nil {
		return err
	}
	predictions := []Prediction{}
	correct, ai := 0, 0
	confusion := map[string]map[string]int{}
	provenance := map[string]Score{}
	for _, c := range data.Cases {
		// Ground-truth labels and rubric never enter the diagnosis input.
		w := &db.Workload{Name: "Evaluation workload", Type: "ML_JOB", Status: "failed", JobLogs: &c.Logs, GPUMetrics: &c.Metrics}
		report := diagnosis.RunDiagnosis(w, client)
		match := report.FailureType == c.Expected
		if match {
			correct++
		}
		if report.Source == "ai" {
			ai++
		}
		predictions = append(predictions, Prediction{Case: c, Correct: match, Report: report})
		if confusion[c.Expected] == nil {
			confusion[c.Expected] = map[string]int{}
		}
		confusion[c.Expected][report.FailureType]++
		s := provenance[c.Provenance]
		s.Total++
		if match {
			s.Correct++
		}
		provenance[c.Provenance] = s
		// Checkpoint each result so paid runs interrupted later retain their responses.
		bytes, _ := json.MarshalIndent(predictions, "", "  ")
		if err = os.WriteFile(filepath.Join(*output, "predictions.json"), append(bytes, '\n'), 0644); err != nil {
			return err
		}
	}
	macroF1 := 0.0
	for label, row := range confusion {
		tp, fn, fp := row[label], 0, 0
		for predicted, count := range row {
			if predicted != label {
				fn += count
			}
		}
		for actual, other := range confusion {
			if actual != label {
				fp += other[label]
			}
		}
		if 2*tp+fn+fp > 0 {
			macroF1 += 2 * float64(tp) / float64(2*tp+fn+fp)
		}
	}
	macroF1 = 100 * macroF1 / float64(len(confusion))
	for label, s := range provenance {
		provenance[label] = score(s.Total, s.Correct)
	}
	revision := "unknown"
	if out, err := exec.Command("git", "rev-parse", "HEAD").Output(); err == nil {
		revision = strings.TrimSpace(string(out))
	}
	dirty := true
	if out, err := exec.Command("git", "status", "--porcelain").Output(); err == nil {
		dirty = len(out) > 0
	}

	summary := map[string]interface{}{
		"mode": *mode, "model": "", "code_revision": revision, "working_tree_dirty": dirty, "dataset_sha256": fmt.Sprintf("%x", sha256.Sum256(raw)),
		"dataset_description": data.Description, "generated_at": time.Now().UTC(),
		"classification": score(len(predictions), correct), "macro_f1_percent": macroF1,
		"wilson_95_percent_interval": wilson(len(predictions), correct), "confusion_matrix": confusion, "by_provenance": provenance,
		"ai_reports": ai, "rules_reports": len(predictions) - ai,
		"classification_origin":  "Production heuristic classifier in both modes; Fireworks does not independently choose the category.",
		"root_cause_correctness": "pending human review", "remediation_correctness": "pending human review",
		"limitations": "Controlled exploratory dataset; consult the dataset description and per-case provenance for hardware execution. Repeated variants are correlated. Wilson interval assumes independent cases and is descriptive only.",
	}
	if client != nil {
		summary["model"] = client.Model
	}
	bytes, _ := json.MarshalIndent(summary, "", "  ")
	if err = os.WriteFile(filepath.Join(*output, "summary.json"), append(bytes, '\n'), 0644); err != nil {
		return err
	}
	file, err := os.Create(filepath.Join(*output, "review.csv"))
	if err != nil {
		return err
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	writer.Write([]string{"id", "root_cause_correct", "remediation_correct", "reviewer", "notes", "expected_root_cause", "acceptable_remediation", "predicted_root_cause", "predicted_fix"})
	for _, p := range predictions {
		writer.Write([]string{p.ID, "", "", "", "", p.Cause, p.Fix, p.Report.RootCause, p.Report.RecommendedFix})
	}
	writer.Flush()
	if err = writer.Error(); err != nil {
		return err
	}
	fmt.Printf("Classification: %d/%d (%.1f%%); macro F1 %.1f%%. Root cause and remediation need human review.\n", correct, len(predictions), 100*float64(correct)/float64(len(predictions)), macroF1)
	return nil
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
