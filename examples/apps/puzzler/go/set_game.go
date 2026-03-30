// Package puzzler provides Set Game types.
//
// Set Game solver functions removed — real solving now uses nxusKit ClipsSession
// in cmd/main.go. See internal/reference/puzzler-pure-go-set-game.go.
package puzzler

// SetGameResult holds the result of finding sets in a hand.
type SetGameResult struct {
	// Sets found.
	Sets []ValidSet `json:"sets"`
	// FoundAny indicates whether at least one set was found.
	FoundAny bool `json:"found_any"`
	// Iterations is the number of rule firings/checks.
	Iterations int64 `json:"iterations"`
	// LLMCalls is the number of LLM API calls.
	LLMCalls int64 `json:"llm_calls"`
	// TokensUsed is the total tokens consumed.
	TokensUsed int64 `json:"tokens_used"`
}
