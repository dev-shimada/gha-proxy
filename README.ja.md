# gha-proxy

[English](README.md)

GitHub Actions OIDC トークンを使用してリクエストを認証するセキュアな Go モジュールプロキシです。

## 機能

- **IP バイパスリスト**: 認証なしで特定の IP アドレス/CIDR 範囲からのリクエストを許可
- **OIDC 認証**: バイパスされていないリクエストに対して GitHub Actions OIDC トークンを検証
- **リポジトリフィルタリング**: パターンマッチングを使用して特定のリポジトリへのアクセスを制御
- **リバースプロキシ**: 認証されたリクエストをバックエンドの Go モジュールプロキシに転送

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
| `GOPROXY_URL` | はい | バックエンドプロキシの URL | `https://proxy.golang.org` |
| `ALLOWED_REPOSITORIES` | いいえ | リポジトリアクセスパターン（デフォルト: なし - すべて拒否） | `dev-shimada/*,owner/repo1` |

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
  -e GOPROXY_URL=https://proxy.golang.org \
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

      - name: Go プロキシを設定
        env:
          TOKEN: ${{ steps.token.outputs.token }}
        run: |
          echo "GOPROXY=https://goproxy.example.com" >> $GITHUB_ENV
          echo "GOAUTH=github.com/${{ github.repository }}=Bearer $TOKEN" >> $GITHUB_ENV

      - name: ビルド
        run: go build
```

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

# 有効なトークンあり（リポジトリの一致に応じて 200 または 403 が返るはず）
curl -H "Authorization: Bearer $TOKEN" \
     http://localhost:8080/github.com/myorg/myrepo/@v/list
```
