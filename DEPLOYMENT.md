# Deployment Guide

## Quick Local Test

Test the proxy locally with whitelisted IPs:

```bash
./test-local.sh
```

This will verify that the proxy forwards requests correctly for whitelisted IPs.

## Production Deployment

### Option 1: Docker Container (Recommended)

Build and run with Docker:

```bash
# Build
docker build -t gha-proxy .

# Run
docker run -p 8080:8080 \
  -e PORT=8080 \
  -e IP_WHITELIST="10.0.0.0/8" \
  -e AUDIENCE="https://your-domain.com" \
  -e GOPROXY_URL="https://proxy.golang.org" \
  gha-proxy
```

### Option 2: Google Cloud Run

```bash
# Build and push to GCR
gcloud builds submit --tag gcr.io/YOUR_PROJECT/gha-proxy

# Deploy
gcloud run deploy gha-proxy \
  --image gcr.io/YOUR_PROJECT/gha-proxy \
  --platform managed \
  --region us-central1 \
  --allow-unauthenticated \
  --set-env-vars="IP_WHITELIST=,AUDIENCE=https://gha-proxy-xxx.run.app,GOPROXY_URL=https://proxy.golang.org"
```

### Option 3: Fly.io

Create `fly.toml`:

```toml
app = "gha-proxy"

[build]
  dockerfile = "Dockerfile"

[env]
  PORT = "8080"
  GOPROXY_URL = "https://proxy.golang.org"

[[services]]
  internal_port = 8080
  protocol = "tcp"

  [[services.ports]]
    handlers = ["http"]
    port = 80

  [[services.ports]]
    handlers = ["tls", "http"]
    port = 443
```

Deploy:

```bash
# Set secrets
fly secrets set AUDIENCE="https://gha-proxy.fly.dev"
fly secrets set IP_WHITELIST=""

# Deploy
fly deploy
```

### Option 4: Heroku

```bash
# Create app
heroku create gha-proxy

# Set config
heroku config:set GOPROXY_URL=https://proxy.golang.org
heroku config:set AUDIENCE=https://gha-proxy.herokuapp.com
heroku config:set IP_WHITELIST=""

# Deploy
git push heroku main
```

## Post-Deployment Configuration

After deploying, update your GitHub Actions workflow:

```yaml
permissions:
  id-token: write

steps:
  - name: Get OIDC Token
    id: token
    run: |
      TOKEN=$(curl -H "Authorization: bearer $ACTIONS_ID_TOKEN_REQUEST_TOKEN" \
        "$ACTIONS_ID_TOKEN_REQUEST_URL&audience=https://YOUR_DEPLOYED_URL" \
        | jq -r .value)
      echo "::add-mask::$TOKEN"
      echo "token=$TOKEN" >> $GITHUB_OUTPUT

  - name: Download dependencies
    env:
      GOAUTH: "github.com/${{ github.repository }}=Bearer ${{ steps.token.outputs.token }}"
      GOPROXY: https://YOUR_DEPLOYED_URL
    run: go mod download
```

## Environment Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `PORT` | Server port | `8080` |
| `IP_WHITELIST` | Comma-separated IPs/CIDRs | `192.168.1.0/24,10.0.0.1` |
| `AUDIENCE` | OIDC token audience (your proxy URL) | `https://gha-proxy.example.com` |
| `GOPROXY_URL` | Backend Go proxy URL | `https://proxy.golang.org` |

## Security Considerations

1. **Always use HTTPS in production** - The proxy forwards authentication tokens
2. **Set appropriate IP_WHITELIST** - Only include trusted networks
3. **AUDIENCE must match your deployed URL** - This prevents token replay attacks
4. **Monitor logs** - Use structured logging to track authentication failures
5. **Set resource limits** - Configure timeouts and rate limiting in production

## Testing the Deployment

Test from GitHub Actions:

```yaml
- name: Test proxy
  env:
    GOPROXY: https://YOUR_DEPLOYED_URL
  run: |
    # Should work from GitHub Actions with OIDC token
    go list -m -versions github.com/${{ github.repository }}
```

Test from local machine (should fail unless IP whitelisted):

```bash
# Should return 401 Unauthorized
GOPROXY=https://YOUR_DEPLOYED_URL go list -m -versions github.com/some/module
```
