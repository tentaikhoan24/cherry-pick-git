use serde::{Deserialize, Serialize};
use serde_json::{json, Value};
use std::collections::HashMap;
use std::fs;
use std::path::PathBuf;
use std::sync::atomic::{AtomicU64, Ordering};
use std::sync::Mutex;
use tauri::Emitter;
use tauri::Manager;
use tauri_plugin_shell::process::CommandEvent;
use tauri_plugin_shell::ShellExt;

static CALL_ID: AtomicU64 = AtomicU64::new(1);

// Tracks all currently-running sidecar children by call ID.
// Each concurrent call gets its own slot so kills are scoped correctly.
struct ActiveSidecar(Mutex<HashMap<u64, tauri_plugin_shell::process::CommandChild>>);

// ── sidecar_call ─────────────────────────────────────────────────────────────

#[tauri::command]
async fn sidecar_call(
    app: tauri::AppHandle,
    method: String,
    params: Option<Value>,
) -> Result<Value, String> {
    let sidecar = app
        .shell()
        .sidecar("sidecar")
        .map_err(|e| format!("sidecar lookup failed: {e}"))?;

    let (mut rx, mut child) = sidecar
        .spawn()
        .map_err(|e| format!("sidecar spawn failed: {e}"))?;

    let request = json!({
        "jsonrpc": "2.0",
        "id": 1,
        "method": method,
        "params": params,
    });
    child
        .write(format!("{}\n", request).as_bytes())
        .map_err(|e| format!("sidecar stdin write failed: {e}"))?;

    // Give this call a unique ID and store child so sidecar_cancel can kill it.
    let call_id = CALL_ID.fetch_add(1, Ordering::Relaxed);
    {
        let state = app.state::<ActiveSidecar>();
        state.0.lock().unwrap().insert(call_id, child);
    }

    let mut result: Option<Result<Value, String>> = None;

    while let Some(event) = rx.recv().await {
        match event {
            CommandEvent::Stdout(bytes) => {
                let text = String::from_utf8_lossy(&bytes);
                let parsed: Value = match serde_json::from_str(text.trim()) {
                    Ok(v) => v,
                    Err(e) => {
                        result = Some(Err(format!(
                            "failed to parse sidecar response `{text}`: {e}"
                        )));
                        break;
                    }
                };

                // Progress notification: has "progress" field, no "result"/"error".
                if let Some(progress) = parsed.get("progress") {
                    let _ = app.emit("cp-progress", progress.clone());
                    continue;
                }

                // Final result — remove only this call's child and kill it.
                if let Some(c) = app.state::<ActiveSidecar>().0.lock().unwrap().remove(&call_id) {
                    let _ = c.kill();
                }
                result = Some(Ok(parsed));
                break;
            }
            CommandEvent::Stderr(bytes) => {
                eprintln!("sidecar[stderr]: {}", String::from_utf8_lossy(&bytes));
            }
            CommandEvent::Terminated(payload) => {
                app.state::<ActiveSidecar>().0.lock().unwrap().remove(&call_id);
                if result.is_none() {
                    result = Some(Err(format!(
                        "sidecar terminated before response (code={:?})",
                        payload.code
                    )));
                }
                break;
            }
            CommandEvent::Error(err) => {
                app.state::<ActiveSidecar>().0.lock().unwrap().remove(&call_id);
                result = Some(Err(format!("sidecar error: {err}")));
                break;
            }
            _ => {}
        }
    }

    result.unwrap_or(Err("sidecar closed without sending a response".to_string()))
}

// ── sidecar_cancel ───────────────────────────────────────────────────────────

#[tauri::command]
async fn sidecar_cancel(app: tauri::AppHandle) -> Result<(), String> {
    let state = app.state::<ActiveSidecar>();
    let children: Vec<_> = state.0.lock().unwrap().drain().collect();
    for (_, child) in children {
        let _ = child.kill();
    }
    Ok(())
}

// ── recent repos ─────────────────────────────────────────────────────────────

#[derive(Debug, Clone, Serialize, Deserialize)]
struct RecentRepo {
    path: String,
    #[serde(rename = "lastOpened")]
    last_opened: u64,
}

fn recents_file(app: &tauri::AppHandle) -> Result<PathBuf, String> {
    let mut p = app
        .path()
        .app_data_dir()
        .map_err(|e| format!("app_data_dir failed: {e}"))?;
    p.push("recents.json");
    Ok(p)
}

#[tauri::command]
fn recents_load(app: tauri::AppHandle) -> Result<Vec<RecentRepo>, String> {
    let path = recents_file(&app)?;
    if !path.exists() {
        return Ok(vec![]);
    }
    let data = fs::read_to_string(&path).map_err(|e| format!("read recents failed: {e}"))?;
    serde_json::from_str(&data).map_err(|e| format!("parse recents failed: {e}"))
}

#[tauri::command]
fn recents_save(app: tauri::AppHandle, recents: Vec<RecentRepo>) -> Result<(), String> {
    let path = recents_file(&app)?;
    if let Some(parent) = path.parent() {
        fs::create_dir_all(parent).map_err(|e| format!("create dir failed: {e}"))?;
    }
    let data =
        serde_json::to_string_pretty(&recents).map_err(|e| format!("serialize failed: {e}"))?;
    fs::write(&path, data).map_err(|e| format!("write recents failed: {e}"))?;
    Ok(())
}

// ── entry point ───────────────────────────────────────────────────────────────

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .manage(ActiveSidecar(Mutex::new(HashMap::new())))
        .plugin(tauri_plugin_dialog::init())
        .plugin(tauri_plugin_opener::init())
        .plugin(tauri_plugin_shell::init())
        .invoke_handler(tauri::generate_handler![
            sidecar_call,
            sidecar_cancel,
            recents_load,
            recents_save,
        ])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
