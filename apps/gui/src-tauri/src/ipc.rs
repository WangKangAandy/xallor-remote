use std::io::{BufRead, BufReader, Write};
use std::sync::atomic::{AtomicU64, Ordering};

static NEXT_ID: AtomicU64 = AtomicU64::new(1);

fn next_id() -> String {
    format!("g{}", NEXT_ID.fetch_add(1, Ordering::Relaxed))
}

#[cfg(windows)]
type IpcStream = std::fs::File;

#[cfg(unix)]
type IpcStream = std::os::unix::net::UnixStream;

fn ipc_path() -> String {
    if cfg!(windows) {
        return r"\\.\pipe\XallorRemote".into();
    }
    if let Ok(dir) = std::env::var("XDG_RUNTIME_DIR") {
        if !dir.is_empty() {
            return format!("{dir}/xallor-remote.sock");
        }
    }
    let home = std::env::var("HOME").unwrap_or_default();
    if let Ok(xdg) = std::env::var("XDG_CONFIG_HOME") {
        if !xdg.is_empty() {
            return format!("{xdg}/xallor-remote/ipc.sock");
        }
    }
    format!("{home}/.config/xallor-remote/ipc.sock")
}

#[cfg(windows)]
fn open_ipc() -> std::io::Result<IpcStream> {
    std::fs::OpenOptions::new()
        .read(true)
        .write(true)
        .open(ipc_path())
}

#[cfg(unix)]
fn open_ipc() -> std::io::Result<IpcStream> {
    std::os::unix::net::UnixStream::connect(ipc_path())
}

pub fn try_connect() -> Result<IpcStream, String> {
    open_ipc().map_err(|_| "Runtime 未运行。".to_string())
}

fn connect() -> Result<IpcStream, String> {
    match try_connect() {
        Ok(s) => Ok(s),
        Err(_) => {
            crate::ensure::ensure()?;
            try_connect()
        }
    }
}

fn write_req(stream: &mut IpcStream, id: &str, method: &str, params: serde_json::Value) -> Result<(), String> {
    let line = serde_json::json!({"id": id, "method": method, "params": params});
    let mut raw = serde_json::to_vec(&line).map_err(|_| "失败。".to_string())?;
    raw.push(b'\n');
    stream.write_all(&raw).map_err(|_| "Runtime 未运行。".to_string())?;
    stream.flush().map_err(|_| "Runtime 未运行。".to_string())?;
    Ok(())
}

fn read_frames<F>(stream: IpcStream, id: &str, mut on_event: F) -> Result<serde_json::Value, String>
where
    F: FnMut(&str, &str),
{
    let mut reader = BufReader::new(stream);
    let mut line = String::new();
    loop {
        line.clear();
        let n = reader.read_line(&mut line).map_err(|_| "连接已断开。".to_string())?;
        if n == 0 {
            return Err("连接已断开。".into());
        }
        let raw = line.trim();
        if raw.is_empty() {
            continue;
        }
        let f: serde_json::Value = serde_json::from_str(raw).map_err(|_| "失败。".to_string())?;
        if f.get("id").and_then(|v| v.as_str()) != Some(id) {
            continue;
        }
        if let Some(ev) = f.get("event").and_then(|v| v.as_str()) {
            let data = f.get("data").and_then(|v| v.as_str()).unwrap_or("");
            on_event(ev, data);
            continue;
        }
        if f.get("ok") == Some(&serde_json::Value::Bool(false)) {
            let msg = f
                .get("message")
                .and_then(|v| v.as_str())
                .unwrap_or("失败。");
            return Err(msg.to_string());
        }
        return Ok(f.get("result").cloned().unwrap_or(serde_json::json!({})));
    }
}

pub fn rpc(method: &str, params: serde_json::Value) -> Result<serde_json::Value, String> {
    let mut stream = connect()?;
    let id = next_id();
    write_req(&mut stream, &id, method, params)?;
    read_frames(stream, &id, |_, _| {})
}

pub fn exec_stream(command: &str, device_id: &str, mut write: impl FnMut(&str)) -> Result<(), String> {
    let mut stream = connect()?;
    let id = next_id();
    let mut params = serde_json::json!({"command": command});
    if !device_id.is_empty() {
        params["device_id"] = serde_json::Value::String(device_id.to_string());
    }
    write_req(&mut stream, &id, "exec", params)?;
    read_frames(stream, &id, |ev, data| {
        if ev == "stdout" || ev == "stderr" {
            write(data);
        }
    })?;
    Ok(())
}
