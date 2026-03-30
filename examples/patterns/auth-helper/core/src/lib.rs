//! Shared auth flow logic consumed by both the egui and CLI auth helper examples.
//!
//! Wraps `nxuskit::auth_*` functions with ergonomic types for UI/CLI consumption.

use nxuskit::{
    AuthStatus, ProviderAuthMetadata, auth_providers, auth_remove_credential, auth_resolve,
    auth_set_credential, auth_status_all,
};

/// Provider info with resolved auth status.
#[derive(Debug, Clone)]
pub struct ProviderEntry {
    pub id: String,
    pub display_name: String,
    pub auth_required: bool,
    pub dashboard_url: Option<String>,
    pub env_var_name: String,
    pub status: ProviderAuthStatus,
    pub masked_preview: Option<String>,
}

/// Simplified auth status enum for display.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum ProviderAuthStatus {
    AuthenticatedStore,
    AuthenticatedEnv,
    AuthenticatedExplicit,
    NotAuthenticated,
    NotRequired,
}

impl ProviderAuthStatus {
    pub fn label(&self) -> &'static str {
        match self {
            Self::AuthenticatedStore => "authenticated_store",
            Self::AuthenticatedEnv => "authenticated_env",
            Self::AuthenticatedExplicit => "authenticated_explicit",
            Self::NotAuthenticated => "not_authenticated",
            Self::NotRequired => "not_required",
        }
    }

    pub fn is_authenticated(&self) -> bool {
        matches!(
            self,
            Self::AuthenticatedStore | Self::AuthenticatedEnv | Self::AuthenticatedExplicit
        )
    }

    fn from_status_string(s: &str) -> Self {
        match s {
            "authenticated_store" => Self::AuthenticatedStore,
            "authenticated_env" => Self::AuthenticatedEnv,
            "authenticated_explicit" => Self::AuthenticatedExplicit,
            "not_required" => Self::NotRequired,
            _ => Self::NotAuthenticated,
        }
    }
}

/// Error type for auth helper operations.
#[derive(Debug)]
pub enum AuthHelperError {
    Sdk(String),
    NoBrowser(String),
}

impl std::fmt::Display for AuthHelperError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            Self::Sdk(msg) => write!(f, "{msg}"),
            Self::NoBrowser(msg) => write!(f, "Could not open browser: {msg}"),
        }
    }
}

impl std::error::Error for AuthHelperError {}

impl From<nxuskit::NxuskitError> for AuthHelperError {
    fn from(e: nxuskit::NxuskitError) -> Self {
        Self::Sdk(e.to_string())
    }
}

/// List all providers with their current auth status.
pub fn list_providers() -> Result<Vec<ProviderEntry>, AuthHelperError> {
    let metadata = auth_providers().map_err(|e| AuthHelperError::Sdk(e.to_string()))?;
    let statuses = auth_status_all().map_err(|e| AuthHelperError::Sdk(e.to_string()))?;

    let entries = metadata
        .into_iter()
        .map(|meta| {
            let status_info = statuses.iter().find(|s| s.provider_id == meta.provider_id);
            build_entry(meta, status_info)
        })
        .collect();

    Ok(entries)
}

fn build_entry(meta: ProviderAuthMetadata, status: Option<&AuthStatus>) -> ProviderEntry {
    let (auth_status, masked) = match status {
        Some(s) => (
            ProviderAuthStatus::from_status_string(&s.status),
            s.masked_preview.clone(),
        ),
        None if !meta.auth_required => (ProviderAuthStatus::NotRequired, None),
        None => (ProviderAuthStatus::NotAuthenticated, None),
    };

    ProviderEntry {
        id: meta.provider_id,
        display_name: meta.display_name,
        auth_required: meta.auth_required,
        dashboard_url: meta.dashboard_url,
        env_var_name: meta.env_var_name,
        status: auth_status,
        masked_preview: masked,
    }
}

/// Format a status line for terminal display.
pub fn format_status_line(entry: &ProviderEntry) -> String {
    let preview = entry
        .masked_preview
        .as_deref()
        .unwrap_or(match entry.status {
            ProviderAuthStatus::NotRequired => "(local)",
            ProviderAuthStatus::AuthenticatedEnv => {
                // Show env var hint
                return format!(
                    "  {:<14} {:<24} (from {})",
                    entry.id,
                    entry.status.label(),
                    entry.env_var_name
                );
            }
            _ => "-",
        });
    format!(
        "  {:<14} {:<24} {}",
        entry.id,
        entry.status.label(),
        preview,
    )
}

/// Store a credential for a provider.
pub fn set_credential(provider_id: &str, api_key: &str) -> Result<(), AuthHelperError> {
    auth_set_credential(provider_id, api_key)?;
    Ok(())
}

/// Remove a stored credential for a provider.
pub fn remove_credential(provider_id: &str) -> Result<(), AuthHelperError> {
    auth_remove_credential(provider_id)?;
    Ok(())
}

/// Resolve a credential and return its source.
pub fn resolve_credential(
    provider_id: &str,
    explicit: Option<&str>,
) -> Result<(bool, String), AuthHelperError> {
    let resolution = auth_resolve(provider_id, explicit)?;
    Ok((resolution.has_credential, resolution.source))
}

/// Open the provider's dashboard URL in the default browser.
pub fn open_dashboard(provider_id: &str) -> Result<(), AuthHelperError> {
    let providers = auth_providers().map_err(|e| AuthHelperError::Sdk(e.to_string()))?;
    let meta = providers
        .iter()
        .find(|p| p.provider_id == provider_id)
        .ok_or_else(|| AuthHelperError::Sdk(format!("Unknown provider: {provider_id}")))?;

    let url = meta
        .dashboard_url
        .as_deref()
        .ok_or_else(|| AuthHelperError::Sdk(format!("No dashboard URL for {provider_id}")))?;

    open::that(url).map_err(|e| AuthHelperError::NoBrowser(e.to_string()))
}
