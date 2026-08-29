"""把 api.xallor.com/xr 接到宿主机 Relay。不改 /opt/xallor/.env，不重建 api。"""
from __future__ import annotations

import sys
import time
from pathlib import Path

if hasattr(sys.stdout, "reconfigure"):
    sys.stdout.reconfigure(encoding="utf-8", errors="replace")
    sys.stderr.reconfigure(encoding="utf-8", errors="replace")

XALLOR = Path.home() / "Desktop" / "repo" / "xallor"
FILES = {
    "deploy/gateway/default.conf.template": XALLOR / "deploy" / "gateway" / "default.conf.template",
    "docker-compose.gateway.xallor-remote.yml": XALLOR / "docker-compose.gateway.xallor-remote.yml",
}

REMOTE = r"""
set -euo pipefail
cd /opt/xallor
ts=$(date +%Y%m%d%H%M%S)
cp -a deploy/gateway/default.conf.template "deploy/gateway/default.conf.template.bak.xr.$ts"
cp -a /opt/tmp-xr-gateway/default.conf.template deploy/gateway/default.conf.template
cp -a /opt/tmp-xr-gateway/docker-compose.gateway.xallor-remote.yml docker-compose.gateway.xallor-remote.yml
rm -rf /opt/tmp-xr-gateway

COMPOSE='docker-compose -p xallor-api -f docker-compose.yml -f docker-compose.gateway.yml'
if [ -f docker-compose.gateway.tab-https.yml ]; then
  COMPOSE="$COMPOSE -f docker-compose.gateway.tab-https.yml"
fi
if [ -f docker-compose.gateway.admin-https.yml ]; then
  COMPOSE="$COMPOSE -f docker-compose.gateway.admin-https.yml"
fi
COMPOSE="$COMPOSE -f docker-compose.gateway.xallor-remote.yml"

echo '=== recreate gateway ==='
$COMPOSE up -d --no-deps --force-recreate gateway

echo '=== wait nginx ==='
for i in $(seq 1 20); do
  code=$(curl -sSk -o /dev/null -w '%{http_code}' https://api.xallor.com/api/health || echo fail)
  echo "api_health try$i:$code"
  if [ "$code" = "200" ]; then break; fi
  sleep 2
done
if [ "$code" != "200" ]; then
  echo 'ROLLBACK: api health not 200' >&2
  latest=$(ls -1t deploy/gateway/default.conf.template.bak.xr.* | head -1)
  cp -a "$latest" deploy/gateway/default.conf.template
  $COMPOSE up -d --no-deps --force-recreate gateway
  exit 4
fi

echo '=== xr health ==='
curl -sSk -w '\nhttp:%{http_code}\n' https://api.xallor.com/xr/health
echo '=== gateway extra_hosts ==='
docker inspect xallor-api_gateway_1 --format '{{json .HostConfig.ExtraHosts}}'
echo '=== apply done ==='
"""

sys.path.insert(0, str(Path.home() / "Desktop" / "repo" / "xallor" / "scripts"))
from ecs_ssh import connect_ecs  # noqa: E402


def main() -> None:
    missing = [str(p) for p in FILES.values() if not p.is_file()]
    if missing:
        raise SystemExit("missing " + ", ".join(missing))
    client = connect_ecs()
    try:
        sftp = client.open_sftp()
        try:
            sftp.mkdir("/opt/tmp-xr-gateway")
        except OSError:
            pass
        for name, src in FILES.items():
            sftp.put(str(src), "/opt/tmp-xr-gateway/" + Path(name).name)
        sftp.close()
        print("[apply-gateway] uploaded, recreating gateway…", flush=True)
        _, stdout, stderr = client.exec_command(REMOTE, get_pty=False, timeout=180)
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
    print("[apply-gateway] OK", flush=True)


if __name__ == "__main__":
    main()
