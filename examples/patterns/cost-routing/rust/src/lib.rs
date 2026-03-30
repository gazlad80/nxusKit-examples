//! Model Router (Cost Tiers) Pattern
//!
//! Demonstrates how to route requests to different models based on
//! task complexity to optimize costs.

use nxuskit::{AsyncProvider, ChatRequest, ChatResponse, Message, NxuskitError};

/// Cost tier for model selection.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum CostTier {
    /// Fast, cheap models for simple tasks
    Economy,
    /// Balanced cost/quality for general tasks
    Standard,
    /// High-quality models for complex reasoning
    Premium,
}

impl CostTier {
    /// Returns the recommended model name for this tier.
    pub fn model_name(&self) -> &'static str {
        match self {
            CostTier::Economy => "gpt-4o-mini",
            CostTier::Standard => "gpt-4o",
            CostTier::Premium => "gpt-4-turbo",
        }
    }

    /// Returns the tier name as a string.
    pub fn name(&self) -> &'static str {
        match self {
            CostTier::Economy => "Economy",
            CostTier::Standard => "Standard",
            CostTier::Premium => "Premium",
        }
    }
}

/// Classifies a task based on prompt characteristics.
///
/// Uses simple heuristics:
/// - Premium: Contains "analyze", "compare", or > 1000 chars
/// - Standard: > 200 chars
/// - Economy: Simple/short prompts
///
/// # Arguments
/// * `prompt` - The user's prompt text
///
/// # Returns
/// The appropriate cost tier for the task
pub fn classify_task(prompt: &str) -> CostTier {
    let prompt_lower = prompt.to_lowercase();

    // Premium tier indicators
    let complex_keywords = ["analyze", "compare", "evaluate", "synthesize", "critique"];
    if prompt.len() > 1000 || complex_keywords.iter().any(|k| prompt_lower.contains(k)) {
        return CostTier::Premium;
    }

    // Standard tier for medium complexity
    if prompt.len() > 200 {
        return CostTier::Standard;
    }

    // Economy for simple tasks
    CostTier::Economy
}

/// Sends a chat request using the appropriate model based on task complexity.
///
/// # Arguments
/// * `provider` - The LLM provider to use
/// * `prompt` - The user's prompt
///
/// # Returns
/// * `Ok((ChatResponse, CostTier))` - The response and tier used
/// * `Err(NxuskitError)` - If the request fails
pub async fn routed_chat(
    provider: &dyn AsyncProvider,
    prompt: &str,
) -> Result<(ChatResponse, CostTier), NxuskitError> {
    let tier = classify_task(prompt);
    let model = tier.model_name();

    println!("Task classified as: {} (using {})", tier.name(), model);

    let request = ChatRequest::new(model).with_message(Message::user(prompt));

    let response = provider.chat(request).await?;
    Ok((response, tier))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_short_prompt_selects_economy_tier() {
        let prompt = "What is 2+2?";
        let tier = classify_task(prompt);
        assert_eq!(tier, CostTier::Economy);
        assert_eq!(tier.model_name(), "gpt-4o-mini");
    }

    #[test]
    fn test_complex_prompt_selects_premium_tier() {
        let prompt = "Please analyze the following data and compare the trends across all regions.";
        let tier = classify_task(prompt);
        assert_eq!(tier, CostTier::Premium);
        assert_eq!(tier.model_name(), "gpt-4-turbo");
    }

    #[test]
    fn test_medium_prompt_selects_standard_tier() {
        // Create a prompt that's deterministically > 200 chars but ≤ 1000 chars,
        // without any premium keywords (analyze, compare, evaluate, synthesize, critique).
        let prompt = "Please help me write a function. ".repeat(7); // 7 × 32 = 224 chars
        assert!(prompt.len() > 200 && prompt.len() <= 1000);
        let tier = classify_task(&prompt);
        assert_eq!(tier, CostTier::Standard);
        assert_eq!(tier.model_name(), "gpt-4o");
    }

    #[test]
    fn test_long_prompt_selects_premium_tier() {
        // Create a very long prompt (> 1000 chars)
        let prompt = "x".repeat(1001);
        let tier = classify_task(&prompt);
        assert_eq!(tier, CostTier::Premium);
    }

    #[test]
    fn test_evaluate_keyword_selects_premium() {
        let prompt = "Evaluate this solution";
        let tier = classify_task(prompt);
        assert_eq!(tier, CostTier::Premium);
    }
}
