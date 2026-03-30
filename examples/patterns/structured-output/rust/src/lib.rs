//! Structured Output (JSON Mode) Pattern
//!
//! Demonstrates how to extract typed structured data from LLM responses
//! using JSON mode.

use nxuskit::{AsyncProvider, ChatRequest, ChatResponse, Message, NxuskitError, ResponseFormat};
use serde::{Deserialize, Serialize};

/// Classification result for a log entry.
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct LogClassification {
    /// Severity level: "info", "warning", "error", "critical"
    pub severity: String,
    /// Category: "auth", "network", "system", "application"
    pub category: String,
    /// One-line summary of the log entry
    pub summary: String,
    /// Whether immediate action is required
    pub actionable: bool,
}

impl LogClassification {
    /// Validates the classification fields.
    pub fn validate(&self) -> Result<(), String> {
        let valid_severities = ["info", "warning", "error", "critical"];
        if !valid_severities.contains(&self.severity.as_str()) {
            return Err(format!("Invalid severity: {}", self.severity));
        }

        let valid_categories = ["auth", "network", "system", "application"];
        if !valid_categories.contains(&self.category.as_str()) {
            return Err(format!("Invalid category: {}", self.category));
        }

        Ok(())
    }
}

/// Error type for classification failures.
#[derive(Debug)]
pub enum ClassificationError {
    /// LLM request failed
    LlmError(NxuskitError),
    /// JSON parsing failed
    ParseError(String),
    /// Validation failed
    ValidationError(String),
}

impl From<NxuskitError> for ClassificationError {
    fn from(e: NxuskitError) -> Self {
        ClassificationError::LlmError(e)
    }
}

impl From<serde_json::Error> for ClassificationError {
    fn from(e: serde_json::Error) -> Self {
        ClassificationError::ParseError(e.to_string())
    }
}

const SYSTEM_PROMPT: &str = r#"You are a log classifier. Analyze the log entry and respond with JSON only.
Format: {"severity": "info|warning|error|critical", "category": "auth|network|system|application", "summary": "one-line summary", "actionable": true|false}"#;

/// Classifies a log entry using an LLM with JSON mode.
///
/// # Arguments
/// * `provider` - The LLM provider to use
/// * `model` - Model name to use
/// * `log_entry` - The log entry text to classify
///
/// # Returns
/// * `Ok(LogClassification)` - The parsed classification
/// * `Err(ClassificationError)` - If classification fails
pub async fn classify_log<P: AsyncProvider>(
    provider: &P,
    model: &str,
    log_entry: &str,
) -> Result<LogClassification, ClassificationError> {
    let mut request = ChatRequest::new(model)
        .with_message(Message::system(SYSTEM_PROMPT))
        .with_message(Message::user(log_entry));

    // Enable JSON mode for structured output
    request.response_format = Some(ResponseFormat::Json);

    let response: ChatResponse = provider.chat(request).await?;

    let classification: LogClassification = serde_json::from_str(&response.content)?;

    classification
        .validate()
        .map_err(ClassificationError::ValidationError)?;

    Ok(classification)
}

/// Parses a JSON string into a LogClassification.
/// Useful for testing with mock responses.
pub fn parse_classification(json: &str) -> Result<LogClassification, ClassificationError> {
    let classification: LogClassification =
        serde_json::from_str(json).map_err(|e| ClassificationError::ParseError(e.to_string()))?;
    classification
        .validate()
        .map_err(ClassificationError::ValidationError)?;
    Ok(classification)
}

#[cfg(test)]
mod tests {
    use super::*;
    use nxuskit::MockProvider;

    #[test]
    fn test_classify_log_parses_valid_json() {
        let json = r#"{"severity": "error", "category": "auth", "summary": "Failed login attempt", "actionable": true}"#;
        let result = parse_classification(json);
        assert!(result.is_ok());
        let classification = result.unwrap();
        assert_eq!(classification.severity, "error");
        assert_eq!(classification.category, "auth");
        assert!(classification.actionable);
    }

    #[test]
    fn test_classify_log_handles_malformed_json() {
        let json = r#"{"severity": "error", incomplete"#;
        let result = parse_classification(json);
        assert!(matches!(result, Err(ClassificationError::ParseError(_))));
    }

    #[test]
    fn test_classification_validates_severity() {
        let json = r#"{"severity": "invalid", "category": "auth", "summary": "Test", "actionable": false}"#;
        let result = parse_classification(json);
        assert!(matches!(
            result,
            Err(ClassificationError::ValidationError(_))
        ));
    }

    #[tokio::test]
    async fn test_classify_log_with_mock_provider() {
        let mock_response = r#"{"severity": "warning", "category": "network", "summary": "Connection timeout", "actionable": true}"#;

        let mock = MockProvider::new(mock_response);

        let result = classify_log(&mock, "test-model", "Connection timed out after 30s").await;
        assert!(result.is_ok());
        let classification = result.unwrap();
        assert_eq!(classification.severity, "warning");
        assert_eq!(classification.category, "network");
    }
}
