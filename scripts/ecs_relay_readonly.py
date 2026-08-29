"""Read-only: what's on the Xallor ECS that would host Relay. No writes."""
from __future__ import annotations

import sys
from pathlib import Path

if hasattr(sys.stdout, "reconfigure"):
    sys.stdout.reconfigure(encoding="utf-8", errors="replace")
    sys.stderr.reconfigure(encoding="utf-8", errors="replace")

sys.path.insert(0, str(Path.home() / "Desktop" / "repo" / "xallor" / "scripts"))
from ecs_ssh import connect_ecs  # noqa: E402

REMOTE = r"""
set +e
echo '=== host ==='
hostname; uname -a
echo '=== listen 80/443/8443 ==='
ss -lnt | grep -E ':80 |:443 |:8443 |:18443 ' || true
echo '=== processes xallor-remote / relay ==='
ps aux | grep -E 'xallor-remote|relay' | grep -v grep || true
echo '=== systemd ==='
systemctl list-units --type=service --all 2>/dev/null | grep -Ei 'xallor|relay' || true
echo '=== dirs ==='
ls -ld /opt/xallor /opt/xallor-remote /opt/xallorremote /data/xallor-remote 2>/dev/null || true
echo '=== docker ==='
docker ps --format '{{.Names}}\t{{.Status}}\t{{.Ports}}' 2>/dev/null || true
echo '=== nginx server_name (container) ==='
gw=$(docker ps --format '{{.Names}}' | grep -E 'gateway' | head -n1)
echo "gw=$gw"
if [ -n "$gw" ]; then
  docker exec "$gw" sh -c 'grep -R "server_name" /etc/nginx/conf.d /etc/nginx/templates 2>/dev/null | head -n 80' || true
fi
echo '=== certs (names only) ==='
ls /etc/letsencrypt/live 2>/dev/null || true
echo '=== compose files ==='
ls /opt/xallor/docker-compose*.yml 2>/dev/null || true
echo '=== done ==='
"""

client = connect_ecs()
try:
    _, stdout, stderr = client.exec_command(REMOTE, get_pty=False, timeout=60)
    sys.stdout.write(stdout.read().decode("utf-8", "replace"))
    err = stderr.read().decode("utf-8", "replace")
    if err.strip():
        sys.stderr.write(err)
    code = stdout.channel.recv_exit_status()
    sys.exit(code)
finally:
    client.close()
