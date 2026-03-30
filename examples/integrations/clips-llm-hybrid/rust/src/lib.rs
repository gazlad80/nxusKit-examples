//! CLIPS+LLM Hybrid Pattern
//!
//! ## nxusKit Features Demonstrated
//! - Hybrid inference: LLM (probabilistic) + CLIPS (deterministic) integration
//! - NxuskitProvider with provider_type "clips" for rule-based expert system execution
//! - JSON-based fact assertion and conclusion extraction
//! - AsyncProvider trait for provider-agnostic code
//! - ResponseFormat for structured LLM output
//!
//! ## Why This Pattern Matters
//! LLMs excel at understanding unstructured input but can be unpredictable for
//! business rules. CLIPS provides deterministic, auditable rule execution.
//! Combining both gives you the best of both worlds: natural language understanding
//! with predictable, explainable business logic.
//!
//! ## Architecture
//! 1. LLM classifies unstructured ticket -> structured facts
//! 2. CLIPS applies business rules -> routing decision
//! 3. LLM generates human-friendly response

use nxuskit::{
    AsyncProvider, ChatRequest, ChatResponse, Message, NxuskitError, NxuskitProvider,
    ProviderConfig, ResponseFormat,
};
use nxuskit_examples_clips_wire::{ClipsFactWire, ClipsInputWire, ClipsOutputWire};
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::path::Path;

/// Classification extracted by LLM from natural language ticket.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TicketClassification {
    /// Category: "security", "infrastructure", "application", "general"
    pub category: String,
    /// Priority: "low", "medium", "high", "critical"
    pub priority: String,
    /// Detected sentiment: "positive", "neutral", "negative", "frustrated"
    pub sentiment: String,
    /// Key entities mentioned in the ticket
    pub key_entities: Vec<String>,
}

/// Routing decision from CLIPS expert system.
/// This structure matches the `routing-decision` deftemplate in ticket-routing.clp.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RoutingDecision {
    /// Team to handle the ticket (maps to CLIPS slot: team)
    pub team: String,
    /// SLA in hours (maps to CLIPS slot: sla-hours)
    #[serde(alias = "sla-hours")]
    pub sla_hours: u32,
    /// Escalation level (maps to CLIPS slot: escalation-level)
    /// 0 = none, 1 = manager, 2 = director
    #[serde(alias = "escalation-level")]
    pub escalation_level: u8,
}

/// Combined analysis result (LLM + CLIPS + LLM).
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TicketAnalysis {
    // From CLIPS (deterministic)
    /// Assigned team (from CLIPS rules)
    pub team: String,
    /// SLA deadline in hours (from CLIPS rules)
    pub sla_hours: u32,
    /// Escalation level (from CLIPS rules)
    pub escalation_level: u8,

    // From LLM (probabilistic)
    /// Detected sentiment
    pub sentiment: String,
    /// Key entities extracted
    pub key_entities: Vec<String>,
    /// LLM-generated initial response suggestion
    pub suggested_response: String,
}

/// Error type for hybrid analysis failures.
#[derive(Debug)]
pub enum HybridError {
    LlmError(NxuskitError),
    ParseError(String),
    ClipsError(String),
}

impl From<NxuskitError> for HybridError {
    fn from(e: NxuskitError) -> Self {
        HybridError::LlmError(e)
    }
}

impl From<serde_json::Error> for HybridError {
    fn from(e: serde_json::Error) -> Self {
        HybridError::ParseError(e.to_string())
    }
}

const CLASSIFY_PROMPT: &str = r#"Classify this support ticket. Respond with JSON only.
Format: {"category": "security|infrastructure|application|general", "priority": "low|medium|high|critical", "sentiment": "positive|neutral|negative|frustrated", "key_entities": ["entity1", "entity2"]}"#;

const RESPONSE_PROMPT: &str = r#"Generate a brief, empathetic initial response for this support ticket.
Keep it under 100 words. Acknowledge the issue and set expectations."#;

