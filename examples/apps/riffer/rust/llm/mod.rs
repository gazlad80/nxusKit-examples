//! LLM Integration Module for Riffer
//!
//! Provides LLM-powered features:
//! - Natural language transformation prompts
//! - Narrative analysis of music sequences

pub mod narrative;
pub mod transform;

pub use narrative::generate_narrative;
pub use transform::transform_with_prompt;
