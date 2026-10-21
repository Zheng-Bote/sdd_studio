//! Tauri v2 shell for `sdd-desk`: one window, a menu to open a project, and the sidecar lifecycle.
//!
//! The window shows `sdd-desk`'s own UI from `http://127.0.0.1:<port>`; that page and the local
//! start page get no Tauri capabilities, and the shell offers no commands.

mod settings;
mod sidecar;

use std::path::Path;

use settings::{Settings, choose_project};
use sidecar::{SidecarManager, origin_key};
use tauri::menu::{MenuBuilder, MenuItemBuilder, SubmenuBuilder};
use tauri::webview::NewWindowResponse;
use tauri::{AppHandle, Manager, RunEvent, Url, WebviewUrl, WebviewWindowBuilder};
use tauri_plugin_dialog::DialogExt;
use tauri_plugin_shell::ShellExt;

const WINDOW: &str = "main";
const MENU_OPEN_PROJECT: &str = "open-project";

/// URL of the start page, as the window loaded it (scheme and host differ per platform).
struct Splash(Url);

#[derive(Debug, PartialEq, Eq)]
enum Navigation {
    Allow,
    External,
    Deny,
}

/// The window may show only the bundled start page and the running sidecar. Other web addresses
/// go to the default browser; anything else (file:, data:, javascript:, …) is dropped.
fn navigation(url: &Url, sidecar_origin: Option<&str>) -> Navigation {
    let app_page = matches!(
        (url.scheme(), url.host_str()),
        ("tauri", Some("localhost")) | ("http" | "https", Some("tauri.localhost"))
    );
    if app_page || sidecar_origin == Some(origin_key(url).as_str()) {
        Navigation::Allow
    } else if matches!(url.scheme(), "http" | "https") {
        Navigation::External
    } else {
        Navigation::Deny
    }
}

/// Start page URL with the state to show; the message must already be free of tokens.
fn splash_url(base: &Url, state: &str, project: Option<&Path>, message: Option<&str>) -> Url {
    let mut url = base.join("index.html").unwrap_or_else(|_| base.clone());
    url.set_fragment(None);
    {
        let mut query = url.query_pairs_mut();
        query.clear().append_pair("state", state);
        if let Some(project) = project {
            query.append_pair("project", &project.to_string_lossy());
        }
        if let Some(message) = message {
            query.append_pair("message", message);
        }
    }
    url
}

pub(crate) fn show_splash(app: &AppHandle, state: &str, project: Option<&Path>, message: Option<&str>) {
    let (Some(window), Some(splash)) = (app.get_webview_window(WINDOW), app.try_state::<Splash>()) else {
        return;
    };
    let _ = window.navigate(splash_url(&splash.0, state, project, message));
}

pub(crate) fn show_desk(app: &AppHandle, url: Url) {
    if let Some(window) = app.get_webview_window(WINDOW) {
        let _ = window.navigate(url);
    }
}

fn open_external(app: &AppHandle, url: &Url) {
    if matches!(url.scheme(), "http" | "https") {
        #[allow(deprecated)]
        let _ = app.shell().open(url.as_str(), None);
    }
}

fn pick_project(app: &AppHandle) {
    let handle = app.clone();
    app.dialog()
        .file()
        .set_title("Projekt öffnen")
        .pick_folder(move |folder| match folder.and_then(|f| f.into_path().ok()) {
            Some(project) => handle.state::<SidecarManager>().start(&handle, project),
            None if handle.state::<SidecarManager>().origin().is_none() => {
                show_splash(&handle, "choose", None, None);
            }
            None => {}
        });
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_shell::init())
        .plugin(tauri_plugin_dialog::init())
        .manage(SidecarManager::default())
        .menu(|app| {
            let open = MenuItemBuilder::with_id(MENU_OPEN_PROJECT, "Projekt öffnen …")
                .accelerator("CmdOrCtrl+O")
                .build(app)?;
            let file = SubmenuBuilder::new(app, "Datei")
                .item(&open)
                .separator()
                .quit_with_text("Beenden")
                .build()?;
            let edit = SubmenuBuilder::new(app, "Bearbeiten")
                .undo()
                .redo()
                .separator()
                .cut()
                .copy()
                .paste()
                .select_all()
                .build()?;
            MenuBuilder::new(app).items(&[&file, &edit]).build()
        })
        .on_menu_event(|app, event| {
            if event.id() == MENU_OPEN_PROJECT {
                pick_project(app);
            }
        })
        .setup(|app| {
            let nav_app = app.handle().clone();
            let popup_app = app.handle().clone();
            let window = WebviewWindowBuilder::new(app, WINDOW, WebviewUrl::App("index.html".into()))
                .title("SDD Studio Desktop")
                .inner_size(1200.0, 800.0)
                .min_inner_size(720.0, 520.0)
                .on_navigation(move |url| {
                    let origin = nav_app.state::<SidecarManager>().origin();
                    match navigation(url, origin.as_deref()) {
                        Navigation::Allow => true,
                        Navigation::External => {
                            open_external(&nav_app, url);
                            false
                        }
                        Navigation::Deny => false,
                    }
                })
                .on_new_window(move |url, _| {
                    open_external(&popup_app, &url);
                    NewWindowResponse::Deny
                })
                .build()?;
            app.manage(Splash(window.url()?));

            let handle = app.handle().clone();
            let saved = app.path().app_config_dir().map(|dir| Settings::load(&dir).project).unwrap_or_default();
            let arg = std::env::args().nth(1);
            match choose_project(arg.as_deref(), saved.as_deref()) {
                Some(project) => handle.state::<SidecarManager>().start(&handle, project),
                None => {
                    show_splash(&handle, "choose", None, None);
                    pick_project(&handle);
                }
            }
            Ok(())
        })
        .build(tauri::generate_context!())
        .expect("error while building the Tauri application")
        .run(|app, event| {
            if let RunEvent::Exit = event {
                app.state::<SidecarManager>().stop();
            }
        });
}

