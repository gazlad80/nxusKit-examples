//go:build nxuskit

package llm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nxus-SYSTEMS/nxusKit/examples/apps/riffer"
)

const narrativeSystemPrompt = `You are a music analyst writing for a general audience. Given a technical music analysis, write a clear, engaging prose description that explains the musical characteristics in accessible terms.

Guidelines:
- Avoid overly technical jargon; explain musical concepts when necessary
- Be concise but informative (2-3 paragraphs)
- Highlight the most interesting or notable features
- Mention both strengths and areas that could be improved
- Use descriptive language that helps readers "hear" the music
- If the analysis suggests the music is simple (like a scale), acknowledge that while still being informative`

// GenerateNarrative generates a prose description of a music analysis.
// It uses an LLM to create accessible, engaging content for non-technical audiences.
func GenerateNarrative(ctx context.Context, analysis *riffer.AnalysisResult) (string, error) {
	analysisJSON, err := json.MarshalIndent(analysis, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to serialize analysis: %w", err)
	}

	userPrompt := fmt.Sprintf(
		"Please write a narrative description of this music analysis:\n\n%s",
		string(analysisJSON),
	)

	return callLLM(ctx, narrativeSystemPrompt, userPrompt)
}
