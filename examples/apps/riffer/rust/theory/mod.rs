//! Music theory module for Riffer
//!
//! Provides interval classification, scale definitions, and key detection.

pub mod intervals;
pub mod keys;
pub mod scales;

#[allow(unused_imports)]
pub use keys::{KeyDetection, detect_key};
