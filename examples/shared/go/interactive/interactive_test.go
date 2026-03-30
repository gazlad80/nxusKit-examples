package interactive

import (
	"testing"
)

func TestConfigDefault(t *testing.T) {
	config := New(false, false)
	if config.IsVerbose() {
		t.Error("Expected verbose to be false by default")
	}
	// Step mode depends on TTY, may be false even if not requested
	if config.GetVerboseLimit() != 2000 {
		t.Errorf("Expected verbose limit to be 2000, got %d", config.GetVerboseLimit())
	}
}

func TestConfigVerboseEnabled(t *testing.T) {
	config := New(true, false)
	if !config.IsVerbose() {
		t.Error("Expected verbose to be true when enabled")
	}
}

func TestStepActionConstants(t *testing.T) {
	if ActionContinue == ActionQuit {
		t.Error("ActionContinue should not equal ActionQuit")
	}
	if ActionQuit == ActionSkip {
		t.Error("ActionQuit should not equal ActionSkip")
	}
	if ActionContinue == ActionSkip {
		t.Error("ActionContinue should not equal ActionSkip")
	}
}

func TestVerboseDoesNothingWhenDisabled(t *testing.T) {
	config := New(false, false)
	// These should not panic when verbose is disabled
	config.PrintRequest("GET", "http://test", "test")
	config.PrintResponse(200, 100, "test")
	config.PrintStreamChunk(1, "data")
	config.PrintStreamDone(100, 5)
}

func TestSummarizeBase64(t *testing.T) {
	// Create a fake base64 string (1500+ chars)
	longBase64 := `"` + string(make([]byte, 1500)) + `"`
	// This won't actually match the regex since it's null bytes,
	// but tests the function doesn't panic
	_ = summarizeBase64(longBase64)
}

func TestGetStatusText(t *testing.T) {
	tests := []struct {
		status int
		want   string
	}{
		{200, "OK"},
		{201, "Created"},
		{400, "Bad Request"},
		{401, "Unauthorized"},
		{403, "Forbidden"},
		{404, "Not Found"},
		{429, "Too Many Requests"},
		{500, "Internal Server Error"},
		{999, ""},
	}

	for _, tt := range tests {
		got := getStatusText(tt.status)
		if got != tt.want {
			t.Errorf("getStatusText(%d) = %q, want %q", tt.status, got, tt.want)
		}
	}
}

func TestClamp(t *testing.T) {
	tests := []struct {
		value, min, max, want int
	}{
		{50, 0, 100, 50},
		{-10, 0, 100, 0},
		{150, 0, 100, 100},
		{0, 0, 100, 0},
		{100, 0, 100, 100},
	}

	for _, tt := range tests {
		got := clamp(tt.value, tt.min, tt.max)
		if got != tt.want {
			t.Errorf("clamp(%d, %d, %d) = %d, want %d", tt.value, tt.min, tt.max, got, tt.want)
		}
	}
}

func TestSkipSteps(t *testing.T) {
	config := New(false, true)
	// Note: Step mode might be disabled due to non-TTY environment
	// So we test the skip mechanism directly
	config.Step = true // Force enable for test
	config.stepSkipped = false

	if config.stepSkipped {
		t.Error("stepSkipped should be false initially")
	}

	config.SkipSteps()

	if !config.stepSkipped {
		t.Error("stepSkipped should be true after SkipSteps()")
	}
}
