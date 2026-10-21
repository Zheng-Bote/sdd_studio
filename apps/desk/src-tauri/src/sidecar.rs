//! Lifecycle of the `sdd-desk` sidecar.
//!
//! Every project change starts a new generation; events of an older generation (output, exit,
//! start timeout) are discarded, so switching projects never leaves stale state behind.
//! `sdd-desk` runs with `--exit-with-stdin` and starts `sdd mcp` itself; both end when their stdin
//! closes, so a crashed or killed shell leaves no process behind.

use std::collections::VecDeque;
use std::path::{Path, PathBuf};
use std::sync::Mutex;
use std::time::Duration;

use tauri::{AppHandle, Manager, Url};
use tauri_plugin_shell::ShellExt;
use tauri_plugin_shell::process::{CommandChild, CommandEvent};

use crate::settings::Settings;

const SIDECAR: &str = "sdd-desk";
const READY_TIMEOUT: Duration = Duration::from_secs(30);
const LOG_TAIL_LINES: usize = 30;
const CONFIG_FILE: &str = "llm.env";

/// Arguments for `sdd-desk`: the project, a free port on 127.0.0.1 (`sdd-desk` binds only there) and
/// `--exit-with-stdin`, so it also ends when the shell dies without running its exit handler.
pub fn desk_args(project: &Path, config: Option<&ConfigArg>) -> Vec<String> {
    let mut args = vec![
        "--project".to_string(),
        project.to_string_lossy().into_owned(),
        "--port".to_string(),
        "0".to_string(),
        "--exit-with-stdin".to_string(),
    ];
    match config {
        Some(ConfigArg::File(path)) => args.extend(["--config".to_string(), path.to_string_lossy().into_owned()]),
        Some(ConfigArg::Hint(path)) => args.extend(["--config-hint".to_string(), path.to_string_lossy().into_owned()]),
        None => {}
    }
    args
}

/// Where `sdd-desk` finds `llm.env`: the file in the app config dir, or, if it is missing, only the hint naming that
/// location in the warning while `sdd-desk` falls back to `config/llm.env` next to itself.
#[derive(Debug, PartialEq, Eq)]
pub enum ConfigArg {
    File(PathBuf),
    Hint(PathBuf),
}

/// `<app_config_dir>/llm.env` as `File` if it exists, otherwise as `Hint`.
/// Only the path is passed on; `sdd-desk` checks the file's permissions and reads the keys.
pub fn config_arg(app_config_dir: &Path) -> ConfigArg {
    let path = app_config_dir.join(CONFIG_FILE);
    if path.is_file() {
        ConfigArg::File(path)
    } else {
        ConfigArg::Hint(path)
    }
}

/// Recognises the line in which `sdd-desk` announces its address with the session token.
pub fn ready_url(line: &str) -> Option<Url> {
    let line = line.trim();
    if !line.starts_with("http://127.0.0.1:") || !line.contains("/?token=") {
        return None;
    }
    let url = Url::parse(line).ok()?;
    let token = url.query()?.strip_prefix("token=")?;
    let valid = url.host_str() == Some("127.0.0.1")
        && url.port().is_some()
        && url.path() == "/"
        && !token.is_empty()
        && token.chars().all(|c| c.is_ascii_hexdigit());
    valid.then_some(url)
}

/// Replaces every session token in a line, so log and error page never show it.
pub fn redact(line: &str) -> String {
    const KEY: &str = "token=";
    let mut out = String::with_capacity(line.len());
    let mut rest = line;
    while let Some(i) = rest.find(KEY) {
        out.push_str(&rest[..i + KEY.len()]);
        rest = &rest[i + KEY.len()..];
        let end = rest.find(|c: char| !c.is_ascii_alphanumeric()).unwrap_or(rest.len());
        if end > 0 {
            out.push('…');
        }
        rest = &rest[end..];
    }
    out.push_str(rest);
    out
}

/// Scheme, host and port of a URL; the navigation rule compares these.
pub fn origin_key(url: &Url) -> String {
    format!(
        "{}://{}:{}",
        url.scheme(),
        url.host_str().unwrap_or(""),
        url.port_or_known_default().unwrap_or(0)
    )
}

#[derive(Default)]
struct Inner {
    generation: u64,
    child: Option<CommandChild>,
    ready: bool,
    origin: Option<String>,
    project: Option<PathBuf>,
    log_tail: VecDeque<String>,
}

