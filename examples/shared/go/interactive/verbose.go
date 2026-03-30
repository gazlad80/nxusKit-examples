package interactive

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// PrintRequest prints a request in verbose format.
//
// Displays the HTTP method, URL, and JSON body with pretty-printing.
// Binary data (base64) is summarized rather than shown in full.
func (c *Config) PrintRequest(method, url string, body interface{}) {
	if !c.Verbose {
		return
	}

	fmt.Fprintf(os.Stderr, "\n[nxusKit REQUEST] %s %s\n", method, url)

	jsonBytes, err := json.MarshalIndent(body, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "  (Could not serialize request: %v)\n", err)
		return
	}

	processed := c.processJSONForDisplay(string(jsonBytes))
	fmt.Fprintln(os.Stderr, processed)
}

// PrintResponse prints a response in verbose format.
//
// Displays the status code, elapsed time, and JSON body.
func (c *Config) PrintResponse(status int, elapsedMs int64, body interface{}) {
	if !c.Verbose {
		return
	}

	statusText := getStatusText(status)
	fmt.Fprintf(os.Stderr, "\n[nxusKit RESPONSE] %d %s (%dms)\n", status, statusText, elapsedMs)

	jsonBytes, err := json.MarshalIndent(body, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "  (Could not serialize response: %v)\n", err)
		return
	}

	processed := c.processJSONForDisplay(string(jsonBytes))
	fmt.Fprintln(os.Stderr, processed)
}

// PrintStreamChunk prints a streaming chunk.
//
// Displays chunk number and raw data for SSE debugging.
func (c *Config) PrintStreamChunk(chunkNum int, data string) {
	if !c.Verbose {
		return
	}

	// Truncate long chunks
	displayData := data
	if len(data) > 200 {
		displayData = fmt.Sprintf("%s... [truncated, %d chars]", data[:200], len(data))
	}

	fmt.Fprintf(os.Stderr, "[nxusKit STREAM] chunk %d: %s\n", chunkNum, displayData)
}

// PrintStreamDone prints stream completion summary.
func (c *Config) PrintStreamDone(elapsedMs int64, totalChunks int) {
	if !c.Verbose {
		return
	}

	fmt.Fprintf(os.Stderr, "[nxusKit STREAM] done (%dms, %d chunks)\n", elapsedMs, totalChunks)
}

// processJSONForDisplay handles truncation and base64 summarization.
func (c *Config) processJSONForDisplay(jsonStr string) string {
	// Check for base64 data and summarize it
	processed := summarizeBase64(jsonStr)

	// Truncate if too long
	if len(processed) > c.VerboseLimit {
		return fmt.Sprintf("%s... [truncated, %d chars total]", processed[:c.VerboseLimit], len(processed))
	}
	return processed
}

// summarizeBase64 replaces long base64 strings with a summary.
func summarizeBase64(jsonStr string) string {
	// Pattern: strings longer than 1000 chars that are mostly alphanumeric with +/=
	re := regexp.MustCompile(`"([A-Za-z0-9+/=]{1000,})"`)

	return re.ReplaceAllStringFunc(jsonStr, func(match string) string {
		// Remove quotes to get actual length
		content := strings.Trim(match, `"`)
		kb := float64(len(content)) / 1024.0
		return fmt.Sprintf(`"[base64: %.1fKB data]"`, kb)
	})
}

// getStatusText returns a human-readable status text for common HTTP codes.
func getStatusText(status int) string {
	switch status {
	case 200:
		return "OK"
	case 201:
		return "Created"
	case 400:
		return "Bad Request"
	case 401:
		return "Unauthorized"
	case 403:
		return "Forbidden"
	case 404:
		return "Not Found"
	case 429:
		return "Too Many Requests"
	case 500:
		return "Internal Server Error"
	default:
		return ""
	}
}
