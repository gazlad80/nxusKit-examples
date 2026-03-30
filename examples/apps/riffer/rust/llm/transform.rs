//! Natural Language Music Transformation
//!
//! Uses LLMs to interpret natural language prompts and generate
//! transformation instructions for music sequences.

use std::env;

use nxuskit::builders::{ClaudeProvider, OllamaProvider, OpenAIProvider};
use nxuskit::{ChatRequest, Message};

use crate::engine::{augment, change_tempo, diminish, invert, key_change, retrograde, transpose};
use crate::errors::{Result, RifferError};
use crate::types::{KeySignature, Mode, PitchClass, Sequence};

/// Transform a sequence based on a natural language prompt
///
/// # Arguments
/// * `sequence` - The music sequence to transform (modified in place)
/// * `prompt` - Natural language description of the desired transformation
///
/// # Example prompts:
/// - "Transpose up a perfect fifth"
/// - "Make it twice as slow"
/// - "Convert to A minor"
/// - "Invert the melody"
/// - "Play it backwards"
pub async fn transform_with_prompt(sequence: &mut Sequence, prompt: &str) -> Result<String> {
    // Build the system prompt for transformation interpretation
    let system_prompt = r#"You are a music transformation assistant. Given a natural language description of a musical transformation, you will respond with a JSON object containing the transformation parameters.

Available transformations:
- "transpose": Shift all notes by semitones. Use "semitones" field (-12 to +12).
  Examples: "up a major third" = +4, "down a perfect fifth" = -7, "up an octave" = +12
- "tempo": Change the tempo. Use "bpm" field (20-300).
- "invert": Mirror the melody around a pivot pitch. Use "pivot" field (MIDI note, 0-127, or null for first note).
- "retrograde": Reverse the note order. Use "enabled" field (true/false).
- "augment": Stretch durations. Use "factor" field (e.g., 2.0 = twice as long).
- "diminish": Compress durations. Use "factor" field (e.g., 2.0 = half as long).
- "key_change": Change key. Use "root" (C, C#, D, etc.) and "mode" (major/minor) fields.

Respond ONLY with a JSON object. Example:
{"transpose": {"semitones": 7}, "description": "Transposed up a perfect fifth"}

For multiple transformations:
{"transpose": {"semitones": 5}, "tempo": {"bpm": 90}, "description": "Transposed up and slowed down"}

If the prompt is unclear, respond with your best interpretation."#;

    let user_prompt = format!(
        "Transform request: \"{}\"\n\nCurrent sequence info:\n- Notes: {}\n- Key: {}\n- Tempo: {} BPM",
        prompt,
        sequence.notes.len(),
        sequence
            .context
            .key_signature
            .as_ref()
            .map(|k| k.to_string())
            .unwrap_or_else(|| "Unknown".to_string()),
        sequence.context.tempo
    );

    // Get LLM response
    let response = call_llm(system_prompt, &user_prompt).await?;

    // Parse the JSON response - try to extract JSON from markdown code blocks
    let json_str = extract_json(&response);
    let parsed: serde_json::Value = serde_json::from_str(json_str).map_err(|e| {
        RifferError::LlmParseError(format!(
            "Failed to parse LLM response: {}. Response: {}",
            e, response
        ))
    })?;

    // Apply transformations
    let mut applied = Vec::new();

    if let Some(transpose_obj) = parsed.get("transpose") {
        if let Some(semitones) = transpose_obj.get("semitones").and_then(|s| s.as_i64()) {
            transpose(sequence, semitones as i8)?;
            applied.push(format!("Transposed by {} semitones", semitones));
        }
    }

    if let Some(tempo_obj) = parsed.get("tempo") {
        if let Some(bpm) = tempo_obj.get("bpm").and_then(|b| b.as_u64()) {
            change_tempo(sequence, bpm as u16)?;
            applied.push(format!("Changed tempo to {} BPM", bpm));
        }
    }

    if let Some(invert_obj) = parsed.get("invert") {
        let pivot = invert_obj
            .get("pivot")
            .and_then(|p| p.as_u64())
            .map(|p| p as u8);
        invert(sequence, pivot)?;
        applied.push("Inverted melody".to_string());
    }

    if let Some(retrograde_obj) = parsed.get("retrograde") {
        if retrograde_obj
            .get("enabled")
            .and_then(|e| e.as_bool())
            .unwrap_or(false)
        {
            retrograde(sequence)?;
            applied.push("Applied retrograde".to_string());
        }
    }

    if let Some(augment_obj) = parsed.get("augment") {
        if let Some(factor) = augment_obj.get("factor").and_then(|f| f.as_f64()) {
            augment(sequence, factor as f32)?;
            applied.push(format!("Augmented by factor {}", factor));
        }
    }

    if let Some(diminish_obj) = parsed.get("diminish") {
        if let Some(factor) = diminish_obj.get("factor").and_then(|f| f.as_f64()) {
            diminish(sequence, factor as f32)?;
            applied.push(format!("Diminished by factor {}", factor));
        }
    }

    if let Some(key_obj) = parsed.get("key_change") {
        let root_str = key_obj.get("root").and_then(|r| r.as_str()).unwrap_or("C");
        let mode_str = key_obj
            .get("mode")
            .and_then(|m| m.as_str())
            .unwrap_or("major");

        let root = parse_pitch_class(root_str).unwrap_or(PitchClass::C);
        let mode = if mode_str.to_lowercase().contains("minor") {
            Mode::Minor
        } else {
            Mode::Major
        };

        let target_key = KeySignature::new(root, mode);
        key_change(sequence, &target_key)?;
        applied.push(format!("Changed key to {}", target_key));
    }

    // Build result description
    let description = if applied.is_empty() {
        parsed
            .get("description")
            .and_then(|d| d.as_str())
            .unwrap_or("No transformations applied")
            .to_string()
    } else {
        applied.join("; ")
    };

    Ok(description)
}

/// Extract JSON from a response that might be wrapped in markdown code blocks
fn extract_json(response: &str) -> &str {
    // Try to find JSON in code blocks
    if let Some(start) = response.find("```json") {
        let start = start + 7;
        if let Some(end) = response[start..].find("```") {
            return response[start..start + end].trim();
        }
    }
    if let Some(start) = response.find("```") {
        let start = start + 3;
        if let Some(end) = response[start..].find("```") {
            let content = response[start..start + end].trim();
            // Skip the language identifier if present
            if let Some(newline) = content.find('\n') {
                return content[newline + 1..].trim();
            }
        }
    }
    // Return as-is if no code blocks found
    response.trim()
}

/// Parse a pitch class string like "C", "C#", "Db", etc.
fn parse_pitch_class(s: &str) -> Option<PitchClass> {
    match s.to_uppercase().as_str() {
        "C" => Some(PitchClass::C),
        "C#" | "DB" => Some(PitchClass::Cs),
        "D" => Some(PitchClass::D),
        "D#" | "EB" => Some(PitchClass::Ds),
        "E" => Some(PitchClass::E),
        "F" => Some(PitchClass::F),
        "F#" | "GB" => Some(PitchClass::Fs),
        "G" => Some(PitchClass::G),
        "G#" | "AB" => Some(PitchClass::Gs),
        "A" => Some(PitchClass::A),
        "A#" | "BB" => Some(PitchClass::As),
        "B" => Some(PitchClass::B),
        _ => None,
    }
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