#[cfg(test)]
mod tests {
    use super::{Navigation, navigation, splash_url};
    use serde_json::Value;
    use std::path::Path;
    use tauri::Url;

    fn url(s: &str) -> Url {
        Url::parse(s).unwrap()
    }

    fn json(path: &str) -> Value {
        let full = Path::new(env!("CARGO_MANIFEST_DIR")).join(path);
        serde_json::from_slice(&std::fs::read(&full).unwrap()).unwrap()
    }

    // sdd-desk-tauri AC-4: only the start page and the running sidecar; web links leave, the rest is dropped
    #[test]
    fn navigation_allows_only_splash_and_sidecar() {
        let sidecar = Some("http://127.0.0.1:47123");
        for allowed in [
            "tauri://localhost/index.html?state=starting",
            "http://tauri.localhost/index.html",
            "http://127.0.0.1:47123/?token=abc",
            "http://127.0.0.1:47123/api/meta",
        ] {
            assert_eq!(navigation(&url(allowed), sidecar), Navigation::Allow, "{allowed}");
        }
        for external in [
            "http://127.0.0.1:47124/",
            "http://localhost:47123/",
            "https://example.com/",
            "http://tauri.localhost.evil.example/",
        ] {
            assert_eq!(navigation(&url(external), sidecar), Navigation::External, "{external}");
        }
        for denied in ["file:///etc/passwd", "data:text/html,hi", "javascript:alert(1)", "tauri://evil/x"] {
            assert_eq!(navigation(&url(denied), sidecar), Navigation::Deny, "{denied}");
        }
        assert_eq!(navigation(&url("http://127.0.0.1:47123/"), None), Navigation::External);
    }

    // sdd-desk-tauri AC-5: no capability files, no inline capabilities, no global Tauri object
    #[test]
    fn no_capabilities_are_granted() {
        let dir = Path::new(env!("CARGO_MANIFEST_DIR"));
        assert!(!dir.join("capabilities").exists(), "capabilities/ grants Tauri APIs to pages");
        let conf = json("tauri.conf.json");
        assert!(conf["app"]["security"].get("capabilities").is_none());
        assert_eq!(conf["app"]["withGlobalTauri"], Value::Bool(false));
        let source = std::fs::read_to_string(dir.join("src/lib.rs")).unwrap();
        assert!(!source.contains(concat!("invoke", "_handler")), "the shell must not offer commands");
    }

    // sdd-desk-tauri AC-7: the error page shows message and project, the caller passes only redacted text
    #[test]
    fn splash_url_carries_error_without_token() {
        let base = url("tauri://localhost/index.html");
        let message = crate::sidecar::redact("sdd-desk wurde beendet\nhttp://127.0.0.1:1/?token=deadbeef");
        let page = splash_url(&base, "error", Some(Path::new("/work/repo")), Some(&message));
        assert_eq!(page.path(), "/index.html");
        let pairs: Vec<(String, String)> = page.query_pairs().into_owned().collect();
        assert_eq!(pairs[0], ("state".into(), "error".into()));
        assert_eq!(pairs[1], ("project".into(), "/work/repo".into()));
        assert!(pairs[2].1.starts_with("sdd-desk wurde beendet"));
        assert!(!page.as_str().contains("deadbeef"));
    }

    // sdd-desk-tauri AC-8: one build command runs the UI build, then the sidecars, then bundles them
    #[test]
    fn build_runs_ui_then_sidecars() {
        let conf = json("tauri.conf.json");
        assert_eq!(conf["build"]["beforeBuildCommand"], "npm run desk:sidecars");
        assert_eq!(conf["build"]["beforeDevCommand"], "npm run desk:sidecars");
        let bins: Vec<&str> = conf["bundle"]["externalBin"].as_array().unwrap().iter().filter_map(Value::as_str).collect();
        // sdd-desk-sidecar-name AC-2: sdd is bundled under a unique name, and the build script produces that name
        assert_eq!(bins, ["binaries/sdd-desk", "binaries/sdd-studio-sdd"]);
        let script = std::fs::read_to_string(Path::new(env!("CARGO_MANIFEST_DIR")).join("../scripts/build-sidecars.mjs")).unwrap();
        assert!(script.contains("['sdd-studio-sdd', 'sdd']"), "build-sidecars.mjs must build ./cmd/sdd as sdd-studio-sdd");

        let package = json("../package.json");
        let sidecars = package["scripts"]["desk:sidecars"].as_str().unwrap();
        assert_eq!(sidecars, "npm run build:desk && node scripts/build-sidecars.mjs");
        assert_eq!(package["scripts"]["tauri:build"], "tauri build");
    }
}
