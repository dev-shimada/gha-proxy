# gha-proxy

[日本語](README.ja.md)

A secure authentication proxy that verifies GitHub Actions OIDC tokens before forwarding requests to a backend service.

## Features

- **IP Bypass List**: Allow requests from specific IP addresses/CIDR ranges without authentication
- **OIDC Authentication**: Verify GitHub Actions OIDC tokens for non-bypassed requests
- **Basic Authentication**: Support for Basic auth with username `bearer` and JWT token as password (for GOPROXY URL-embedded credentials)
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
| `TLS_ENABLED` | No | Enable TLS/HTTPS (default: false) | `true` |
| `TLS_CERT_FILE` | Conditional | TLS certificate file path (required if TLS_ENABLED=true) | `cert.pem` |
| `TLS_KEY_FILE` | Conditional | TLS private key file path (required if TLS_ENABLED=true) | `key.pem` |
| `DEBUG` | No | Enable debug mode with detailed logging including request headers (default: false) | `true` |

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

### TLS/HTTPS Configuration

The proxy supports native HTTPS/TLS for secure communication. When TLS is enabled, the server will use HTTPS instead of HTTP.

**Security Settings:**
- Minimum TLS version: TLS 1.2
- Certificate and key files must be provided when TLS is enabled

**Generating Self-Signed Certificates (for local development):**

```bash
# Add hostname to /etc/hosts
echo "127.0.0.1   proxy.example.com" | sudo tee -a /etc/hosts

# Create OpenSSL config with Subject Alternative Names (SANs)
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

# Generate certificate with SANs
openssl req -x509 -newkey rsa:4096 -nodes \
  -keyout key.pem \
  -out cert.pem \
  -days 365 \
  -config san.cnf \
  -extensions v3_req

# Verify the certificate includes the correct SANs
openssl x509 -in cert.pem -text -noout | grep -A 1 "Subject Alternative Name"
```

**Note:** Modern Go versions require certificates to use Subject Alternative Names (SANs) instead of the legacy Common Name field. The above method ensures compatibility.

**Trusting Self-Signed Certificates:**

After generating a self-signed certificate, you need to add it to your system's trusted certificate store to avoid `x509: certificate signed by unknown authority` errors:

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

**Alternative: Using GOINSECURE (not recommended for TLS):**

Note that `GOINSECURE` only disables checksum database validation and allows HTTP (not HTTPS). It does NOT skip HTTPS certificate verification. To use the proxy with self-signed certificates, you must add the certificate to your system trust store as shown above.

**Running with TLS:**

```bash
TLS_ENABLED=true TLS_CERT_FILE=cert.pem TLS_KEY_FILE=key.pem ./gha-proxy
```

**Production Note:** In production environments, use certificates from a trusted Certificate Authority (CA) rather than self-signed certificates.

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

      - name: Configure Go proxy (Option 1: GOAUTH)
        env:
          TOKEN: ${{ steps.token.outputs.token }}
        run: |
          echo "GOPROXY=https://goproxy.example.com" >> $GITHUB_ENV
          echo "GOAUTH=github.com/${{ github.repository }}=Bearer $TOKEN" >> $GITHUB_ENV

      - name: Build
        run: go build
```

**Alternative: URL-embedded credentials (Basic Auth)**

For environments where `GOAUTH` is not available or for simpler configuration, you can embed credentials directly in the GOPROXY URL. The proxy accepts Basic auth with username `bearer` and the JWT token as the password:

```yaml
      - name: Configure Go proxy (Option 2: URL-embedded)
        env:
          TOKEN: ${{ steps.token.outputs.token }}
        run: |
          echo "GOPROXY=https://bearer:${TOKEN}@goproxy.example.com" >> $GITHUB_ENV

      - name: Build
        run: go build
```

**Note:** When using URL-embedded credentials, Go automatically sends them as Basic authentication headers.

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

# With valid Bearer token (should return 200 or 403 depending on repository match)
curl -H "Authorization: Bearer $TOKEN" \
     http://localhost:8080/github.com/myorg/myrepo/@v/list

# With Basic auth (username "bearer", password is the JWT token)
BASIC_AUTH=$(echo -n "bearer:$TOKEN" | base64)
curl -H "Authorization: Basic $BASIC_AUTH" \
     http://localhost:8080/github.com/myorg/myrepo/@v/list
```
