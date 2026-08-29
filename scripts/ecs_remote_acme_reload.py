"""Add remote.xallor.com :80 ACME vhost. Reload nginx only. No compose recreate, no .env rewrite."""
from __future__ import annotations

import sys
from pathlib import Path

if hasattr(sys.stdout, "reconfigure"):
    sys.stdout.reconfigure(encoding="utf-8", errors="replace")

XALLOR = Path.home() / "Desktop" / "repo" / "xallor"
SRC = XALLOR / "deploy" / "gateway" / "remote-acme.conf.template"

sys.path.insert(0, str(XALLOR / "scripts"))
from ecs_ssh import connect_ecs  # noqa: E402

REMOTE = r"""
set -euo pipefail
install -d /opt/xallor/deploy/gateway
cp -a /opt/tmp-remote-acme.conf /opt/xallor/deploy/gateway/remote-acme.conf.template
docker cp /opt/tmp-remote-acme.conf xallor-api_gateway_1:/etc/nginx/conf.d/remote-acme.conf
docker exec xallor-api_gateway_1 nginx -t
docker exec xallor-api_gateway_1 nginx -s reload
rm -f /opt/tmp-remote-acme.conf
echo '=== server_name ==='
docker exec xallor-api_gateway_1 sh -c 'grep -R server_name /etc/nginx/conf.d | grep -v template'
curl -sS -o /dev/null -w 'api:%{http_code}\n' https://api.xallor.com/api/health
curl -sS -o /dev/null -w 'tab:%{http_code}\n' https://tab.xallor.com/
"""

if not SRC.is_file():
    raise SystemExit(f"missing {SRC}")

client = connect_ecs()
try:
    sftp = client.open_sftp()
    sftp.put(str(SRC), "/opt/tmp-remote-acme.conf")
    sftp.close()
    _, stdout, stderr = client.exec_command(REMOTE, get_pty=False, timeout=60)
    sys.stdout.write(stdout.read().decode("utf-8", "replace"))
    sys.stderr.write(stderr.read().decode("utf-8", "replace"))
    sys.exit(stdout.channel.recv_exit_status())
finally:
    client.close()
