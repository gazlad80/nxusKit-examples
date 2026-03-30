//! Verbose output formatting for request/response debugging.

use crate::InteractiveConfig;
use serde::Serialize;

impl InteractiveConfig {
    /// Print a request in verbose format.
    ///
    /// Displays the HTTP method, URL, and JSON body with pretty-printing.
    /// Binary data (base64) is summarized rather than shown in full.
    pub fn print_request<T: Serialize>(&self, method: &str, url: &str, body: &T) {
        if !self.verbose {
            return;
        }

        eprintln!("\n[nxusKit REQUEST] {} {}", method, url);

        match serde_json::to_string_pretty(body) {
            Ok(json) => {
                let processed = self.process_json_for_display(&json);
                eprintln!("{}", processed);
            }
            Err(e) => {
                eprintln!("  (Could not serialize request: {})", e);
            }
        }
    }

    /// Print a response in verbose format.
    ///
    /// Displays the status code, elapsed time, and JSON body.
    pub fn print_response<T: Serialize>(&self, status: u16, elapsed_ms: u64, body: &T) {
        if !self.verbose {
            return;
        }

        let status_text = match status {
            200 => "OK",
            201 => "Created",
            400 => "Bad Request",
            401 => "Unauthorized",
            403 => "Forbidden",
            404 => "Not Found",
            429 => "Too Many Requests",
            500 => "Internal Server Error",
            _ => "",
        };

        eprintln!(
            "\n[nxusKit RESPONSE] {} {} ({}ms)",
            status, status_text, elapsed_ms
        );

        match serde_json::to_string_pretty(body) {
            Ok(json) => {
                let processed = self.process_json_for_display(&json);
                eprintln!("{}", processed);
            }
            Err(e) => {
                eprintln!("  (Could not serialize response: {})", e);
            }
        }
    }

    /// Print a streaming chunk.
    ///
    /// Displays chunk number and raw data for SSE debugging.
    pub fn print_stream_chunk(&self, chunk_num: usize, data: &str) {
        if !self.verbose {
            return;
        }

        // Truncate long chunks
        let display_data = if data.len() > 200 {
            format!("{}... [truncated, {} chars]", &data[..200], data.len())
        } else {
            data.to_string()
        };

        eprintln!("[nxusKit STREAM] chunk {}: {}", chunk_num, display_data);
    }

    /// Print stream completion summary.
    pub fn print_stream_done(&self, elapsed_ms: u64, total_chunks: usize) {
        if !self.verbose {
            return;
        }

        eprintln!(
            "[nxusKit STREAM] done ({}ms, {} chunks)",
            elapsed_ms, total_chunks
        );
    }

    /// Process JSON for display, handling truncation and base64 summarization.
    fn process_json_for_display(&self, json: &str) -> String {
        // Check for base64 data and summarize it
        let processed = summarize_base64(json);

        // Truncate if too long
        if processed.len() > self.verbose_limit {
            format!(
                "{}... [truncated, {} chars total]",
                &processed[..self.verbose_limit],
                processed.len()
            )
        } else {
            processed
        }
    }
}

/// Summarize base64 data in JSON strings.
///
/// Replaces long base64 strings with a summary like "[base64: 45.2KB image/png]".
fn summarize_base64(json: &str) -> String {
    // Simple heuristic: look for very long strings that look like base64
    // This is a simplified implementation - production code would use a proper JSON parser
    let mut result = json.to_string();

    // Pattern: strings longer than 1000 chars that are mostly alphanumeric with +/=
    let base64_pattern = regex_lite::Regex::new(r#""([A-Za-z0-9+/=]{1000,})""#).ok();

    if let Some(re) = base64_pattern {
        result = re
            .replace_all(&result, |caps: &regex_lite::Captures| {
                let len = caps[1].len();
                let kb = len as f64 / 1024.0;
                format!("\"[base64: {:.1}KB data]\"", kb)
            })
            .to_string();
    }

    result
}
