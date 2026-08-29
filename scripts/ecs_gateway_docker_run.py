"""Start gateway with docker run (compose 1.29 ContainerConfig workaround). Does not print .env values."""
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

echo '=== leftover gateway ==='
docker ps -a --filter name=gateway --format '{{.ID}} {{.Names}} {{.Status}}'

echo '=== inspect binds (paths only) ==='
old=$(docker ps -aq --filter name=gateway | head -n1)
if [ -n "$old" ]; then
  docker inspect "$old" --format '{{range .HostConfig.Binds}}{{println .}}{{end}}'
  docker inspect "$old" --format '{{range $k,$v := .NetworkSettings.Networks}}net={{$k}}{{end}}'
  docker inspect "$old" --format '{{range .Config.Env}}{{println .}}{{end}}' | grep -E '^GATEWAY_' || true
fi

set +e
docker ps -aq --filter name=gateway | xargs -r docker rm -f
set -e

# 从 .env 读键，不打印值
eval "$(grep -E '^(GATEWAY_SERVER_NAME|GATEWAY_TLS_CERT_DIR|GATEWAY_TLS_CERT_FILE|GATEWAY_TLS_KEY_FILE|GATEWAY_TAB_TLS_CERT_DIR|GATEWAY_ADMIN_TLS_CERT_DIR)=' .env | sed 's/\r$//')"
net=$(docker network ls --format '{{.Name}}' | grep -E '^xallor-api' | head -n1)
echo "net=$net server_name_set=${GATEWAY_SERVER_NAME:+yes} cert_dir_set=${GATEWAY_TLS_CERT_DIR:+yes}"

docker run -d --name xallor-api_gateway_1 --restart unless-stopped \
  --network "$net" --network-alias gateway \
  --add-host host.docker.internal:host-gateway \
  -p 80:80 -p 443:443 \
  -e "GATEWAY_SERVER_NAME=${GATEWAY_SERVER_NAME}" \
  -e "GATEWAY_TLS_CERT_FILE=${GATEWAY_TLS_CERT_FILE:-fullchain.pem}" \
  -e "GATEWAY_TLS_KEY_FILE=${GATEWAY_TLS_KEY_FILE:-privkey.pem}" \
  -v /opt/xallor/deploy/gateway/default.conf.template:/etc/nginx/templates/default.conf.template:ro \
  -v /opt/xallor/deploy/gateway/tab-site-https.conf.template:/etc/nginx/templates/tab-site-https.conf.template:ro \
  -v /opt/xallor/deploy/gateway/admin-site-https.conf.template:/etc/nginx/templates/admin-site-https.conf.template:ro \
  -v /var/www/certbot:/var/www/certbot:ro \
  -v "${GATEWAY_TLS_CERT_DIR}:/etc/nginx/certs:ro" \
  -v "${GATEWAY_TAB_TLS_CERT_DIR}:/etc/nginx/certs-tab:ro" \
  -v "${GATEWAY_ADMIN_TLS_CERT_DIR}:/etc/nginx/certs-admin:ro" \
  nginx:1.27-alpine

echo '=== wait ==='
for i in $(seq 1 20); do
  code=$(curl -sSk -o /dev/null -w '%{http_code}' https://127.0.0.1/api/health || echo fail)
  echo "local_api try$i:$code"
  if [ "$code" = "200" ]; then break; fi
  sleep 2
done
curl -sSk -o /dev/null -w 'tab:%{http_code}\n' https://tab.xallor.com/ || true
curl -sSk -o /dev/null -w 'admin:%{http_code}\n' https://admin.xallor.com/login || true
curl -sSk -o /dev/null -w 'api:%{http_code}\n' https://api.xallor.com/api/health || true
echo -n 'xr:'
curl -sSk -w '%{http_code}\n' https://api.xallor.com/xr/health || true
docker exec xallor-api_gateway_1 nginx -t
docker exec xallor-api_gateway_1 sh -c 'grep -n "xr" /etc/nginx/conf.d/default.conf | head'
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
