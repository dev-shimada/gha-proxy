# gha-proxy

[日本語](README.ja.md)

A secure authentication proxy that verifies GitHub Actions OIDC tokens before forwarding requests to a backend service.

## Features

- **IP Bypass List**: Allow requests from specific IP addresses/CIDR ranges without authentication
- **OIDC Authentication**: Verify GitHub Actions OIDC tokens for non-bypassed requests
- **Repository Filtering**: Control access to specific repositories using pattern matching
- **Reverse Proxy**: Forward authenticated requests to a backend service (Go module proxy, API, etc.)

## Architecture

```
Request → IP Bypass Check → Token Verification → Repository Filter → Proxy to Backend
         ↓ (bypassed)       ↓ (no token/invalid)  ↓ (not allowed)
         Pass                401 Unauthorized       403 Forbidden
```

## Configuration

Configure the proxy using environment variables:

| Variable | Required | Description | Example |
|----------|----------|-------------|---------|
| `PORT` | No | Server port (default: 8080) | `8080` |
| `BYPASS_IP_LIST` | No | Comma-separated IP/CIDR bypass list | `127.0.0.1,192.168.1.0/24` |
| `AUDIENCE` | Yes | OIDC token audience | `https://goproxy.example.com` |
| `BACKEND_URL` | Yes | Backend service URL to proxy to | `https://proxy.golang.org` or `https://api.example.com` |
| `ALLOWED_REPOSITORIES` | No | Repository access patterns (default: none - all denied) | `dev-shimada/*,owner/repo1` |

### Repository Access Patterns

The `ALLOWED_REPOSITORIES` variable controls which GitHub Actions workflows (repositories) are allowed to use the proxy. It accepts comma-separated patterns and checks against the `repository` claim in the OIDC token:

- `*` - Allow all repositories
- `owner/*` - Allow all repositories under a specific owner
- `owner/repo` - Allow a specific repository
- `owner1/*,owner2/repo1,owner2/repo2` - Combine multiple patterns

**Note:** If not set, the default behavior is to **deny all requests** (except those bypassed by IP). You must explicitly configure allowed repositories.

**Examples:**

```bash
# Allow all repositories
ALLOWED_REPOSITORIES=*

# Allow all repositories from dev-shimada organization
ALLOWED_REPOSITORIES=dev-shimada/*

# Allow specific repositories
ALLOWED_REPOSITORIES=dev-shimada/app,dev-shimada/backend

# Allow multiple owners and specific repositories
ALLOWED_REPOSITORIES=dev-shimada/*,otherowner/trusted-app
```

**Note:** The repository check is performed against the **source repository** (from the OIDC token's `repository` claim), not the module path being requested. For example, if a workflow in `dev-shimada/app` requests any Go module, the pattern must include `dev-shimada/app` or `dev-shimada/*` for access to be granted.

## Usage

### Local Development

1. Copy the example environment file:
```bash
cp .env.example .env
```

2. Edit `.env` with your configuration

3. Run the server:
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

## GitHub Actions Integration

To use the proxy in GitHub Actions workflows:

1. Grant `id-token: write` permission to your workflow
2. Fetch an OIDC token with the correct audience
3. Configure Go to use the token for authentication

Example workflow:

```yaml
name: Build with Private Modules

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

      - name: Get OIDC Token
        id: token
        run: |
          TOKEN=$(curl -H "Authorization: bearer $ACTIONS_ID_TOKEN_REQUEST_TOKEN" \
            "$ACTIONS_ID_TOKEN_REQUEST_URL&audience=https://goproxy.example.com" \
            | jq -r .value)
          echo "::add-mask::$TOKEN"
          echo "token=$TOKEN" >> $GITHUB_OUTPUT

      - name: Configure Go proxy
        env:
          TOKEN: ${{ steps.token.outputs.token }}
        run: |
          echo "GOPROXY=https://goproxy.example.com" >> $GITHUB_ENV
          echo "GOAUTH=github.com/${{ github.repository }}=Bearer $TOKEN" >> $GITHUB_ENV

      - name: Build
        run: go build
```

## Security Considerations

- **TLS Required**: Always use TLS in production to protect tokens in transit
- **Token Logging**: Tokens are never logged (masked in logs)
- **Timeouts**: Read/Write timeouts are set to 30 seconds to prevent resource exhaustion
- **JWKS Caching**: Public keys are cached for 1 hour to reduce external requests

## Testing

### IP Bypass List Test

```bash
# From bypassed IP (should succeed)
curl http://localhost:8080/golang.org/x/text/@v/list
```

### Authentication Test

```bash
# Without token (should return 401)
curl http://localhost:8080/github.com/myorg/myrepo/@v/list

# With valid token (should return 200 or 403 depending on repository match)
curl -H "Authorization: Bearer $TOKEN" \
     http://localhost:8080/github.com/myorg/myrepo/@v/list
```
