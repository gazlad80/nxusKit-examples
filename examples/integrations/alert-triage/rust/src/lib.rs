//! Alert Triage Pattern
//!
//! Demonstrates how to integrate LLMs with observability/monitoring systems
//! to triage alerts and generate actionable recommendations.

use nxuskit::{AsyncProvider, ChatRequest, ChatResponse, Message, NxuskitError, ResponseFormat};
use serde::{Deserialize, Serialize};

/// An alert from a monitoring system (Alertmanager format).
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Alert {
    /// Name of the alert rule
    pub alertname: String,
    /// Severity level
    pub severity: String,
    /// Instance that triggered the alert
    pub instance: String,
    /// Human-readable description
    pub description: String,
}

/// Result of triaging an alert.
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct TriageResult {
    /// Name of the alert (copied from input)
    pub alertname: String,
    /// Priority 1-5 (1 = highest)
    pub priority: u8,
    /// One-line assessment
    pub summary: String,
    /// Best guess at root cause
    pub likely_cause: String,
    /// Recommended actions to take
    pub suggested_actions: Vec<String>,
}

/// Error type for triage failures.
#[derive(Debug)]
pub enum TriageError {
    /// LLM request failed
    LlmError(NxuskitError),
    /// JSON parsing failed
    ParseError(String),
}

impl From<NxuskitError> for TriageError {
    fn from(e: NxuskitError) -> Self {
        TriageError::LlmError(e)
    }
}

impl From<serde_json::Error> for TriageError {
    fn from(e: serde_json::Error) -> Self {
        TriageError::ParseError(e.to_string())
    }
}

const SYSTEM_PROMPT: &str = r#"You are an SRE assistant. Triage the provided alerts and suggest actions.
Return a JSON array with one object per alert. Each object must have:
- alertname: string (copy from input)
- priority: number 1-5 (1 = highest/critical, 5 = lowest/informational)
- summary: string (one-line assessment)
- likely_cause: string (best guess at root cause)
- suggested_actions: array of strings (recommended actions)

Critical and high severity alerts should have priority 1-2."#;

/// Triages a batch of alerts using an LLM.
///
/// # Arguments
/// * `provider` - The LLM provider to use
/// * `model` - Model name to use
/// * `alerts` - The alerts to triage
///
/// # Returns
/// * `Ok(Vec<TriageResult>)` - Triage results for each alert
/// * `Err(TriageError)` - If triage fails
pub async fn triage_alerts(
    provider: &dyn AsyncProvider,
    model: &str,
    alerts: &[Alert],
) -> Result<Vec<TriageResult>, TriageError> {
    let alerts_json = serde_json::to_string_pretty(alerts)?;

    let mut request = ChatRequest::new(model)
        .with_message(Message::system(SYSTEM_PROMPT))
        .with_message(Message::user(format!(
            "Triage these alerts:\n{}",
            alerts_json
        )));

    // Enable JSON mode for structured output
    request.response_format = Some(ResponseFormat::Json);

    let response: ChatResponse = provider.chat(request).await?;

    let results: Vec<TriageResult> = serde_json::from_str(&response.content)?;
    Ok(results)
}

/// Parses triage results from JSON (useful for testing).
pub fn parse_triage_results(json: &str) -> Result<Vec<TriageResult>, TriageError> {
    let results: Vec<TriageResult> =
        serde_json::from_str(json).map_err(|e| TriageError::ParseError(e.to_string()))?;
    Ok(results)
}

#[cfg(test)]
mod tests {
    use super::*;
    use nxuskit::MockProvider;

    fn sample_alerts() -> Vec<Alert> {
        vec![
            Alert {
                alertname: "HighMemoryUsage".into(),
                severity: "warning".into(),
                instance: "web-server-01".into(),
                description: "Memory usage above 85% for 5 minutes".into(),
            },
            Alert {
                alertname: "PodCrashLooping".into(),
                severity: "critical".into(),
                instance: "api-deployment-xyz".into(),
                description: "Pod restarted 5 times in last 10 minutes".into(),
            },
        ]
    }

    #[test]
    fn test_triage_returns_priority_for_critical_alert() {
        let json = r#"[
            {"alertname": "PodCrashLooping", "priority": 1, "summary": "Critical pod failure", "likely_cause": "OOM or application crash", "suggested_actions": ["Check logs", "Scale up resources"]}
        ]"#;

        let results = parse_triage_results(json).unwrap();
        assert_eq!(results.len(), 1);
        assert_eq!(results[0].priority, 1);
        assert_eq!(results[0].alertname, "PodCrashLooping");
    }

    #[test]
    fn test_triage_batch_processes_multiple_alerts() {
        let json = r#"[
            {"alertname": "HighMemoryUsage", "priority": 3, "summary": "Memory pressure", "likely_cause": "Memory leak", "suggested_actions": ["Monitor"]},
            {"alertname": "PodCrashLooping", "priority": 1, "summary": "Pod failing", "likely_cause": "App crash", "suggested_actions": ["Investigate"]}
        ]"#;

        let results = parse_triage_results(json).unwrap();
        assert_eq!(results.len(), 2);
    }

    #[test]
    fn test_triage_includes_suggested_actions() {
        let json = r#"[
            {"alertname": "HighCPU", "priority": 2, "summary": "High CPU", "likely_cause": "Traffic spike", "suggested_actions": ["Scale horizontally", "Check for runaway processes", "Review recent deployments"]}
        ]"#;

        let results = parse_triage_results(json).unwrap();
        assert_eq!(results[0].suggested_actions.len(), 3);
        assert!(
            results[0]
                .suggested_actions
                .contains(&"Scale horizontally".to_string())
        );
    }

    #[tokio::test]
    async fn test_triage_alerts_with_mock_provider() {
        let mock_response = r#"[
            {"alertname": "HighMemoryUsage", "priority": 3, "summary": "Memory pressure on web-server-01", "likely_cause": "Memory leak or traffic spike", "suggested_actions": ["Check for memory leaks", "Consider scaling"]},
            {"alertname": "PodCrashLooping", "priority": 1, "summary": "Critical pod failure", "likely_cause": "Application crash or OOM", "suggested_actions": ["Check pod logs", "Review recent changes"]}
        ]"#;

        let mock = MockProvider::new(mock_response);

        let alerts = sample_alerts();
        let result = triage_alerts(&mock, "test-model", &alerts).await;
        assert!(result.is_ok());
        let results = result.unwrap();
        assert_eq!(results.len(), 2);
        assert_eq!(results[1].priority, 1); // Critical alert should have priority 1
    }
}