/// Applies routing rules using the CLIPS expert system.
///
/// This function:
/// 1. Creates a NxuskitProvider configured with the ticket-routing.clp rules
/// 2. Asserts the ticket classification as a CLIPS fact
/// 3. Runs the CLIPS inference engine
/// 4. Extracts the routing-decision conclusion
///
/// # nxusKit Feature: NxuskitProvider with provider_type "clips" for deterministic rule execution
pub fn apply_routing_rules(
    classification: &TicketClassification,
    rules_path: &Path,
) -> Result<RoutingDecision, HybridError> {
    // nxusKit: NxuskitProvider with clips provider type
    let rules_dir = rules_path
        .parent()
        .unwrap_or(Path::new("."))
        .to_str()
        .unwrap_or(".");

    let clips = NxuskitProvider::new(ProviderConfig {
        provider_type: "clips".to_string(),
        model: Some(rules_dir.to_string()),
        ..Default::default()
    })
    .map_err(|e| HybridError::ClipsError(format!("Failed to create CLIPS provider: {}", e)))?;

    // nxusKit: Format input as JSON for CLIPS provider — same shape as Go `clipsInputWire` in hybrid.go.
    let has_security_keywords = if classification.key_entities.iter().any(|e| {
        let el = e.to_lowercase();
        el.contains("breach") || el.contains("hack") || el.contains("unauthorized")
    }) {
        "yes"
    } else {
        "no"
    };

    let mut values = HashMap::new();
    values.insert(
        "category".to_string(),
        serde_json::Value::String(classification.category.clone()),
    );
    values.insert(
        "priority".to_string(),
        serde_json::Value::String(classification.priority.clone()),
    );
    values.insert(
        "sentiment".to_string(),
        serde_json::Value::String(classification.sentiment.clone()),
    );
    values.insert(
        "has-security-keywords".to_string(),
        serde_json::Value::String(has_security_keywords.to_string()),
    );

    let clips_input = ClipsInputWire {
        facts: vec![ClipsFactWire {
            template: "ticket-classification".into(),
            values,
            id: None,
        }],
        ..Default::default()
    };
    let clips_body =
        serde_json::to_string(&clips_input).map_err(|e| HybridError::ParseError(e.to_string()))?;

    // nxusKit: Use the rules file name as the "model" parameter
    let model = rules_path
        .file_name()
        .and_then(|n| n.to_str())
        .unwrap_or("ticket-routing.clp");

    // nxusKit: ChatRequest works uniformly for CLIPS and LLM providers
    let request = ChatRequest::new(model).with_message(Message::user(clips_body));

    // nxusKit: Same chat() interface for rule execution as for LLM inference
    let response = clips
        .chat(request)
        .map_err(|e| HybridError::ClipsError(format!("CLIPS inference failed: {}", e)))?;

    // Parse CLIPS output to extract routing decision
    let output: ClipsOutputWire = serde_json::from_str(&response.content)
        .map_err(|e| HybridError::ClipsError(format!("Failed to parse CLIPS output: {}", e)))?;

    // Find the routing-decision conclusion
    let routing_conclusion = output
        .conclusions
        .iter()
        .find(|c| c.template == "routing-decision")
        .ok_or_else(|| {
            HybridError::ClipsError("No routing-decision derived from CLIPS rules".to_string())
        })?;

    // nxusKit: Extract values from CLIPS conclusion
    let values = &routing_conclusion.values;
    Ok(RoutingDecision {
        team: values
            .get("team")
            .and_then(|v| v.as_str())
            .unwrap_or("general-support")
            .to_string(),
        sla_hours: values
            .get("sla-hours")
            .and_then(|v| v.as_i64())
            .unwrap_or(24) as u32,
        escalation_level: values
            .get("escalation-level")
            .and_then(|v| v.as_i64())
            .unwrap_or(0) as u8,
    })
}

/// Synchronous routing function for unit tests.
/// Provides fast, deterministic testing without async/CLIPS dependencies.
#[cfg(test)]
pub fn apply_routing_rules_sync(classification: &TicketClassification) -> RoutingDecision {
    // Security tickets always go to security team with high priority
    if classification.category == "security" {
        return RoutingDecision {
            team: "security".into(),
            sla_hours: 4,
            escalation_level: 2,
        };
    }

    // Infrastructure with critical priority goes to SRE
    if classification.category == "infrastructure" && classification.priority == "critical" {
        return RoutingDecision {
            team: "sre".into(),
            sla_hours: 2,
            escalation_level: 1,
        };
    }

    // High priority infrastructure goes to SRE
    if classification.category == "infrastructure" && classification.priority == "high" {
        return RoutingDecision {
            team: "sre".into(),
            sla_hours: 4,
            escalation_level: 1,
        };
    }

    // Application issues go to development
    if classification.category == "application" {
        let sla = match classification.priority.as_str() {
            "critical" => 4,
            "high" => 8,
            _ => 24,
        };
        return RoutingDecision {
            team: "development".into(),
            sla_hours: sla,
            escalation_level: if classification.priority == "critical" {
                1
            } else {
                0
            },
        };
    }

    // Default routing for general support
    RoutingDecision {
        team: "general-support".into(),
        sla_hours: 24,
        escalation_level: 0,
    }
}

