//! Settings of the shell: only the last opened project, in `<app_config_dir>/desk.json`.

use std::fs;
use std::path::{Path, PathBuf};

use serde::{Deserialize, Serialize};

const FILE_NAME: &str = "desk.json";

#[derive(Default, Serialize, Deserialize, PartialEq, Eq, Debug)]
pub struct Settings {
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub project: Option<PathBuf>,
}

impl Settings {
    /// Reads the settings; a missing or unreadable file yields the defaults.
    pub fn load(dir: &Path) -> Settings {
        fs::read(dir.join(FILE_NAME))
            .ok()
            .and_then(|data| serde_json::from_slice(&data).ok())
            .unwrap_or_default()
    }

    pub fn save(&self, dir: &Path) -> Result<(), String> {
        fs::create_dir_all(dir).map_err(|e| format!("cannot create {}: {e}", dir.display()))?;
        let data = serde_json::to_vec_pretty(self).map_err(|e| e.to_string())?;
        fs::write(dir.join(FILE_NAME), data).map_err(|e| format!("cannot write settings: {e}"))
    }
}

/// Picks the project: the start argument, then the last project, as long as it is a directory.
/// `None` means the user has to choose one.
pub fn choose_project(arg: Option<&str>, saved: Option<&Path>) -> Option<PathBuf> {
    arg.map(PathBuf::from)
        .filter(|p| p.is_dir())
        .or_else(|| saved.map(Path::to_path_buf).filter(|p| p.is_dir()))
}

#[cfg(test)]
pub(crate) mod tests {
    use super::{Settings, choose_project};
    use std::path::PathBuf;

    /// A fresh directory below the system temp dir, removed by the caller's test run at most.
    pub(crate) fn temp_dir(name: &str) -> PathBuf {
        let dir = std::env::temp_dir().join(format!("sdd-desk-app-{}-{name}", std::process::id()));
        let _ = std::fs::remove_dir_all(&dir);
        std::fs::create_dir_all(&dir).unwrap();
        dir
    }

    // sdd-desk-tauri AC-2: start argument first, then the saved project, then nothing (folder dialog)
    #[test]
    fn project_from_arg_then_settings() {
        let arg = temp_dir("arg");
        let saved = temp_dir("saved");
        let missing = saved.join("missing");

        assert_eq!(choose_project(arg.to_str(), Some(&saved)), Some(arg.clone()));
        assert_eq!(choose_project(missing.to_str(), Some(&saved)), Some(saved.clone()));
        assert_eq!(choose_project(Some("-psn_0_12345"), Some(&saved)), Some(saved.clone()));
        assert_eq!(choose_project(None, Some(&missing)), None);
        assert_eq!(choose_project(None, None), None);
    }

    // sdd-desk-tauri AC-2: the chosen project is remembered across starts
    #[test]
    fn settings_roundtrip() {
        let dir = temp_dir("settings");
        assert_eq!(Settings::load(&dir), Settings::default());

        let settings = Settings { project: Some(PathBuf::from("/work/repo")) };
        settings.save(&dir.join("nested")).unwrap();
        assert_eq!(Settings::load(&dir.join("nested")), settings);

        std::fs::write(dir.join("desk.json"), b"not json").unwrap();
        assert_eq!(Settings::load(&dir), Settings::default());
    }
}