#[derive(Default)]
pub struct SidecarManager {
    inner: Mutex<Inner>,
}

impl SidecarManager {
    /// Origin of the running, ready sidecar; the only remote origin the window may show.
    pub fn origin(&self) -> Option<String> {
        self.inner.lock().unwrap().origin.clone()
    }

    /// Stops the current sidecar and starts one for `project`.
    pub fn start(&self, app: &AppHandle, project: PathBuf) {
        let project = project.canonicalize().unwrap_or(project);
        let generation = {
            let mut inner = self.inner.lock().unwrap();
            inner.generation += 1;
            kill(&mut inner);
            inner.project = Some(project.clone());
            inner.log_tail.clear();
            inner.generation
        };
        crate::show_splash(app, "starting", Some(&project), None);

        let config = app.path().app_config_dir().ok().map(|dir| config_arg(&dir));
        let command = match app.shell().sidecar(SIDECAR) {
            Ok(command) => command.args(desk_args(&project, config.as_ref())),
            Err(e) => return self.fail(app, generation, format!("{SIDECAR} nicht gefunden: {e}")),
        };
        let (mut events, child) = match command.spawn() {
            Ok(spawned) => spawned,
            Err(e) => return self.fail(app, generation, format!("{SIDECAR} startet nicht: {e}")),
        };
        {
            let mut inner = self.inner.lock().unwrap();
            if inner.generation != generation {
                let _ = child.kill();
                return;
            }
            inner.child = Some(child);
        }

        let events_app = app.clone();
        tauri::async_runtime::spawn(async move {
            let manager = events_app.state::<SidecarManager>();
            while let Some(event) = events.recv().await {
                match event {
                    CommandEvent::Stdout(line) | CommandEvent::Stderr(line) => {
                        let text = String::from_utf8_lossy(&line);
                        match ready_url(&text) {
                            Some(url) => manager.ready(&events_app, generation, url),
                            None => manager.log(generation, &text),
                        }
                    }
                    CommandEvent::Terminated(payload) => {
                        let code = payload.code.map_or_else(|| "Signal".to_string(), |c| c.to_string());
                        manager.fail(&events_app, generation, format!("{SIDECAR} wurde beendet (Code {code})."));
                        break;
                    }
                    _ => {}
                }
            }
        });

        let timeout_app = app.clone();
        tauri::async_runtime::spawn(async move {
            tokio::time::sleep(READY_TIMEOUT).await;
            let manager = timeout_app.state::<SidecarManager>();
            if manager.waiting(generation) {
                let secs = READY_TIMEOUT.as_secs();
                manager.fail(&timeout_app, generation, format!("{SIDECAR} hat sich nach {secs} s nicht gemeldet."));
            }
        });
    }

    /// Terminates the sidecar and invalidates any running start.
    pub fn stop(&self) {
        let mut inner = self.inner.lock().unwrap();
        inner.generation += 1;
        kill(&mut inner);
    }

    fn waiting(&self, generation: u64) -> bool {
        let inner = self.inner.lock().unwrap();
        inner.generation == generation && inner.child.is_some() && !inner.ready
    }

    fn ready(&self, app: &AppHandle, generation: u64, url: Url) {
        let project = {
            let mut inner = self.inner.lock().unwrap();
            if inner.generation != generation || inner.ready {
                return;
            }
            inner.ready = true;
            inner.origin = Some(origin_key(&url));
            inner.project.clone()
        };
        if let (Some(project), Ok(dir)) = (project, app.path().app_config_dir()) {
            let _ = Settings { project: Some(project) }.save(&dir);
        }
        crate::show_desk(app, url);
    }

    fn log(&self, generation: u64, text: &str) {
        let mut inner = self.inner.lock().unwrap();
        if inner.generation != generation {
            return;
        }
        for line in text.lines().map(str::trim_end).filter(|l| !l.is_empty()) {
            if inner.log_tail.len() == LOG_TAIL_LINES {
                inner.log_tail.pop_front();
            }
            inner.log_tail.push_back(redact(line));
        }
    }

    /// Ends the generation with an error page that shows the message and the last log lines.
    fn fail(&self, app: &AppHandle, generation: u64, message: String) {
        let (project, text) = {
            let mut inner = self.inner.lock().unwrap();
            if inner.generation != generation {
                return;
            }
            inner.generation += 1;
            kill(&mut inner);
            let mut text = message;
            if !inner.log_tail.is_empty() {
                text.push_str("\n\n");
                text.push_str(&inner.log_tail.iter().cloned().collect::<Vec<_>>().join("\n"));
            }
            (inner.project.clone(), text)
        };
        crate::show_splash(app, "error", project.as_deref(), Some(&text));
    }
}

