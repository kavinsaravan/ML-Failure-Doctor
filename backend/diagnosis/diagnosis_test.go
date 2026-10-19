package diagnosis

import (
	"crashlens/db"
	"strings"
	"testing"
)

func TestStoredClassificationDoesNotInflateConfidence(t *testing.T) {
	logs, kind := "unexpected failure", "UNKNOWN_ERROR"
	report := RunDiagnosis(&db.Workload{JobLogs: &logs, FailureType: &kind}, nil)
	if report.Confidence != .5 || report.ConfidenceBasis != "heuristic_log_match" {
		t.Fatalf("misleading confidence: %+v", report)
	}
}
func TestRetryRequiresFixForMissingDependency(t *testing.T) {
	logs := "ModuleNotFoundError: no module named torch"
	if RunDiagnosis(&db.Workload{JobLogs: &logs}, nil).SafeToRetry {
		t.Fatal("unchanged retry cannot install dependencies")
	}
}

func TestMPSOOMGuidance(t *testing.T) {
	logs := "RuntimeError: MPS backend out of memory"
	report := RunDiagnosis(&db.Workload{JobLogs: &logs}, nil)
	if report.Source != "rules" || report.SafeToRetry || !strings.Contains(report.RecommendedFix, "torch.mps.empty_cache") || !strings.Contains(report.RootCause, "Apple MPS") {
		t.Fatalf("incorrect MPS fallback: %+v", report)
	}
}
