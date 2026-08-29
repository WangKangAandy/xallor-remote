"""把本机交叉编译的 Linux Relay 二进制放到 Xallor 生产 ECS。

只上传二进制 + systemd 单元。不碰 /opt/xallor、不传 .env、不改网关证书。
进程只听 127.0.0.1:8443；公网 WSS 要等域名解析和证书后再挂 Nginx。

环境与 Xallor 相同：~/.ssh/xallor_ecs_ed25519 或 XALLOR_ECS_PASSWORD。
"""
from __future__ import annotations

import sys
import time
from pathlib import Path

if hasattr(sys.stdout, "reconfigure"):
    sys.stdout.reconfigure(encoding="utf-8", errors="replace")
    sys.stderr.reconfigure(encoding="utf-8", errors="replace")

ROOT = Path(__file__).resolve().parents[1]
BIN = ROOT / "dist" / "xallor-remote-linux-amd64"
REMOTE_TMP = "/opt/tmp-xallor-remote"
UNIT = """[Unit]
Description=XallorRemote Relay
After=network.target

[Service]
Type=simple
ExecStart=/opt/xallor-remote/bin/xallor-remote relay --listen 0.0.0.0:8443 --data /var/lib/xallor-remote --quota
Restart=on-failure
RestartSec=2

[Install]
WantedBy=multi-user.target
"""

REMOTE = r"""
set -euo pipefail
install -d -m 0755 /opt/xallor-remote/bin /var/lib/xallor-remote
if [ -f /opt/xallor-remote/bin/xallor-remote ]; then
  cp -a /opt/xallor-remote/bin/xallor-remote "/opt/xallor-remote/bin/xallor-remote.bak.$(date +%Y%m%d%H%M%S)"
fi
mv /opt/tmp-xallor-remote /opt/xallor-remote/bin/xallor-remote
chmod 0755 /opt/xallor-remote/bin/xallor-remote
file /opt/xallor-remote/bin/xallor-remote
/opt/xallor-remote/bin/xallor-remote --help >/dev/null
cat >/etc/systemd/system/xallor-remote-relay.service <<'EOF'
""" + UNIT + r"""EOF
systemctl daemon-reload
systemctl enable xallor-remote-relay
systemctl restart xallor-remote-relay
sleep 1
systemctl is-active xallor-remote-relay
ss -lnt | grep 8443 || true
code=$(curl -sS -o /dev/null -w '%{http_code}' \
  -H 'Connection: Upgrade' -H 'Upgrade: websocket' \
  -H 'Sec-WebSocket-Version: 13' -H 'Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==' \
  http://127.0.0.1:8443/ || echo fail)
echo "ws_upgrade:$code"
echo '=== deploy done ==='
"""

sys.path.insert(0, str(Path.home() / "Desktop" / "repo" / "xallor" / "scripts"))
from ecs_ssh import HOST, connect_ecs  # noqa: E402


def main() -> None:
    if not BIN.is_file():
        raise SystemExit(f"missing {BIN} — first: GOOS=linux GOARCH=amd64 go build")
    head = BIN.read_bytes()[:4]
    if head != b"\x7fELF":
        raise SystemExit(f"{BIN} is not a Linux ELF binary")
    print(f"[deploy-relay] {BIN} ({BIN.stat().st_size} bytes) -> {HOST}", flush=True)
    client = connect_ecs()
    try:
        sftp = client.open_sftp()
        sftp.put(str(BIN), REMOTE_TMP)
        sftp.close()
        print("[deploy-relay] uploaded, installing…", flush=True)
        _, stdout, stderr = client.exec_command(REMOTE, get_pty=False, timeout=120)
        channel = stdout.channel
        while True:
            if channel.recv_ready():
                sys.stdout.write(channel.recv(4096).decode("utf-8", "replace"))
                sys.stdout.flush()
            if channel.recv_stderr_ready():
                sys.stderr.write(channel.recv_stderr(4096).decode("utf-8", "replace"))
                sys.stderr.flush()
            if channel.exit_status_ready() and not channel.recv_ready() and not channel.recv_stderr_ready():
                break
            time.sleep(0.2)
        while channel.recv_ready():
            sys.stdout.write(channel.recv(4096).decode("utf-8", "replace"))
        while channel.recv_stderr_ready():
            sys.stderr.write(channel.recv_stderr(4096).decode("utf-8", "replace"))
        code = channel.recv_exit_status()
        if code != 0:
            sys.exit(code)
    finally:
        client.close()
    print("[deploy-relay] OK", flush=True)


if __name__ == "__main__":
    main()