fn kill(inner: &mut Inner) {
    if let Some(child) = inner.child.take() {
        let _ = child.kill();
    }
    inner.ready = false;
    inner.origin = None;
}

#[cfg(test)]
mod tests {
    use super::{config_arg, desk_args, origin_key, ready_url, redact, ConfigArg};
    use crate::settings::tests::temp_dir;
    use std::path::Path;

    const TOKEN: &str = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef";

    // sdd-desk-tauri AC-1: sdd-desk gets the project and port 0, so it picks a free port on 127.0.0.1
    #[test]
    fn desk_args_bind_project_and_free_port() {
        assert_eq!(
            desk_args(Path::new("/work/repo"), None),
            ["--project", "/work/repo", "--port", "0", "--exit-with-stdin"]
        );
        let args = desk_args(Path::new("/work/repo"), Some(&ConfigArg::File("/cfg/llm.env".into())));
        assert!(args.ends_with(&["--config".to_string(), "/cfg/llm.env".to_string()]));
        assert!(!args.iter().any(|a| a == "--open"));
        // sdd-desk-config-hint AC-2: a missing llm.env in the app config dir is named as hint, not passed as config
        let args = desk_args(Path::new("/work/repo"), Some(&ConfigArg::Hint("/cfg/llm.env".into())));
        assert!(args.ends_with(&["--config-hint".to_string(), "/cfg/llm.env".to_string()]));
        assert!(!args.iter().any(|a| a == "--config"));
    }

    // sdd-desk-tauri AC-7: sdd-desk ends with the shell even if the shell is killed (stdin closes)
    #[test]
    fn sidecar_ends_with_the_shell() {
        let args = desk_args(Path::new("/work/repo"), Some(&ConfigArg::File("/cfg/llm.env".into())));
        assert!(args.iter().any(|a| a == "--exit-with-stdin"));
    }

    // sdd-desk-tauri AC-6: llm.env in the app config dir is passed as a path, otherwise sdd-desk's default applies
    // sdd-desk-config-hint AC-2: without the file the same path becomes the hint for the warning
    #[test]
    fn config_is_passed_only_when_present() {
        let dir = temp_dir("config");
        let path = dir.join("llm.env");
        assert_eq!(config_arg(&dir), ConfigArg::Hint(path.clone()));
        std::fs::write(&path, b"OLLAMA_API_KEY=\n").unwrap();
        assert_eq!(config_arg(&dir), ConfigArg::File(path.clone()));
        std::fs::remove_file(&path).unwrap();
        std::fs::create_dir(&path).unwrap();
        assert_eq!(config_arg(&dir), ConfigArg::Hint(path));
    }

    // sdd-desk-tauri AC-3: only sdd-desk's address line on 127.0.0.1 with a hex token counts as ready
    #[test]
    fn ready_url_is_recognised() {
        let line = format!("http://127.0.0.1:47123/?token={TOKEN}\n");
        let url = ready_url(&line).expect("ready line");
        assert_eq!(origin_key(&url), "http://127.0.0.1:47123");

        for bad in [
            "sdd-desk läuft auf 127.0.0.1:47123 für /work/repo (Konfiguration: x). Beenden mit Strg+C.".to_string(),
            format!("http://localhost:47123/?token={TOKEN}"),
            format!("https://127.0.0.1:47123/?token={TOKEN}"),
            format!("http://127.0.0.1:47123/other?token={TOKEN}"),
            "http://127.0.0.1:47123/?token=".to_string(),
            "http://127.0.0.1:47123/?token=<script>".to_string(),
            format!("http://127.0.0.1.evil.example:80/?token={TOKEN}"),
        ] {
            assert!(ready_url(&bad).is_none(), "accepted {bad}");
        }
    }

    // sdd-desk-tauri AC-3: tokens never reach the log or the error page
    #[test]
    fn token_never_reaches_the_log() {
        let line = format!("GET /api/meta?token={TOKEN} failed; token={TOKEN}");
        let clean = redact(&line);
        assert!(!clean.contains(TOKEN), "{clean}");
        assert_eq!(clean, "GET /api/meta?token=… failed; token=…");
        assert_eq!(redact("no secret here"), "no secret here");
        assert_eq!(redact("token="), "token=");
    }
}
