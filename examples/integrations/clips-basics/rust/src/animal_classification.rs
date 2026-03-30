//! Example: Animal Classification with CLIPS
//!
//! This example demonstrates basic CLIPS usage with the animal classification
//! rule base. It shows how to:
//! - Load a .clp rule file
//! - Assert facts about animals
//! - Run inference to classify them
//!
//! Run with: cargo run --bin animal_classification

use clips_basics_example::clips_wire::ClipsInputWire;
use nxuskit::{ChatRequest, Message, NxuskitProvider, ProviderConfig};

fn main() -> Result<(), Box<dyn std::error::Error>> {
    println!("=== Animal Classification Expert System ===\n");

    // Create CLIPS provider via NxuskitProvider
    let provider = NxuskitProvider::new(ProviderConfig {
        provider_type: "clips".to_string(),
        model: Some("../../../shared/rules".to_string()),
        ..Default::default()
    })?;

    // Example 1: Classify a dog (mammal)
    println!("--- Classifying: Dog ---\n");

    let dog_input: ClipsInputWire = serde_json::from_value(serde_json::json!({
        "facts": [
            {
                "template": "animal",
                "values": {
                    "name": "Buddy",
                    "has-backbone": {"symbol": "yes"},
                    "body-temperature": {"symbol": "warm"},
                    "has-feathers": {"symbol": "no"},
                    "has-fur": {"symbol": "yes"},
                    "has-scales": {"symbol": "no"},
                    "lives-in-water": {"symbol": "no"},
                    "can-fly": {"symbol": "no"},
                    "lays-eggs": {"symbol": "no"}
                }
            }
        ]
    }))?;

    let request = ChatRequest::new("animal-classification.clp")
        .with_message(Message::user(serde_json::to_string(&dog_input)?));

    let response = provider.chat(request)?;
    println!("Dog classification result:\n{}\n", response.content);

    // Example 2: Classify an eagle (flying bird)
    println!("--- Classifying: Eagle ---\n");

    let eagle_input: ClipsInputWire = serde_json::from_value(serde_json::json!({
        "facts": [
            {
                "template": "animal",
                "values": {
                    "name": "Skyler",
                    "has-backbone": {"symbol": "yes"},
                    "body-temperature": {"symbol": "warm"},
                    "has-feathers": {"symbol": "yes"},
                    "has-fur": {"symbol": "no"},
                    "has-scales": {"symbol": "no"},
                    "lives-in-water": {"symbol": "no"},
                    "can-fly": {"symbol": "yes"},
                    "lays-eggs": {"symbol": "yes"}
                }
            }
        ]
    }))?;

    let request = ChatRequest::new("animal-classification.clp")
        .with_message(Message::user(serde_json::to_string(&eagle_input)?));

    let response = provider.chat(request)?;
    println!("Eagle classification result:\n{}\n", response.content);

    // Example 3: Classify a salmon (fish)
    println!("--- Classifying: Salmon ---\n");

    let salmon_input: ClipsInputWire = serde_json::from_value(serde_json::json!({
        "facts": [
            {
                "template": "animal",
                "values": {
                    "name": "Finn",
                    "has-backbone": {"symbol": "yes"},
                    "body-temperature": {"symbol": "cold"},
                    "has-feathers": {"symbol": "no"},
                    "has-fur": {"symbol": "no"},
                    "has-scales": {"symbol": "yes"},
                    "lives-in-water": {"symbol": "yes"},
                    "can-fly": {"symbol": "no"},
                    "lays-eggs": {"symbol": "yes"}
                }
            }
        ]
    }))?;

    let request = ChatRequest::new("animal-classification.clp")
        .with_message(Message::user(serde_json::to_string(&salmon_input)?));

    let response = provider.chat(request)?;
    println!("Salmon classification result:\n{}\n", response.content);

    // Example 4: Classify a platypus (monotreme - egg-laying mammal)
    println!("--- Classifying: Platypus (edge case) ---\n");

    let platypus_input: ClipsInputWire = serde_json::from_value(serde_json::json!({
        "facts": [
            {
                "template": "animal",
                "values": {
                    "name": "Perry",
                    "has-backbone": {"symbol": "yes"},
                    "body-temperature": {"symbol": "warm"},
                    "has-feathers": {"symbol": "no"},
                    "has-fur": {"symbol": "yes"},
                    "has-scales": {"symbol": "no"},
                    "lives-in-water": {"symbol": "partial"},
                    "can-fly": {"symbol": "no"},
                    "lays-eggs": {"symbol": "yes"}
                }
            }
        ]
    }))?;

    let request = ChatRequest::new("animal-classification.clp")
        .with_message(Message::user(serde_json::to_string(&platypus_input)?));

    let response = provider.chat(request)?;
    println!("Platypus classification result:\n{}\n", response.content);

    // Example 5: Classify multiple animals at once
    println!("--- Classifying multiple animals ---\n");

    let multi_input: ClipsInputWire = serde_json::from_value(serde_json::json!({
        "facts": [
            {
                "template": "animal",
                "values": {
                    "name": "Frog",
                    "has-backbone": {"symbol": "yes"},
                    "body-temperature": {"symbol": "cold"},
                    "has-feathers": {"symbol": "no"},
                    "has-fur": {"symbol": "no"},
                    "has-scales": {"symbol": "no"},
                    "lives-in-water": {"symbol": "partial"},
                    "can-fly": {"symbol": "no"},
                    "lays-eggs": {"symbol": "yes"}
                }
            },
            {
                "template": "animal",
                "values": {
                    "name": "Penguin",
                    "has-backbone": {"symbol": "yes"},
                    "body-temperature": {"symbol": "warm"},
                    "has-feathers": {"symbol": "yes"},
                    "has-fur": {"symbol": "no"},
                    "has-scales": {"symbol": "no"},
                    "lives-in-water": {"symbol": "partial"},
                    "can-fly": {"symbol": "no"},
                    "lays-eggs": {"symbol": "yes"}
                }
            },
            {
                "template": "animal",
                "values": {
                    "name": "Spider",
                    "has-backbone": {"symbol": "no"},
                    "body-temperature": {"symbol": "cold"},
                    "has-feathers": {"symbol": "no"},
                    "has-fur": {"symbol": "no"},
                    "has-scales": {"symbol": "no"},
                    "lives-in-water": {"symbol": "no"},
                    "can-fly": {"symbol": "no"},
                    "lays-eggs": {"symbol": "yes"}
                }
            }
        ]
    }))?;

    let request = ChatRequest::new("animal-classification.clp")
        .with_message(Message::user(serde_json::to_string(&multi_input)?));

    let response = provider.chat(request)?;
    println!("Multiple animal classification:\n{}\n", response.content);

    println!("=== Animal Classification Complete ===");

    Ok(())
}
