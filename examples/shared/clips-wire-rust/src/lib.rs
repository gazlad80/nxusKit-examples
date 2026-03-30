//! Shared **provider chat** JSON types (`ClipsInput` / `ClipsOutput`) for nxusKit-examples.
//!
//! Not exported by the `nxuskit` crate. Use this crate from any example that marshals
//! CLIPS provider payloads in Rust.
//!
//! - JSON reference: `conformance/clips-json-contract.json` (nxusKit-examples repo root).
//! - Documentation: nxusKit SDK `sdk-packaging/docs/rule-authoring.md` — *ClipsInput JSON Reference*.

use serde::{Deserialize, Serialize};
use std::collections::HashMap;

#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct ClipsFactWire {
    pub template: String,
    pub values: HashMap<String, serde_json::Value>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub id: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct ClipsSlotWire {
    pub name: String,
    #[serde(rename = "type")]
    pub slot_type: String,
}

#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct ClipsTemplateWire {
    pub name: String,
    #[serde(default)]
    pub slots: Vec<ClipsSlotWire>,
}

#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct ClipsRequestConfigWire {
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub max_rules: Option<i64>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub include_trace: Option<bool>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub derived_only_new: Option<bool>,
    #[serde(default, skip_serializing_if = "Vec::is_empty")]
    pub output_templates: Vec<String>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub stream_mode: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct ClipsInputWire {
    #[serde(default)]
    pub facts: Vec<ClipsFactWire>,
    #[serde(default, skip_serializing_if = "Vec::is_empty")]
    pub templates: Vec<ClipsTemplateWire>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub config: Option<ClipsRequestConfigWire>,
    #[serde(default, skip_serializing_if = "Vec::is_empty")]
    pub focus: Vec<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct ClipsConclusionWire {
    pub template: String,
    pub values: HashMap<String, serde_json::Value>,
    #[serde(default)]
    pub fact_index: i64,
    #[serde(default)]
    pub derived: bool,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub id: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct ClipsExecStatsWire {
    #[serde(default)]
    pub total_rules_fired: u64,
    #[serde(default)]
    pub conclusions_count: u64,
    #[serde(default)]
    pub execution_time_ms: u64,
}

#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct ClipsRuleFiringWire {
    pub rule_name: String,
    #[serde(default)]
    pub fire_count: u64,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub module: Option<String>,
    #[serde(default)]
    pub salience: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct ClipsTraceWire {
    #[serde(default)]
    pub rules_fired: Vec<ClipsRuleFiringWire>,
}

#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct ClipsOutputWire {
    #[serde(default)]
    pub conclusions: Vec<ClipsConclusionWire>,
    #[serde(default, skip_serializing_if = "Vec::is_empty")]
    pub input_facts: Vec<ClipsConclusionWire>,
    #[serde(default)]
    pub stats: ClipsExecStatsWire,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub trace: Option<ClipsTraceWire>,
}
