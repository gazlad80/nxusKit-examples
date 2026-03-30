//! egui auth helper — credential management GUI.
//!
//! Demonstrates the nxusKit auth helper API via a native desktop application.
//! Provider list with status indicators, credential set/remove forms,
//! and dashboard links.

use auth_helper_core::{
    ProviderAuthStatus, ProviderEntry, list_providers, open_dashboard, remove_credential,
    set_credential,
};
use eframe::egui;
use egui_notify::Toasts;

fn main() -> eframe::Result {
    let options = eframe::NativeOptions {
        viewport: egui::ViewportBuilder::default()
            .with_inner_size([640.0, 480.0])
            .with_min_inner_size([480.0, 360.0]),
        ..Default::default()
    };
    eframe::run_native(
        "nxusKit Auth Helper",
        options,
        Box::new(|_cc| Ok(Box::new(AuthHelperApp::new()))),
    )
}

struct AuthHelperApp {
    providers: Vec<ProviderEntry>,
    selected: Option<usize>,
    key_input: String,
    error_msg: Option<String>,
    toasts: Toasts,
}

impl AuthHelperApp {
    fn new() -> Self {
        let providers = list_providers().unwrap_or_default();
        Self {
            providers,
            selected: None,
            key_input: String::new(),
            error_msg: None,
            toasts: Toasts::default(),
        }
    }

    fn refresh(&mut self) {
        match list_providers() {
            Ok(p) => {
                self.providers = p;
                self.error_msg = None;
            }
            Err(e) => {
                self.error_msg = Some(e.to_string());
            }
        }
    }

    fn status_color(status: &ProviderAuthStatus) -> egui::Color32 {
        match status {
            ProviderAuthStatus::AuthenticatedStore | ProviderAuthStatus::AuthenticatedEnv => {
                egui::Color32::from_rgb(166, 227, 161) // green
            }
            ProviderAuthStatus::AuthenticatedExplicit => {
                egui::Color32::from_rgb(249, 226, 175) // yellow
            }
            ProviderAuthStatus::NotAuthenticated => {
                egui::Color32::from_rgb(243, 139, 168) // red
            }
            ProviderAuthStatus::NotRequired => {
                egui::Color32::from_rgb(147, 153, 178) // gray
            }
        }
    }
}

impl eframe::App for AuthHelperApp {
    fn update(&mut self, ctx: &egui::Context, _frame: &mut eframe::Frame) {
        // Left sidebar: provider list
        egui::SidePanel::left("providers")
            .min_width(200.0)
            .show(ctx, |ui| {
                ui.heading("Providers");
                ui.separator();

                if ui.button("Refresh").clicked() {
                    self.refresh();
                }
                ui.separator();

                for (i, entry) in self.providers.iter().enumerate() {
                    let selected = self.selected == Some(i);
                    let color = Self::status_color(&entry.status);

                    ui.horizontal(|ui| {
                        // Status dot
                        let (rect, _) =
                            ui.allocate_exact_size(egui::vec2(10.0, 10.0), egui::Sense::hover());
                        ui.painter().circle_filled(rect.center(), 4.0, color);

                        // Provider name button
                        if ui.selectable_label(selected, &entry.display_name).clicked() {
                            self.selected = Some(i);
                            self.key_input.clear();
                        }
                    });

                    // Masked preview under name
                    if let Some(preview) = &entry.masked_preview {
                        ui.indent(entry.id.as_str(), |ui| {
                            ui.label(
                                egui::RichText::new(preview)
                                    .small()
                                    .color(egui::Color32::GRAY),
                            );
                        });
                    }
                }
            });

        // Main panel: credential form
        egui::CentralPanel::default().show(ctx, |ui| {
            if let Some(idx) = self.selected {
                if let Some(entry) = self.providers.get(idx).cloned() {
                    ui.heading(format!("Configure {}", entry.display_name));
                    ui.separator();

                    // Status
                    let color = Self::status_color(&entry.status);
                    ui.horizontal(|ui| {
                        ui.label("Status:");
                        ui.colored_label(color, entry.status.label());
                    });

                    if let Some(preview) = &entry.masked_preview {
                        ui.horizontal(|ui| {
                            ui.label("Key:");
                            ui.monospace(preview);
                        });
                    }

                    // Precedence info
                    ui.horizontal(|ui| {
                        ui.label("Env var:");
                        ui.monospace(&entry.env_var_name);
                    });

                    ui.add_space(16.0);

                    // Set credential
                    if entry.auth_required {
                        ui.label("API Key:");
                        ui.add(
                            egui::TextEdit::singleline(&mut self.key_input)
                                .password(true)
                                .hint_text("Enter API key..."),
                        );

                        ui.horizontal(|ui| {
                            if ui.button("Set").clicked() && !self.key_input.is_empty() {
                                match set_credential(&entry.id, &self.key_input) {
                                    Ok(()) => {
                                        self.toasts.success("Credential stored");
                                        self.key_input.clear();
                                        self.refresh();
                                    }
                                    Err(e) => {
                                        self.toasts.error(format!("Set failed: {e}"));
                                    }
                                }
                            }

                            if ui.button("Remove").clicked() {
                                match remove_credential(&entry.id) {
                                    Ok(()) => {
                                        self.toasts.success("Credential removed");
                                        self.refresh();
                                    }
                                    Err(e) => {
                                        self.toasts.error(format!("Remove failed: {e}"));
                                    }
                                }
                            }

                            if entry.dashboard_url.is_some()
                                && ui.button("Dashboard").clicked()
                                && let Err(e) = open_dashboard(&entry.id)
                            {
                                self.toasts.error(format!("Dashboard: {e}"));
                            }
                        });
                    } else {
                        ui.label("This is a local provider — no authentication required.");
                    }

                    // Precedence explanation
                    ui.add_space(16.0);
                    ui.separator();
                    ui.label(
                        egui::RichText::new("Credential Precedence: explicit > env > store > none")
                            .small()
                            .color(egui::Color32::GRAY),
                    );
                }
            } else {
                ui.centered_and_justified(|ui| {
                    ui.label("Select a provider from the sidebar");
                });
            }

            if let Some(err) = &self.error_msg {
                ui.colored_label(egui::Color32::RED, err);
            }
        });

        self.toasts.show(ctx);
    }
}
