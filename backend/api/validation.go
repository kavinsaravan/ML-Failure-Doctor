package api

import (
	"crashlens/db"
	"encoding/json"
	"fmt"
	"strings"
)

func validStatus(s string) bool {
	return s == "pending" || s == "running" || s == "failed" || s == "succeeded"
}
func validateIdentity(name, kind string) error {
	if strings.TrimSpace(name) == "" || len(name) > 200 {
		return fmt.Errorf("name must contain 1–200 bytes of nonblank text")
	}
	if kind != "ML_JOB" {
		return fmt.Errorf("type must be ML_JOB")
	}
	return nil
}
func validTransition(from, to string) bool {
	if to == "" || from == to {
		return true
	}
	return from == "pending" && (to == "running" || to == "failed" || to == "succeeded") || from == "running" && (to == "failed" || to == "succeeded")
}
func validateUpdates(u db.Workload) error {
	if u.Status != "" && !validStatus(u.Status) {
		return fmt.Errorf("invalid status")
	}
	if u.Name != "" || u.Type != "" {
		return fmt.Errorf("name and type cannot be updated")
	}
	if u.RuntimeSeconds != nil && *u.RuntimeSeconds < 0 {
		return fmt.Errorf("runtime_seconds cannot be negative")
	}
	if u.WastedGPUSeconds != nil && *u.WastedGPUSeconds < 0 {
		return fmt.Errorf("wasted_gpu_seconds cannot be negative")
	}
	if u.GPUMetrics == nil || *u.GPUMetrics == "" {
		return nil
	}
	var samples []map[string]json.RawMessage
	if err := json.Unmarshal([]byte(*u.GPUMetrics), &samples); err != nil || samples == nil {
		return fmt.Errorf("gpu_metrics must encode an array of sample objects")
	}
	if len(samples) > 300 {
		return fmt.Errorf("gpu_metrics exceeds 300 samples")
	}
	for _, sample := range samples {
		if sample == nil {
			return fmt.Errorf("GPU samples cannot be null")
		}
		for _, key := range []string{"gpu_memory_used_mb", "gpu_memory_total_mb", "gpu_memory_percent"} {
			raw, ok := sample[key]
			if !ok || string(raw) == "null" {
				return fmt.Errorf("GPU sample requires numeric %s", key)
			}
		}
		for key, raw := range sample {
			switch key {
			case "gpu_memory_used_mb", "gpu_memory_total_mb", "gpu_memory_percent", "gpu_memory_peak_mb", "gpu_tensor_memory_mb", "gpu_utilization_percent", "temperature_celsius", "device_index":
				if string(raw) == "null" {
					continue
				}
				var n float64
				if json.Unmarshal(raw, &n) != nil || (key != "temperature_celsius" && n < 0) {
					return fmt.Errorf("invalid GPU sample field %s", key)
				}
				if key == "gpu_utilization_percent" && n > 100 {
					return fmt.Errorf("GPU utilization exceeds 100 percent")
				}
			case "timestamp", "source", "device_id", "device_name", "memory_scope", "memory_total_basis":
				if string(raw) == "null" {
					continue
				}
				var value string
				if json.Unmarshal(raw, &value) != nil {
					return fmt.Errorf("invalid GPU sample field %s", key)
				}
			}
		}
	}
	return nil
}
