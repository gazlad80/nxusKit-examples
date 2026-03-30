//! Multi-Provider Fallback Pattern
//!
//! Demonstrates how to implement provider resilience with automatic fallback
//! through multiple LLM providers when one fails.

use nxuskit::{AsyncProvider, ChatRequest, ChatResponse, NxuskitError};

/// Attempts to send a chat request through multiple providers in sequence.
///
/// Returns the response from the first provider that succeeds, or an error
/// if all providers fail.
///
/// # Arguments
/// * `providers` - Slice of providers to try in order
/// * `request` - The chat request to send
///
/// # Returns
/// * `Ok(ChatResponse)` - Response from the first successful provider
/// * `Err(NxuskitError)` - Error if all providers fail
pub async fn chat_with_fallback(
    providers: &[Box<dyn AsyncProvider>],
    request: &ChatRequest,
) -> Result<ChatResponse, NxuskitError> {
    let mut last_error = None;

    for (i, provider) in providers.iter().enumerate() {
        match provider.chat(request.clone()).await {
            Ok(response) => {
                println!("Provider {} succeeded", i + 1);
                return Ok(response);
            }
            Err(e) => {
                eprintln!("Provider {} failed: {}", i + 1, e);
                last_error = Some(e);
            }
        }
    }

    Err(last_error.unwrap_or_else(|| NxuskitError::Provider {
        message: "No providers available".into(),
        provider: None,
    }))
}

#[cfg(test)]
mod tests {
    use super::*;
    use nxuskit::{Message, MockProvider};

    fn create_test_request() -> ChatRequest {
        ChatRequest::new("test-model").with_message(Message::user("Hello"))
    }

    #[tokio::test]
    async fn test_first_provider_succeeds() {
        let provider1 = MockProvider::new("First response");
        let provider2 = MockProvider::new("Second response");

        let providers: Vec<Box<dyn AsyncProvider>> = vec![Box::new(provider1), Box::new(provider2)];

        let result = chat_with_fallback(&providers, &create_test_request()).await;
        assert!(result.is_ok());
        assert_eq!(result.unwrap().content, "First response");
    }

    #[tokio::test]
    async fn test_no_providers_returns_error() {
        let providers: Vec<Box<dyn AsyncProvider>> = vec![];

        let result = chat_with_fallback(&providers, &create_test_request()).await;
        assert!(result.is_err());
    }
}
