use std::env;
use std::path::PathBuf;
use std::process::Command;

#[cfg(windows)]
use std::os::windows::process::CommandExt;

const CREATE_NO_WINDOW: u32 = 0x0800_0000;

fn exe_name() -> &'static str {
    if cfg!(windows) {
        "xallor-remote.exe"
    } else {
        "xallor-remote"
    }
}

fn data_dir() -> PathBuf {
    if cfg!(windows) {
        if let Ok(base) = env::var("APPDATA") {
            return PathBuf::from(base).join("XallorRemote");
        }
        let home = env::var("USERPROFILE").unwrap_or_default();
        return PathBuf::from(home).join("AppData").join("Roaming").join("XallorRemote");
    }
    if let Ok(xdg) = env::var("XDG_CONFIG_HOME") {
        return PathBuf::from(xdg).join("xallor-remote");
    }
    let home = env::var("HOME").unwrap_or_default();
    PathBuf::from(home).join(".config").join("xallor-remote")
}

fn look_in_path() -> Option<PathBuf> {
    let name = exe_name();
    let path = env::var_os("PATH")?;
    for dir in env::split_paths(&path) {
        let cand = dir.join(name);
        if cand.is_file() {
            return Some(cand);
        }
    }
    None
}

fn find_bin() -> Result<PathBuf, String> {
    if let Ok(p) = env::var("XALLOR_REMOTE_BIN") {
        if !p.is_empty() {
            return Ok(PathBuf::from(p));
        }
    }
    if let Some(p) = look_in_path() {
        return Ok(p);
    }
    let bundled = data_dir().join("bin").join(exe_name());
    if bundled.is_file() {
        return Ok(bundled);
    }
    Err("找不到 xallor-remote。请先安装后再打开。".into())
}

pub fn ensure() -> Result<(), String> {
    if crate::ipc::try_connect().is_ok() {
        return Ok(());
    }
    let bin = find_bin()?;
    let mut cmd = Command::new(&bin);
    cmd.arg("ensure");
    #[cfg(windows)]
    {
        cmd.creation_flags(CREATE_NO_WINDOW);
    }
    let status = cmd.status().map_err(|_| "无法拉起 Runtime。".to_string())?;
    if !status.success() {
        return Err("无法拉起 Runtime。".into());
    }
    Ok(())
}
