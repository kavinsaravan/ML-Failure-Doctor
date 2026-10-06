package fireworks

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

type responseTransport func(*http.Request) (*http.Response, error)

func (f responseTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestMalformedAIResponseRejected(t *testing.T) {
	c := &Client{BaseURL: "https://test.invalid", Client: &http.Client{Transport: responseTransport(func(r *http.Request) (*http.Response, error) {
		body := `{"choices":[{"message":{"tool_calls":[{"function":{"name":"diagnose_ml_failure","arguments":"{}"}}]}}]}`
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})}}
	if report, err := c.DiagnoseFailure("logs"); err == nil || report != nil {
		t.Fatal("empty AI report accepted")
	}
}

func TestValidAIResponseRetainsPrevention(t *testing.T) {
	expected := DiagnosisResult{RootCause: "GPU OOM", Evidence: []string{"CUDA out of memory"}, RecommendedFixes: []string{"Reduce batch size"}, Prevention: "Monitor memory"}
	args, err := json.Marshal(expected)
	if err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(map[string]interface{}{"choices": []interface{}{map[string]interface{}{"message": map[string]interface{}{"tool_calls": []interface{}{map[string]interface{}{"function": map[string]string{"name": "diagnose_ml_failure", "arguments": string(args)}}}}}}})
	if err != nil {
		t.Fatal(err)
	}
	c := &Client{BaseURL: "https://test.invalid", Client: &http.Client{Transport: responseTransport(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(body))), Header: make(http.Header)}, nil
	})}}
	result, err := c.DiagnoseFailure("logs")
	if err != nil || result == nil || result.Prevention != expected.Prevention {
		t.Fatalf("valid report lost: %+v %v", result, err)
	}
}
