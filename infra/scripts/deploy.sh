#!/usr/bin/env bash
set -euo pipefail

# ====== 必要変数（環境に合わせて変更）======
SSH_USER="ec2-user"
SSH_HOST="ec2-xxx-xxx-xxx-xxx.ap-northeast-1.compute.amazonaws.com"  # ← ここを埋めてください: EC2 Public DNS
SSH_PORT="22"
SSH_KEY="${HOME}/.ssh/myslack-key2.pem"                               # ← ここを埋めてください: 秘密鍵パス

# EIPに合わせた nip.io FQDN
DOMAIN="api.203.0.113.42.nip.io"  # ← ここを埋めてください: 例) api.<EIP>.nip.io

# GHCR
GHCR_USER="your-ghcr-username"                                       # ← ここを埋めてください
GHCR_TOKEN="pAT_1xA7fQm2zY9vK4hR8nE3sW6tB0cD5"                        # ← ここを埋めてください（GitHubに上げない！）
API_IMAGE="ghcr.io/your-ghcr-username/myslack-api:latest"            # ← ここを埋めてください: イメージ名

# ====== .env（必要に応じて編集）======
DOT_ENV="$(cat <<'EOF'
# ===== API =====
PORT=8000
DB_URL=postgres://app_user:AppP@ss-9kQ@postgres:5432/appdb           # ← ここを埋めてください（ユーザ/パス）
DATABASE_URL=postgres://app_user:AppP@ss-9kQ@postgres:5432/appdb      # ← ここを埋めてください（ユーザ/パス）
JWT_SECRET=K2rX8pQ1uV7mZ4sT0bN9wH3cF6yA5eD1                           # ← ここを埋めてください（十分長い乱数）

BIND_ADDR=0.0.0.0:8000

# ===== Auth0 =====
AUTH0_DOMAIN=your-tenant.us.auth0.com                                 # ← ここを埋めてください
AUTH0_AUDIENCE=https://api.example.com                                 # ← ここを埋めてください（API識別子）

# ===== Postgres =====
PGHOST=postgres
PGUSER=app_user                                                        # ← ここを埋めてください
PGPASSWORD=AppP@ss-9kQ                                                 # ← ここを埋めてください（DBパス）
PGDATABASE=appdb                                                       # ← ここを埋めてください
PGPORT=5432

# ===== S3 =====
S3_PROVIDER=aws
AWS_REGION=ap-northeast-1                                              # ← 必要なら変更
S3_BUCKET=myslack-web-xxxxxxxxxxxx                                     # ← ここを埋めてください（バケット名）
S3_PREFIX=uploads                                                      # ← 必要なら変更
S3_URL_EXPIRY_SEC=3600
S3_USE_PATH_STYLE=false

# ===== CORS / WebSocket =====
CORS_ALLOW_ORIGINS=https://dxxxxxxxxxxxx.cloudfront.net                # ← ここを埋めてください（Webのオリジン）
WS_ALLOWED_ORIGIN=https://dxxxxxxxxxxxx.cloudfront.net                 # ← ここを埋めてください（WSのオリジン）
EOF
)"

# ====== 事前チェック ======
[[ -n "${GHCR_TOKEN}" ]] || { echo "ERROR: export GHCR_TOKEN=*** してから実行してね"; exit 1; }
[[ -f "${SSH_KEY}" ]] || { echo "ERROR: SSH鍵がない: ${SSH_KEY}"; exit 1; }
chmod 600 "${SSH_KEY}"

TMP_DIR="$(mktemp -d)"; trap 'rm -rf "${TMP_DIR}"' EXIT

# ====== リモートで実行するブートストラップ ======
cat > "${TMP_DIR}/remote_bootstrap.sh" <<"RSCRIPT"
#!/usr/bin/env bash
set -euo pipefail

DOMAIN="$1"
GHCR_USER="$2"
GHCR_TOKEN="$3"
API_IMAGE="$4"

APP_DIR="/opt/myslack"
NGINX_DIR="${APP_DIR}/nginx"
RENDERED_DIR="${NGINX_DIR}/rendered"
CERTBOT_WEBROOT="/var/www/certbot"

SUDO="sudo"; if [[ "$(id -u)" -eq 0 ]]; then SUDO=""; fi

