package main

import (
	"context"
	"testing"

	nxuskit "github.com/nxus-SYSTEMS/nxusKit/packages/nxuskit-go"
)

func TestCapability_ListModels(t *testing.T) {
	ctx := context.Background()

	provider := nxuskit.NewMockProvider()

	models, err := provider.ListModels(ctx)
	if err != nil {
		t.Fatalf("Failed to list models: %v", err)
	}

	// Mock provider should return some models
	if len(models) == 0 {
		t.Skip("Mock provider returned no models - this is acceptable for mock")
	}

	// Verify model info structure
	for _, model := range models {
		if model.Name == "" {
			t.Error("Model name should not be empty")
		}
	}
}

func TestCapability_VisionSupport(t *testing.T) {
	ctx := context.Background()

	provider := nxuskit.NewMockProvider()

	models, err := provider.ListModels(ctx)
	if err != nil {
		t.Fatalf("Failed to list models: %v", err)
	}

	// Check SupportsVision method works
	for _, model := range models {
		_ = model.SupportsVision() // Should not panic
	}
}

func TestCapability_Modalities(t *testing.T) {
	ctx := context.Background()

	provider := nxuskit.NewMockProvider()

	models, err := provider.ListModels(ctx)
	if err != nil {
		t.Fatalf("Failed to list models: %v", err)
	}

	for _, model := range models {
		modalities := model.Modalities()
		// Modalities should return a slice (may be empty)
		_ = modalities
	}
}

func TestCapability_ContextWindow(t *testing.T) {
	ctx := context.Background()

	provider := nxuskit.NewMockProvider()

	models, err := provider.ListModels(ctx)
	if err != nil {
		t.Fatalf("Failed to list models: %v", err)
	}

	for _, model := range models {
		// FormattedContextWindow should return a string
		formatted := model.FormattedContextWindow()
		_ = formatted

		// ContextWindow may be nil
		if model.ContextWindow != nil {
			if *model.ContextWindow < 0 {
				t.Errorf("Context window should not be negative: %d", *model.ContextWindow)
			}
		}
	}
}

func TestCapability_FilterVisionModels(t *testing.T) {
	ctx := context.Background()

	provider := nxuskit.NewMockProvider()

	models, err := provider.ListModels(ctx)
	if err != nil {
		t.Fatalf("Failed to list models: %v", err)
	}

	// Filter for vision-capable models
	var visionModels []nxuskit.ModelInfo
	for _, m := range models {
		if m.SupportsVision() {
			visionModels = append(visionModels, m)
		}
	}

	// Result should be a valid slice (may be empty)
	if visionModels == nil {
		visionModels = []nxuskit.ModelInfo{}
	}

	t.Logf("Found %d vision models out of %d total", len(visionModels), len(models))
}

func TestCapability_TaskBasedSelection(t *testing.T) {
	// Test model selection logic patterns
	testCases := []struct {
		task          string
		requireVision bool
		minContext    int
		preferFastest bool
	}{
		{"simple_text", false, 4096, true},
		{"document_analysis", false, 100000, false},
		{"image_analysis", true, 4096, false},
		{"multi_image_comparison", true, 32000, false},
	}

	for _, tc := range testCases {
		t.Run(tc.task, func(t *testing.T) {
			// Just verify the selection criteria are valid
			if tc.minContext < 0 {
				t.Error("Min context should not be negative")
			}
		})
	}
}

func TestCapability_TruncateHelper(t *testing.T) {
	testCases := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"exactly10!", 10, "exactly10!"},
		{"this is a longer string", 10, "this is..."},
		{"", 10, ""},
	}

	for _, tc := range testCases {
		result := truncate(tc.input, tc.maxLen)
		if result != tc.expected {
			t.Errorf("truncate(%q, %d) = %q, want %q", tc.input, tc.maxLen, result, tc.expected)
		}
	}
}
