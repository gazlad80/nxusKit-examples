//! Narrative Analysis Module
//!
//! Uses LLMs to generate prose descriptions of music analysis results.

use std::env;

use nxuskit::builders::{ClaudeProvider, OllamaProvider, OpenAIProvider};
use nxuskit::{ChatRequest, Message};

use crate::engine::AnalysisResult;
use crate::errors::{Result, RifferError};

/// Generate a narrative description of a music analysis
///
/// # Arguments
/// * `analysis` - The structured analysis result
///
/// # Returns
/// A prose description suitable for non-technical audiences
pub async fn generate_narrative(analysis: &AnalysisResult) -> Result<String> {
    let system_prompt = r#"You are a music analyst writing for a general audience. Given a technical music analysis, write a clear, engaging prose description that explains the musical characteristics in accessible terms.

Guidelines:
- Avoid overly technical jargon; explain musical concepts when necessary
- Be concise but informative (2-3 paragraphs)
- Highlight the most interesting or notable features
- Mention both strengths and areas that could be improved
- Use descriptive language that helps readers "hear" the music
- If the analysis suggests the music is simple (like a scale), acknowledge that while still being informative"#;

    let analysis_json = serde_json::to_string_pretty(analysis)
        .map_err(|e| RifferError::LlmParseError(e.to_string()))?;

    let user_prompt = format!(
        "Please write a narrative description of this music analysis:\n\n{}",
        analysis_json
    );

    call_llm(system_prompt, &user_prompt).await
}

/// Call an LLM provider to get a response
async fn call_llm(system_prompt: &str, user_prompt: &str) -> Result<String> {
    // Try providers in order: Claude, OpenAI, Ollama
    if let Ok(api_key) = env::var("ANTHROPIC_API_KEY") {
        return call_claude(&api_key, system_prompt, user_prompt).await;
    }

    if let Ok(api_key) = env::var("OPENAI_API_KEY") {
        return call_openai(&api_key, system_prompt, user_prompt).await;
    }

    // Try Ollama as fallback (no API key needed)
    call_ollama(system_prompt, user_prompt).await
}

async fn call_claude(api_key: &str, system_prompt: &str, user_prompt: &str) -> Result<String> {
    let provider = ClaudeProvider::builder()
        .api_key(api_key)
        .model("claude-haiku-4-5-20251001")
        .build()
        .map_err(|e| RifferError::LlmUnavailable(e.to_string()))?;

    let request = ChatRequest::new("claude-haiku-4-5-20251001")
        .with_message(Message::system(system_prompt))
        .with_message(Message::user(user_prompt));

    let response = provider
        .chat_async(request)
        .await
        .map_err(|e| RifferError::LlmUnavailable(e.to_string()))?;

    Ok(response.content)
}

async fn call_openai(api_key: &str, system_prompt: &str, user_prompt: &str) -> Result<String> {
    let provider = OpenAIProvider::builder()
        .api_key(api_key)
        .model("gpt-4o-mini")
        .build()
        .map_err(|e| RifferError::LlmUnavailable(e.to_string()))?;

    let request = ChatRequest::new("gpt-4o-mini")
        .with_message(Message::system(system_prompt))
        .with_message(Message::user(user_prompt));

    let response = provider
        .chat_async(request)
        .await
        .map_err(|e| RifferError::LlmUnavailable(e.to_string()))?;

    Ok(response.content)
}

async fn call_ollama(system_prompt: &str, user_prompt: &str) -> Result<String> {
    let provider = OllamaProvider::builder()
        .model("llama3.2")
        .build()
        .map_err(|e| RifferError::LlmUnavailable(e.to_string()))?;

    let request = ChatRequest::new("llama3.2")
        .with_message(Message::system(system_prompt))
        .with_message(Message::user(user_prompt));

    let response = provider.chat_async(request).await.map_err(|e| {
        RifferError::LlmUnavailable(format!(
            "No LLM available. Set ANTHROPIC_API_KEY or OPENAI_API_KEY, or run Ollama locally. Error: {}",
            e
        ))
    })?;

    Ok(response.content)
}