# ---- OS別インストール（AL2023はdnf）※curlは入れない（curl-minimalがある）
if command -v dnf >/dev/null 2>&1; then
  ${SUDO} dnf -y update
  ${SUDO} dnf install -y docker nginx cronie
  ${SUDO} systemctl enable --now crond
elif command -v yum >/dev/null 2>&1; then
  ${SUDO} yum -y update
  ${SUDO} yum install -y docker nginx cronie
  ${SUDO} systemctl enable --now crond
elif command -v apt-get >/dev/null 2>&1; then
  ${SUDO} apt-get update -y
  ${SUDO} apt-get install -y docker.io nginx cron
  ${SUDO} systemctl enable --now cron
else
  echo "Unsupported OS" >&2; exit 1
fi

${SUDO} systemctl enable --now docker
${SUDO} systemctl disable --now nginx || true

# ---- docker-compose v2 単体バイナリ（curl-minimalでDL）
if ! /usr/local/bin/docker-compose version >/dev/null 2>&1; then
  COMPOSE_URL="https://github.com/docker/compose/releases/download/v2.29.7/docker-compose-$(uname -s)-$(uname -m)"
  ${SUDO} curl -fsSL "${COMPOSE_URL}" -o /usr/local/bin/docker-compose
  ${SUDO} chmod +x /usr/local/bin/docker-compose
  ${SUDO} ln -sf /usr/local/bin/docker-compose /usr/bin/docker-compose || true
fi

# ---- ディレクトリ
${SUDO} mkdir -p "${APP_DIR}/docker" "${RENDERED_DIR}" /etc/letsencrypt "${CERTBOT_WEBROOT}"
${SUDO} chown -R "$(id -u)":"$(id -g)" "${APP_DIR}" "${CERTBOT_WEBROOT}"
${SUDO} chown -R root:root /etc/letsencrypt

# ---- docker-compose.yml
cat > "${APP_DIR}/docker/docker-compose.yml" <<'YAML'
services:
  postgres:
    image: postgres:16
    environment:
      POSTGRES_USER: app_user
      POSTGRES_PASSWORD: AppP@ss-9kQ
      POSTGRES_DB: appdb
    volumes:
      - pgdata:/var/lib/postgresql/data
    restart: unless-stopped

  api:
    image: __API_IMAGE__
    env_file:
      - .env
    depends_on:
      - postgres
    expose:
      - "8000"
    restart: unless-stopped

  nginx:
    image: nginx:alpine
    volumes:
      - ../nginx/rendered/api.conf:/etc/nginx/conf.d/api.conf:ro
      - /etc/letsencrypt:/etc/letsencrypt
      - /var/www/certbot:/var/www/certbot
    ports:
      - "80:80"
      - "443:443"
    depends_on:
      - api
    restart: unless-stopped

volumes:
  pgdata:
YAML
sed -i "s#__API_IMAGE__#${API_IMAGE}#g" "${APP_DIR}/docker/docker-compose.yml"

# ---- Nginx bootstrap(80のみ)
cat > "${RENDERED_DIR}/api_bootstrap.conf" <<NGX
server {
  listen 80;
  server_name ${DOMAIN};

  location ^~ /.well-known/acme-challenge/ {
    root ${CERTBOT_WEBROOT};
    default_type "text/plain";
    try_files \$uri =404;
  }

  location / {
    return 200 'ok';
    add_header Content-Type text/plain;
  }
}
NGX

# ---- Nginx full(80→443 & TLS+リバプロ)
cat > "${RENDERED_DIR}/api_full.conf" <<NGX
server {
  listen 80;
  server_name ${DOMAIN};

  location ^~ /.well-known/acme-challenge/ {
    root ${CERTBOT_WEBROOT};
    default_type "text/plain";
    try_files \$uri =404;
  }

  return 301 https://\$host\$request_uri;
}

