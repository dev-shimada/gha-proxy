# gha-proxy

[English](README.md)

GitHub Actions OIDC トークンを検証してからバックエンドサービスにリクエストを転送するセキュアな認証プロキシです。

## 機能

- **IP バイパスリスト**: 認証なしで特定の IP アドレス/CIDR 範囲からのリクエストを許可
- **OIDC 認証**: バイパスされていないリクエストに対して GitHub Actions OIDC トークンを検証
- **Basic 認証**: ユーザー名 `bearer`、パスワードに JWT トークンを使用した Basic 認証をサポート（GOPROXY の URL 埋め込み認証情報用）
- **リポジトリフィルタリング**: パターンマッチングを使用して特定のリポジトリへのアクセスを制御
- **リバースプロキシ**: 認証されたリクエストをバックエンドサービス（Go モジュールプロキシ、API など）に転送

## アーキテクチャ

```
リクエスト → IP バイパスチェック → トークン検証 → リポジトリフィルタ → バックエンドへプロキシ
            ↓ (バイパス)           ↓ (トークンなし/無効)  ↓ (許可されていない)
            パス                    401 Unauthorized       403 Forbidden
```

## 設定

環境変数を使用してプロキシを設定します：

| 変数 | 必須 | 説明 | 例 |
|----------|----------|-------------|---------|
| `PORT` | いいえ | サーバーポート（デフォルト: 8080） | `8080` |
| `BYPASS_IP_LIST` | いいえ | カンマ区切りの IP/CIDR バイパスリスト | `127.0.0.1,192.168.1.0/24` |
| `AUDIENCE` | はい | OIDC トークンのオーディエンス | `https://goproxy.example.com` |
| `BACKEND_URL` | はい | プロキシ先のバックエンドサービス URL | `https://proxy.golang.org` または `https://api.example.com` |
| `ALLOWED_REPOSITORIES` | いいえ | リポジトリアクセスパターン（デフォルト: なし - すべて拒否） | `dev-shimada/*,owner/repo1` |
| `TLS_ENABLED` | いいえ | TLS/HTTPS を有効化（デフォルト: false） | `true` |
| `TLS_CERT_FILE` | 条件付き | TLS 証明書ファイルのパス（TLS_ENABLED=true の場合は必須） | `cert.pem` |
| `TLS_KEY_FILE` | 条件付き | TLS 秘密鍵ファイルのパス（TLS_ENABLED=true の場合は必須） | `key.pem` |
| `DEBUG` | いいえ | デバッグモードを有効化し、リクエストヘッダーを含む詳細なログを出力（デフォルト: false） | `true` |

### リポジトリアクセスパターン

`ALLOWED_REPOSITORIES` 変数は、プロキシの使用を許可する GitHub Actions ワークフロー（リポジトリ）を制御します。カンマ区切りのパターンを受け付け、OIDC トークンの `repository` クレームに対してチェックを行います：

- `*` - すべてのリポジトリを許可
- `owner/*` - 特定のオーナー配下のすべてのリポジトリを許可
- `owner/repo` - 特定のリポジトリのみを許可
- `owner1/*,owner2/repo1,owner2/repo2` - 複数のパターンを組み合わせ

**注意:** 設定しない場合、デフォルトの動作は**すべてのリクエストを拒否**します（IP でバイパスされたものを除く）。許可するリポジトリを明示的に設定する必要があります。

**例:**

```bash
# すべてのリポジトリを許可
ALLOWED_REPOSITORIES=*

# dev-shimada オーガニゼーションのすべてのリポジトリを許可
ALLOWED_REPOSITORIES=dev-shimada/*

# 特定のリポジトリのみを許可
ALLOWED_REPOSITORIES=dev-shimada/app,dev-shimada/backend

# 複数のオーナーと特定のリポジトリを許可
ALLOWED_REPOSITORIES=dev-shimada/*,otherowner/trusted-app
```

**注意:** リポジトリチェックは、リクエストされているモジュールパスではなく、**送信元リポジトリ**（OIDC トークンの `repository` クレーム）に対して実行されます。例えば、`dev-shimada/app` のワークフローが任意の Go モジュールをリクエストする場合、アクセスを許可するには `dev-shimada/app` または `dev-shimada/*` をパターンに含める必要があります。

### TLS/HTTPS 設定

プロキシはセキュアな通信のためにネイティブ HTTPS/TLS をサポートしています。TLS を有効化すると、サーバーは HTTP ではなく HTTPS を使用します。

**セキュリティ設定:**
- 最小 TLS バージョン: TLS 1.2
- TLS を有効化する場合は証明書と鍵ファイルを提供する必要があります

**自己署名証明書の生成（ローカル開発用）:**

```bash
# /etc/hosts にホスト名を追加
echo "127.0.0.1   proxy.example.com" | sudo tee -a /etc/hosts

# Subject Alternative Names (SANs) を含む OpenSSL 設定ファイルを作成
cat > san.cnf << 'EOF'
[req]
default_bits = 4096
distinguished_name = req_distinguished_name
req_extensions = v3_req
prompt = no

[req_distinguished_name]
CN = proxy.example.com

[v3_req]
subjectAltName = @alt_names

[alt_names]
DNS.1 = proxy.example.com
DNS.2 = *.proxy.example.com
IP.1 = 127.0.0.1
IP.2 = ::1
EOF

# SANs を含む証明書を生成
openssl req -x509 -newkey rsa:4096 -nodes \
  -keyout key.pem \
  -out cert.pem \
  -days 365 \
  -config san.cnf \
  -extensions v3_req

# 証明書に正しい SANs が含まれているか確認
openssl x509 -in cert.pem -text -noout | grep -A 1 "Subject Alternative Name"
```

