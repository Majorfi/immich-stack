package utils

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"
)

// captureOutput captures log output for testing
func captureOutput(f func()) string {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)
	f()
	return buf.String()
}

func TestRoute(t *testing.T) {
	tests := []struct {
		name           string
		setupRequest   func() *http.Request
		expectedFields []string
	}{
		{
			name: "request with X-Forwarded-For header",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/test/path", nil)
				req.Header.Set("X-Forwarded-For", "192.168.1.1")
				return req
			},
			expectedFields: []string{"192.168.1.1", "/test/path"},
		},
		{
			name: "request with empty X-Forwarded-For header",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/test/path", nil)
				req.Header.Set("X-Forwarded-For", "")
				return req
			},
			expectedFields: []string{"/test/path"},
		},
		{
			name: "request without X-Forwarded-For header",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/test/path", nil)
				req.RemoteAddr = "10.0.0.1:12345"
				return req
			},
			expectedFields: []string{"10.0.0.1", "/test/path"},
		},
		{
			name: "request with nil headers",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/test/path", nil)
				req.Header = nil
				req.RemoteAddr = "127.0.0.1:8080"
				return req
			},
			expectedFields: []string{"127.0.0.1", "/test/path"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.setupRequest()
			output := captureOutput(func() {
				Route(req)
			})

			// Check that all expected fields are present in the output
			for _, field := range tt.expectedFields {
				if !strings.Contains(output, field) {
					t.Errorf("Expected output to contain %q, got: %s", field, output)
				}
			}

			// Check for goroutine count format
			if !regexp.MustCompile(`\[\d+\]`).MatchString(output) {
				t.Error("Expected goroutine count in brackets")
			}
		})
	}
}

func TestWebhook(t *testing.T) {
	tests := []struct {
		name     string
		hookName string
		field    string
		item     string
		verb     string
	}{
		{
			name:     "facebook webhook",
			hookName: "Facebook",
			field:    "messages",
			item:     "user",
			verb:     "POST",
		},
		{
			name:     "empty values",
			hookName: "",
			field:    "",
			item:     "",
			verb:     "",
		},
		{
			name:     "special characters",
			hookName: "Test-Hook",
			field:    "field/with/slashes",
			item:     "item.with.dots",
			verb:     "PUT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureOutput(func() {
				Webhook(tt.hookName, tt.field, tt.item, tt.verb)
			})

			// Check that all parameters are present in output
			expectedParts := []string{
				tt.hookName,
				tt.field,
				tt.item,
				tt.verb,
			}

			for _, part := range expectedParts {
				if !strings.Contains(output, part) {
					t.Errorf("Expected output to contain %q, got: %s", part, output)
				}
			}

			// Check for goroutine count and webhook format
			if !regexp.MustCompile(`\[\d+\]`).MatchString(output) {
				t.Error("Expected goroutine count in brackets")
			}
			if !strings.Contains(output, "[Webhook - ") {
				t.Error("Expected webhook label format")
			}
		})
	}
}

func TestErrorCrash(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
	}{
		{
			name:  "string error",
			input: "critical error occurred",
		},
		{
			name:  "integer error",
			input: 404,
		},
		{
			name:  "nil error",
			input: nil,
		},
		{
			name:  "struct error",
			input: struct{ Message string }{"test error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture both stdout and stderr since spew.Printf might use either
			oldStdout := os.Stdout
			oldStderr := os.Stderr
			
			r, w, _ := os.Pipe()
			os.Stdout = w
			os.Stderr = w

			ErrorCrash(tt.input)

			w.Close()
			os.Stdout = oldStdout
			os.Stderr = oldStderr

			var buf bytes.Buffer
			buf.ReadFrom(r)
			output := buf.String()

			// Check for expected elements
			if !regexp.MustCompile(`\[\d+\]`).MatchString(output) {
				t.Error("Expected goroutine count in brackets")
			}
			if !strings.Contains(output, "[ERROR]") {
				t.Error("Expected ERROR label")
			}
			if !regexp.MustCompile(`\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}`).MatchString(output) {
				t.Error("Expected timestamp format")
			}
		})
	}
}

func TestError(t *testing.T) {
	tests := []struct {
		name   string
		errors []interface{}
	}{
		{
			name:   "single error",
			errors: []interface{}{"single error message"},
		},
		{
			name:   "multiple errors",
			errors: []interface{}{"first error", "second error", 123},
		},
		{
			name:   "empty errors",
			errors: []interface{}{},
		},
		{
			name:   "nil error",
			errors: []interface{}{nil},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture both stdout and stderr
			oldStdout := os.Stdout
			oldStderr := os.Stderr
			
			r, w, _ := os.Pipe()
			os.Stdout = w
			os.Stderr = w

			Error(tt.errors...)

			w.Close()
			os.Stdout = oldStdout
			os.Stderr = oldStderr

			var buf bytes.Buffer
			buf.ReadFrom(r)
			output := buf.String()

			// Check for expected elements
			if !regexp.MustCompile(`\[\d+\]`).MatchString(output) {
				t.Error("Expected goroutine count in brackets")
			}
			if !strings.Contains(output, "[ERROR]") {
				t.Error("Expected ERROR label")
			}
			if !regexp.MustCompile(`\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}`).MatchString(output) {
				t.Error("Expected timestamp format")
			}

			// For multiple errors, expect separator lines
			if len(tt.errors) > 1 {
				separatorCount := strings.Count(output, "----------------------------------")
				if separatorCount < 2 {
					t.Error("Expected separator lines for multiple errors")
				}
			}
		})
	}
}