server {
  listen 443 ssl;
  http2 on;
  server_name ${DOMAIN};

  ssl_certificate     /etc/letsencrypt/live/${DOMAIN}/fullchain.pem;
  ssl_certificate_key /etc/letsencrypt/live/${DOMAIN}/privkey.pem;

  client_max_body_size 50m;
  proxy_read_timeout   300s;
  proxy_send_timeout   300s;

  location / {
    proxy_pass http://api:8000;
    proxy_http_version 1.1;

    # WebSocket
    proxy_set_header Upgrade \$http_upgrade;
    proxy_set_header Connection "upgrade";

    # 通常ヘッダ
    proxy_set_header Host \$host;
    proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto \$scheme;
  }
}
NGX

# ---- 最初はbootstrapを有効化して起動（80だけ開く）
cp -f "${RENDERED_DIR}/api_bootstrap.conf" "${RENDERED_DIR}/api.conf"

# GHCR login
echo "${GHCR_TOKEN}" | ${SUDO} docker login ghcr.io -u "${GHCR_USER}" --password-stdin

# コンテナ起動（sudo + 絶対パス）
( cd "${APP_DIR}/docker" \
  && ${SUDO} /usr/local/bin/docker-compose pull \
  && ${SUDO} /usr/local/bin/docker-compose up -d --force-recreate )

# HTTP到達確認（失敗しても続行）
sleep 2
curl -sf "http://${DOMAIN}/.well-known/acme-challenge/does-not-exist" || true

# ---- 証明書取得（webroot、未取得のときだけ）
if [[ ! -f "/etc/letsencrypt/live/${DOMAIN}/fullchain.pem" ]]; then
  ${SUDO} docker run --rm \
    -v /etc/letsencrypt:/etc/letsencrypt \
    -v ${CERTBOT_WEBROOT}:${CERTBOT_WEBROOT} \
    certbot/certbot certonly \
      --non-interactive --webroot -w ${CERTBOT_WEBROOT} \
      -d "${DOMAIN}" -m admin@example.com --agree-tos --no-eff-email  # ← ここを埋めてください（連絡先メール）
fi

# ---- フル設定へ切替 → reload（失敗時はrestart）
cp -f "${RENDERED_DIR}/api_full.conf" "${RENDERED_DIR}/api.conf"
${SUDO} /usr/local/bin/docker-compose -f "${APP_DIR}/docker/docker-compose.yml" exec nginx nginx -s reload \
  || ${SUDO} /usr/local/bin/docker-compose -f "${APP_DIR}/docker/docker-compose.yml" restart nginx

# ---- renew cron を /etc/cron.d にrootジョブとして配置
${SUDO} bash -c 'cat >/etc/cron.d/myslack-certbot <<EOF
SHELL=/bin/bash
PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
0 3,15 * * * root /usr/bin/docker run --rm -v /etc/letsencrypt:/etc/letsencrypt -v '"${CERTBOT_WEBROOT}"':'"${CERTBOT_WEBROOT}"' certbot/certbot renew && /usr/local/bin/docker-compose -f '"${APP_DIR}"'/docker/docker-compose.yml exec nginx nginx -s reload
EOF'
${SUDO} chmod 644 /etc/cron.d/myslack-certbot
${SUDO} systemctl reload crond || ${SUDO} systemctl restart crond || true

echo "✅ Bootstrap finished. Try: curl -vk https://${DOMAIN}/"
RSCRIPT

# ====== .env を作成 ======
echo "${DOT_ENV}" > "${TMP_DIR}/.env"

# ====== 転送 & 実行 ======
SSH_OPTS="-i ${SSH_KEY} -o StrictHostKeyChecking=accept-new"
scp ${SSH_OPTS} -P "${SSH_PORT}" -q "${TMP_DIR}/remote_bootstrap.sh" "${SSH_USER}@${SSH_HOST}:/tmp/remote_bootstrap.sh"
scp ${SSH_OPTS} -P "${SSH_PORT}" -q "${TMP_DIR}/.env"                 "${SSH_USER}@${SSH_HOST}:/tmp/myslack.env"

ssh ${SSH_OPTS} -p "${SSH_PORT}" -t "${SSH_USER}@${SSH_HOST}" "
  sudo mkdir -p /opt/myslack/docker \
  && sudo mv /tmp/myslack.env /opt/myslack/docker/.env \
  && bash /tmp/remote_bootstrap.sh '${DOMAIN}' '${GHCR_USER}' '${GHCR_TOKEN}' '${API_IMAGE}'
"

echo "DONE. Open: https://${DOMAIN}/"