**注意:** 最新の Go バージョンでは、レガシーな Common Name フィールドではなく、Subject Alternative Names (SANs) を使用した証明書が必要です。上記の方法で互換性が確保されます。

**自己署名証明書を信頼する:**

自己署名証明書を生成した後、`x509: certificate signed by unknown authority` エラーを回避するために、システムの信頼された証明書ストアに追加する必要があります：

```bash
# Ubuntu/Debian
sudo cp cert.pem /usr/local/share/ca-certificates/gha-proxy.crt
sudo update-ca-certificates

# macOS
sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain cert.pem

# RHEL/CentOS/Fedora
sudo cp cert.pem /etc/pki/ca-trust/source/anchors/gha-proxy.crt
sudo update-ca-trust

# Alpine Linux
sudo cp cert.pem /usr/local/share/ca-certificates/gha-proxy.crt
sudo update-ca-certificates
```

**別の方法: GOINSECURE の使用（TLS には非推奨）:**

`GOINSECURE` はチェックサムデータベースの検証を無効にし、HTTP（HTTPSではない）を許可するだけです。HTTPS証明書の検証はスキップ**されません**。自己署名証明書を使ったプロキシを使用するには、上記のようにシステムの信頼ストアに証明書を追加する必要があります。

**TLS で実行:**

```bash
TLS_ENABLED=true TLS_CERT_FILE=cert.pem TLS_KEY_FILE=key.pem ./gha-proxy
```

**本番環境での注意:** 本番環境では、自己署名証明書ではなく、信頼された認証局（CA）からの証明書を使用してください。

## 使用方法

### ローカル開発

1. サンプル環境ファイルをコピー:
```bash
cp .env.example .env
```

2. `.env` を編集して設定を記述

3. サーバーを起動:
```bash
export $(cat .env | xargs)
go run main.go
```

### Docker

```bash
docker build -t gha-proxy .
docker run -p 8080:8080 \
  -e AUDIENCE=https://goproxy.example.com \
  -e BACKEND_URL=https://proxy.golang.org \
  gha-proxy
```

## GitHub Actions との統合

GitHub Actions ワークフローでプロキシを使用するには：

1. ワークフローに `id-token: write` 権限を付与
2. 正しいオーディエンスで OIDC トークンを取得
3. 認証にトークンを使用するよう Go を設定

ワークフローの例:

```yaml
name: プライベートモジュールを使用したビルド

permissions:
  id-token: write

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: '1.25'

      - name: OIDC トークンを取得
        id: token
        run: |
          TOKEN=$(curl -H "Authorization: bearer $ACTIONS_ID_TOKEN_REQUEST_TOKEN" \
            "$ACTIONS_ID_TOKEN_REQUEST_URL&audience=https://goproxy.example.com" \
            | jq -r .value)
          echo "::add-mask::$TOKEN"
          echo "token=$TOKEN" >> $GITHUB_OUTPUT

      - name: Go プロキシを設定（オプション1: GOAUTH）
        env:
          TOKEN: ${{ steps.token.outputs.token }}
        run: |
          echo "GOPROXY=https://goproxy.example.com" >> $GITHUB_ENV
          echo "GOAUTH=github.com/${{ github.repository }}=Bearer $TOKEN" >> $GITHUB_ENV

      - name: ビルド
        run: go build
```

**代替方法: URL 埋め込み認証情報（Basic 認証）**

`GOAUTH` が利用できない環境や、よりシンプルな設定が必要な場合は、GOPROXY URL に直接認証情報を埋め込むことができます。プロキシはユーザー名 `bearer`、パスワードに JWT トークンを使用した Basic 認証を受け付けます：

```yaml
      - name: Go プロキシを設定（オプション2: URL 埋め込み）
        env:
          TOKEN: ${{ steps.token.outputs.token }}
        run: |
          echo "GOPROXY=https://bearer:${TOKEN}@goproxy.example.com" >> $GITHUB_ENV

      - name: ビルド
        run: go build
```

**注意:** URL 埋め込み認証情報を使用すると、Go は自動的に Basic 認証ヘッダーとして送信します。

## セキュリティに関する考慮事項

- **TLS が必須**: 本番環境では、転送中のトークンを保護するために常に TLS を使用してください
- **トークンのログ記録**: トークンはログに記録されません（ログ内でマスクされます）
- **タイムアウト**: リソースの枯渇を防ぐため、読み取り/書き込みタイムアウトは 30 秒に設定されています
- **JWKS キャッシング**: 外部リクエストを削減するため、公開鍵は 1 時間キャッシュされます

## テスト

### IP バイパスリストのテスト

```bash
# バイパスされた IP から（成功するはず）
curl http://localhost:8080/golang.org/x/text/@v/list
```

### 認証のテスト

```bash
# トークンなし（401 が返るはず）
curl http://localhost:8080/github.com/myorg/myrepo/@v/list

# 有効な Bearer トークンあり（リポジトリの一致に応じて 200 または 403 が返るはず）
curl -H "Authorization: Bearer $TOKEN" \
     http://localhost:8080/github.com/myorg/myrepo/@v/list

# Basic 認証（ユーザー名 "bearer"、パスワードは JWT トークン）
BASIC_AUTH=$(echo -n "bearer:$TOKEN" | base64)
curl -H "Authorization: Basic $BASIC_AUTH" \
     http://localhost:8080/github.com/myorg/myrepo/@v/list
```