func TestSuccess(t *testing.T) {
	tests := []interface{}{
		"operation completed successfully",
		123,
		struct{ Status string }{"OK"},
		nil,
	}

	for i, input := range tests {
		t.Run(fmt.Sprintf("test_%d", i), func(t *testing.T) {
			oldStdout := os.Stdout
			oldStderr := os.Stderr
			
			r, w, _ := os.Pipe()
			os.Stdout = w
			os.Stderr = w

			Success(input)

			w.Close()
			os.Stdout = oldStdout
			os.Stderr = oldStderr

			var buf bytes.Buffer
			buf.ReadFrom(r)
			output := buf.String()

			if !regexp.MustCompile(`\[\d+\]`).MatchString(output) {
				t.Error("Expected goroutine count in brackets")
			}
			if !strings.Contains(output, "[SUCCESS]") {
				t.Error("Expected SUCCESS label")
			}
			if !regexp.MustCompile(`\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}`).MatchString(output) {
				t.Error("Expected timestamp format")
			}
		})
	}
}

func TestWarning(t *testing.T) {
	tests := []interface{}{
		"this is a warning",
		456,
		struct{ Level string }{"WARN"},
		nil,
	}

	for i, input := range tests {
		t.Run(fmt.Sprintf("test_%d", i), func(t *testing.T) {
			oldStdout := os.Stdout
			oldStderr := os.Stderr
			
			r, w, _ := os.Pipe()
			os.Stdout = w
			os.Stderr = w

			Warning(input)

			w.Close()
			os.Stdout = oldStdout
			os.Stderr = oldStderr

			var buf bytes.Buffer
			buf.ReadFrom(r)
			output := buf.String()

			if !regexp.MustCompile(`\[\d+\]`).MatchString(output) {
				t.Error("Expected goroutine count in brackets")
			}
			if !strings.Contains(output, "[WARNING]") {
				t.Error("Expected WARNING label")
			}
			if !regexp.MustCompile(`\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}`).MatchString(output) {
				t.Error("Expected timestamp format")
			}
		})
	}
}

func TestInfo(t *testing.T) {
	tests := []struct {
		name string
		info []interface{}
	}{
		{
			name: "single info",
			info: []interface{}{"information message"},
		},
		{
			name: "multiple info",
			info: []interface{}{"info1", "info2", 789},
		},
		{
			name: "empty info",
			info: []interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldStdout := os.Stdout
			oldStderr := os.Stderr
			
			r, w, _ := os.Pipe()
			os.Stdout = w
			os.Stderr = w

			Info(tt.info...)

			w.Close()
			os.Stdout = oldStdout
			os.Stderr = oldStderr

			var buf bytes.Buffer
			buf.ReadFrom(r)
			output := buf.String()

			if !regexp.MustCompile(`\[\d+\]`).MatchString(output) {
				t.Error("Expected goroutine count in brackets")
			}
			if !strings.Contains(output, "[INFO]") {
				t.Error("Expected INFO label")
			}
			if !regexp.MustCompile(`\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}`).MatchString(output) {
				t.Error("Expected timestamp format")
			}
		})
	}
}

func TestDebug(t *testing.T) {
	tests := []string{
		"debug message",
		"",
		"debug with special chars: !@#$%^&*()",
		"multi\nline\ndebug",
	}

	for i, input := range tests {
		t.Run(fmt.Sprintf("test_%d", i), func(t *testing.T) {
			oldStdout := os.Stdout
			oldStderr := os.Stderr
			
			r, w, _ := os.Pipe()
			os.Stdout = w
			os.Stderr = w

			Debug(input)

			w.Close()
			os.Stdout = oldStdout
			os.Stderr = oldStderr

			var buf bytes.Buffer
			buf.ReadFrom(r)
			output := buf.String()

			if !regexp.MustCompile(`\[\d+\]`).MatchString(output) {
				t.Error("Expected goroutine count in brackets")
			}
			if !strings.Contains(output, "[DEBUG]") {
				t.Error("Expected DEBUG label")
			}
			if !regexp.MustCompile(`\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}`).MatchString(output) {
				t.Error("Expected timestamp format")
			}
			if !strings.Contains(output, input) {
				t.Errorf("Expected output to contain input %q", input)
			}
		})
	}
}

func TestPretty(t *testing.T) {
	tests := []struct {
		name      string
		variables []interface{}
	}{
		{
			name:      "single variable",
			variables: []interface{}{"test string"},
		},
		{
			name:      "multiple variables",
			variables: []interface{}{"string", 123, struct{ Name string }{"test"}},
		},
		{
			name:      "empty variables",
			variables: []interface{}{},
		},
		{
			name:      "nil variable",
			variables: []interface{}{nil},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldStdout := os.Stdout
			
			r, w, _ := os.Pipe()
			os.Stdout = w

			Pretty(tt.variables...)

			w.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			buf.ReadFrom(r)
			output := buf.String()

			// Check for separator lines
			separatorCount := strings.Count(output, "----------------------------------")
			if separatorCount < 2 {
				t.Error("Expected opening and closing separator lines")
			}
		})
	}
}