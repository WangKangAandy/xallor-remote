"""Read-only: can this ECS add remote.xallor.com? Do not print secrets."""
from __future__ import annotations

import sys
from pathlib import Path

if hasattr(sys.stdout, "reconfigure"):
    sys.stdout.reconfigure(encoding="utf-8", errors="replace")

sys.path.insert(0, str(Path.home() / "Desktop" / "repo" / "xallor" / "scripts"))
from ecs_ssh import connect_ecs  # noqa: E402

REMOTE = r"""
set +e
echo '=== tools ==='
command -v aliyun && echo HAS_ALIYUN || echo NO_ALIYUN
command -v certbot && echo HAS_CERTBOT || echo NO_CERTBOT
echo '=== secret files exist (no content) ==='
for p in /root/.secrets/aliyun-dns.ini /root/.aliyun/config.json /opt/xallor/.secrets/aliyun-dns.ini; do
  if [ -f "$p" ]; then echo "exists $p"; else echo "missing $p"; fi
done
echo '=== live certs ==='
ls /etc/letsencrypt/live 2>/dev/null || true
echo '=== resolve from ecs ==='
getent hosts remote.xallor.com api.xallor.com || true
"""

client = connect_ecs()
try:
    _, stdout, stderr = client.exec_command(REMOTE, get_pty=False, timeout=30)
    sys.stdout.write(stdout.read().decode("utf-8", "replace"))
    sys.stderr.write(stderr.read().decode("utf-8", "replace"))
    sys.exit(stdout.channel.recv_exit_status())
finally:
    client.close()
