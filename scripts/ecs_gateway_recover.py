"""Recover / finish api.xallor.com gateway after compose 1.29 ContainerConfig error."""
from __future__ import annotations

import sys
from pathlib import Path

if hasattr(sys.stdout, "reconfigure"):
    sys.stdout.reconfigure(encoding="utf-8", errors="replace")
    sys.stderr.reconfigure(encoding="utf-8", errors="replace")

sys.path.insert(0, str(Path.home() / "Desktop" / "repo" / "xallor" / "scripts"))
from ecs_ssh import connect_ecs  # noqa: E402

REMOTE = r"""
set -euo pipefail
cd /opt/xallor
echo '=== gateway containers ==='
docker ps -a --filter name=gateway --format '{{.Names}} {{.Status}}'
echo '=== public 80/443 ==='
ss -lnt | grep -E ':80 |:443 ' || true
echo '=== api health now ==='
curl -sSk -o /dev/null -w 'api:%{http_code}\n' https://api.xallor.com/api/health || echo api:fail

COMPOSE='docker-compose -p xallor-api -f docker-compose.yml -f docker-compose.gateway.yml'
if [ -f docker-compose.gateway.tab-https.yml ]; then
  COMPOSE="$COMPOSE -f docker-compose.gateway.tab-https.yml"
fi
if [ -f docker-compose.gateway.admin-https.yml ]; then
  COMPOSE="$COMPOSE -f docker-compose.gateway.admin-https.yml"
fi
if [ -f docker-compose.gateway.xallor-remote.yml ]; then
  COMPOSE="$COMPOSE -f docker-compose.gateway.xallor-remote.yml"
fi

echo '=== rm + up gateway ==='
docker rm -f xallor-api_gateway_1 2>/dev/null || true
$COMPOSE up -d --no-deps gateway

echo '=== wait ==='
for i in $(seq 1 20); do
  code=$(curl -sSk -o /dev/null -w '%{http_code}' https://api.xallor.com/api/health || echo fail)
  echo "try$i:$code"
  if [ "$code" = "200" ]; then break; fi
  sleep 2
done
curl -sSk -w '\nxr:%{http_code}\n' https://api.xallor.com/xr/health || true
docker inspect xallor-api_gateway_1 --format 'extra={{json .HostConfig.ExtraHosts}}'
docker exec xallor-api_gateway_1 sh -c 'grep -n xr /etc/nginx/conf.d/default.conf | head'
"""

client = connect_ecs()
try:
    _, stdout, stderr = client.exec_command(REMOTE, get_pty=False, timeout=180)
    sys.stdout.write(stdout.read().decode("utf-8", "replace"))
    err = stderr.read().decode("utf-8", "replace")
    if err.strip():
        sys.stderr.write(err)
    sys.exit(stdout.channel.recv_exit_status())
finally:
    client.close()