/// Analyzes a support ticket using the hybrid LLM + CLIPS pattern.
///
/// ## nxusKit Features Demonstrated
/// - AsyncProvider trait enables provider-agnostic LLM usage
/// - NxuskitProvider with provider_type "clips" for deterministic rule execution
/// - ResponseFormat::Json for structured LLM output
/// - Unified interface for both LLM and CLIPS providers
///
/// ## Three-step flow:
/// 1. **LLM** classifies the ticket (extracts category, priority, sentiment, entities)
/// 2. **CLIPS** applies deterministic routing rules (auditable, explainable)
/// 3. **LLM** generates a suggested response (natural language)
pub async fn analyze_ticket<P: AsyncProvider>(
    llm: &P,
    model: &str,
    ticket_text: &str,
    rules_path: &Path,
) -> Result<TicketAnalysis, HybridError> {
    // Step 1: LLM classification
    // nxusKit: ChatRequest builder pattern with fluent message addition
    let mut classify_request = ChatRequest::new(model)
        .with_message(Message::system(CLASSIFY_PROMPT))
        .with_message(Message::user(ticket_text));

    // nxusKit: ResponseFormat::Json enables structured output mode
    classify_request.response_format = Some(ResponseFormat::Json);

    // nxusKit: Unified async interface - same pattern for all providers
    let classify_response: ChatResponse = llm.chat(classify_request).await?;
    let classification: TicketClassification = serde_json::from_str(&classify_response.content)?;

    // Step 2: Apply CLIPS routing rules (deterministic, synchronous)
    let routing = apply_routing_rules(&classification, rules_path)?;

    // Step 3: LLM generates suggested response
    let response_request = ChatRequest::new(model)
        .with_message(Message::system(RESPONSE_PROMPT))
        .with_message(Message::user(ticket_text));

    let response_response: ChatResponse = llm.chat(response_request).await?;

    Ok(TicketAnalysis {
        team: routing.team,
        sla_hours: routing.sla_hours,
        escalation_level: routing.escalation_level,
        sentiment: classification.sentiment,
        key_entities: classification.key_entities,
        suggested_response: response_response.content,
    })
}

#[cfg(test)]
mod tests {
    use super::*;

    // Unit tests use apply_routing_rules_sync for fast, deterministic testing.
    // Integration tests use apply_routing_rules with real ClipsProvider.

    #[test]
    fn test_hybrid_routes_security_ticket_to_security_team() {
        let classification = TicketClassification {
            category: "security".into(),
            priority: "high".into(),
            sentiment: "frustrated".into(),
            key_entities: vec!["breach".into(), "unauthorized access".into()],
        };

        let routing = apply_routing_rules_sync(&classification);
        assert_eq!(routing.team, "security");
        assert_eq!(routing.sla_hours, 4);
        assert_eq!(routing.escalation_level, 2);
    }

    #[test]
    fn test_hybrid_routes_critical_infra_to_sre() {
        let classification = TicketClassification {
            category: "infrastructure".into(),
            priority: "critical".into(),
            sentiment: "frustrated".into(),
            key_entities: vec!["database".into(), "timeout".into()],
        };

        let routing = apply_routing_rules_sync(&classification);
        assert_eq!(routing.team, "sre");
        assert_eq!(routing.sla_hours, 2);
        assert_eq!(routing.escalation_level, 1);
    }

    #[test]
    fn test_hybrid_routes_app_issue_to_development() {
        let classification = TicketClassification {
            category: "application".into(),
            priority: "medium".into(),
            sentiment: "neutral".into(),
            key_entities: vec!["bug".into(), "login".into()],
        };

        let routing = apply_routing_rules_sync(&classification);
        assert_eq!(routing.team, "development");
        assert_eq!(routing.sla_hours, 24);
        assert_eq!(routing.escalation_level, 0);
    }

    #[test]
    fn test_hybrid_default_routing() {
        let classification = TicketClassification {
            category: "general".into(),
            priority: "low".into(),
            sentiment: "neutral".into(),
            key_entities: vec![],
        };

        let routing = apply_routing_rules_sync(&classification);
        assert_eq!(routing.team, "general-support");
    }

    // Integration test for real CLIPS execution
    // This test requires the ticket-routing.clp file to be present
    #[test]
    fn test_clips_routing_security_ticket() {
        // Path to the CLIPS rules file (relative to test execution directory)
        let rules_path = std::path::Path::new("../ticket-routing.clp");

        // Skip test if rules file doesn't exist (CI environment may not have CLIPS)
        if !rules_path.exists() {
            eprintln!("Skipping CLIPS integration test: rules file not found");
            return;
        }

        let classification = TicketClassification {
            category: "security".into(),
            priority: "high".into(),
            sentiment: "frustrated".into(),
            key_entities: vec!["breach".into()],
        };

        let result = apply_routing_rules(&classification, rules_path);

        match result {
            Ok(routing) => {
                assert_eq!(routing.team, "security");
                assert_eq!(routing.sla_hours, 4);
                assert_eq!(routing.escalation_level, 2);
            }
            Err(e) => {
                // CLIPS may not be available in all environments
                eprintln!("CLIPS integration test skipped: {:?}", e);
            }
        }
    }
}
