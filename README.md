# SendGrid Mock API

SendGrid Dev は開発中の SendGrid メール送信をテストするためのモック API です。

このプロジェクトは [yKanazawa/sendgrid-dev](https://github.com/yKanazawa/sendgrid-dev) をフォークし、以下の機能を追加したものです：

- Docker Compose サポート（MailDev 同梱）
- Event Webhook サポート
- Signed Event Webhook サポート（ECDSA P-256）
- `custom_args` サポート

## 動作環境

- Go 1.26.2+

## 環境変数

| 変数名 | デフォルト | 説明 |
|---|---|---|
| `SENDGRID_DEV_API_SERVER` | `:3030` | リッスンアドレス |
| `SENDGRID_DEV_API_KEY` | `SG.xxxxx` | Bearer トークン検証値 |
| `SENDGRID_DEV_SMTP_SERVER` | `127.0.0.1:1025` | 転送先 SMTP サーバー |
| `SENDGRID_DEV_SMTP_USERNAME` | (空) | SMTP 認証ユーザー名。設定時のみ認証あり |
| `SENDGRID_DEV_SMTP_PASSWORD` | (空) | SMTP 認証パスワード |
| `SENDGRID_DEV_TEST` | (空) | `1` にすると SMTP 送信をスキップ（テスト用） |
| `SENDGRID_DEV_EVENT_WEBHOOK_URL` | (空) | Event Webhook の送信先 URL。空の場合は無効 |
| `SENDGRID_DEV_EVENT_WEBHOOK_SIGNING_KEY` | (空) | Signed Event Webhook 用 ECDSA P-256 秘密鍵（Base64 DER 形式）。空の場合は署名なし |

## 使い方

### Docker Compose（MailDev 同梱）

最も簡単な起動方法です。sendgrid-dev と MailDev をまとめて起動します。

Docker Hub のイメージを使う場合は `compose.yml` の `build: .` を以下に置き換えてください：

```yaml
image: servalneko/sendgrid-dev:latest
```

または直接 pull して起動：

```bash
docker pull servalneko/sendgrid-dev
docker compose up
```

ソースからビルドする場合：

```bash
docker compose up --build
```

**compose.yml の例：**

```yaml
services:
  sendgrid-dev:
    image: servalneko/sendgrid-dev:latest
    ports:
      - "3030:3030"   # sendgrid-dev API
      - "1025:1025"   # maildev SMTP（外部ツールからの接続用）
      - "1080:1080"   # maildev Web UI
    environment:
      SENDGRID_DEV_API_SERVER: ":3030"
      SENDGRID_DEV_API_KEY: "SG.xxxxx"
      SENDGRID_DEV_SMTP_SERVER: "127.0.0.1:1025"
      # SENDGRID_DEV_SMTP_USERNAME: ""
      # SENDGRID_DEV_SMTP_PASSWORD: ""
      # SENDGRID_DEV_EVENT_WEBHOOK_URL: "http://host.docker.internal:8080/webhook/event"
      # SENDGRID_DEV_EVENT_WEBHOOK_SIGNING_KEY: "<Base64 DER形式の秘密鍵>"
    restart: unless-stopped
    # Linux でホスト OS 宛に Webhook を送る場合に必要
    # Mac / Windows では不要（host.docker.internal が自動で解決される）
    extra_hosts:
      - "host.docker.internal:host-gateway"
```

必要に応じて環境変数を編集してください。

ホスト上のサービスに Event Webhook を送る場合：

```yaml
environment:
  SENDGRID_DEV_EVENT_WEBHOOK_URL: "http://host.docker.internal:8080/webhook/event"
```

Linux では `host.docker.internal` の解決に `extra_hosts` が必要ですが、`compose.yml` に設定済みです。Mac / Windows では自動で解決されます。

### 手動起動（MailDev）

MailDev を起動：
```
docker pull maildev/maildev
docker run -p 1080:1080 -p 1025:1025 maildev/maildev
```

SendGrid Mock API を起動：
```
go run main.go
```

curl でメール送信：
```
curl --request POST \
  --url http://localhost:3030/v3/mail/send \
  --header 'Authorization: Bearer SG.xxxxx' \
  --header 'Content-Type: application/json' \
  --data '{"personalizations": [{ 
    "to": [{"email": "to@example.com"}]}], 
    "from": {"email": "from@example.com"}, 
    "subject": "Test Subject", 
    "content": [{"type": "text/plain", "value": "Test Content"}] 
  }'
```

MailDev で確認：http://localhost:1080/

### 手動起動（MailTrap）

```
export SENDGRID_DEV_API_SERVER=:3030
export SENDGRID_DEV_API_KEY=SG.xxxxx
export SENDGRID_DEV_SMTP_SERVER=smtp.mailtrap.io:25
export SENDGRID_DEV_SMTP_USERNAME=mailtrap_username
export SENDGRID_DEV_SMTP_PASSWORD=mailtrap_password
go run main.go
```

MailTrap Inbox で確認：https://mailtrap.io/inboxes

## Event Webhook

`SENDGRID_DEV_EVENT_WEBHOOK_URL` を設定すると、メール送信のたびに以下のイベントを POST します：

- `processed` — SMTP 送信前に送信
- `delivered` — SMTP 送信成功後に送信

ペイロード形式は [SendGrid Event Webhook](https://docs.sendgrid.com/for-developers/tracking-events/event) 仕様に準拠します。`custom_args` のフィールドは各イベントのトップレベルに展開されます。

### Signed Event Webhook

`SENDGRID_DEV_EVENT_WEBHOOK_SIGNING_KEY` を設定すると、Webhook リクエストに ECDSA P-256 署名ヘッダーが付与されます：

- `X-Twilio-Email-Event-Webhook-Signature` — `SHA-256(timestamp + payload)` の ECDSA 署名（Base64 DER）
- `X-Twilio-Email-Event-Webhook-Timestamp` — Unix タイムスタンプ文字列

鍵は `x509.MarshalECPrivateKey` の出力を `base64.StdEncoding` でエンコードした Base64 DER 形式で指定します。

## テスト

```
go test
```

## ビルド

### x86_64

```
env GOOS=linux GOARCH=amd64 go build -o sendgrid-dev_x86_64 main.go
```

### arm64

```
env GOOS=linux GOARCH=arm64 go build -o sendgrid-dev_aarch64 main.go
```
