# myslack(個人開発)


## 全体像
- フロント：`app/frontend` をビルド → S3 に配置 → CloudFront で配信（必要に応じて無効化で反映）
- API：EC2 上で `docker compose` で **postgres / api / nginx** を起動  
  初回は **HTTP(80)** でブート → **Certbot(webroot)** で証明書発行 → **HTTPS(443)** に切替
- 認証：Auth0。**CloudFront ドメイン**を Allowed Origins/Callback 等に登録
- 機密値は **.env（サーバ側）** / **環境変数（GHCR_TOKEN など）** で受け渡し。**Git 管理に含めない**

---

### IaC と分離方針
- IaC（Terraform）で S3 / CloudFront / 付随リソースを構築  
  - `scratch.tfvars` 等で環境差分を吸収（バケット名、OAC/OAI、CF 設定など）
- **API の EC2 構築**は必要に応じて Terraform or 手動。アプリ展開自体は `deploy.sh` で自動化
- 秘密値（JWT/DB パス/GHCR_TOKEN 等）は **別管理**（`.env` / CI のシークレット / SSM）

---

### デプロイ 
**インフラ（必要に応じて）**
```bash
terraform init
terraform validate
terraform plan -var-file=scratch.tfvars
terraform apply -var-file=scratch.tfvars
```

### フロント（ビルド & 配信）
```bash
cd app/frontend
npm ci
npm run build   # dist/ 生成

# HTML は Content-Type を固定
aws s3 sync ./dist s3://myslack-web-scratch-123456/ \
  --delete \
  --cache-control "public,max-age=300" \
  --content-type "text/html" --exclude "*" --include "*.html"

# HTML 以外は自動判定でOK
aws s3 sync ./dist s3://myslack-web-scratch-123456/ --delete

# 反映を早める
aws cloudfront create-invalidation \
  --distribution-id E1EO7DA43X00KR \
  --paths "/*"
```

### API（GHCR ログイン → デプロイ）
```bash
export GHCR_TOKEN=ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
echo "$GHCR_TOKEN" | docker login ghcr.io -u yuima0928 --password-stdin

chmod +x ./deploy.sh
bash -x ./deploy.sh
```

### Auth0設定メモ
- Allowed Web Origins / Allowed Origins (CORS)：https://<CloudFront ドメイン>
- Allowed Callback URLs：実装のコールバックパス（例 https://<CF>/callback）
- Allowed Logout URLs：https://<CloudFront ドメイン>/
- API 側 .env に AUTH0_DOMAIN / AUTH0_AUDIENCE を設定

### 環境差分の保ち方
- Terraform：*.tfvars 単位でバケット名や CF 設定を切替
- API：.envで JWT/DB/S3/CORS 等を切替
- ドメイン：api.<EIP>.nip.io を利用（EIP が変わったら DOMAIN を更新）

## ローカル開発

### backendの起動
```bash
cd app
docker compose up -d --build
```

### frontendの起動
```bash
cd app/frontend
npm ci
npm run dev
```
