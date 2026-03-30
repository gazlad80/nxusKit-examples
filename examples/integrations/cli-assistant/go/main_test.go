// Package main tests for the CLI assistant example.
package main

import (
	"context"
	"strings"
	"testing"

	"github.com/nxus-SYSTEMS/nxusKit/packages/nxuskit-go"
)

func TestGenerateCommand_StreamsOutput(t *testing.T) {
	ctx := context.Background()

	// Mock provider that streams a shell command
	provider := nxuskit.NewMockProvider(
		nxuskit.WithMockStreamResponse([]nxuskit.StreamChunk{
			{Delta: "find . "},
			{Delta: "-name "},
			{Delta: "\"*.rs\" "},
			{Delta: "-mtime -7"},
		}),
	)

	command, err := GenerateCommand(ctx, provider, "find rust files modified in the last week")
	if err != nil {
		t.Fatalf("Expected success, got error: %v", err)
	}

	if !strings.Contains(command, "find") {
		t.Errorf("Expected command to contain 'find', got '%s'", command)
	}

	if !strings.Contains(command, ".rs") {
		t.Errorf("Expected command to reference .rs files, got '%s'", command)
	}
}

func TestGenerateCommand_HandlesError(t *testing.T) {
	ctx := context.Background()

	// Default mock provider returns default stream response
	provider := nxuskit.NewMockProvider()

	// This should succeed with the default mock
	command, err := GenerateCommand(ctx, provider, "list files")
	if err != nil {
		t.Fatalf("Expected success, got error: %v", err)
	}

	if command == "" {
		t.Error("Expected non-empty command")
	}
}

func TestSystemPrompt_ContainsRules(t *testing.T) {
	if !strings.Contains(systemPrompt, "CLI assistant") {
		t.Error("System prompt should mention CLI assistant")
	}

	if !strings.Contains(systemPrompt, "ONLY the command") {
		t.Error("System prompt should instruct to output only the command")
	}

	if !strings.Contains(systemPrompt, "dangerous") {
		t.Error("System prompt should mention dangerous operations")
	}
}
