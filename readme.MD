# Secret Service

Ever wanted to share a file or information via a link that only works once and didn't want to have to pay or sign in to something to do so?
Then this is the service for you.

You can self host this service behind whatever reverse proxy you prefer.

## Build

```bash
go build -ldflags="-s -w"
```

## Docker

### Build

```bash
docker build -t secretservice .
```

### Run

```bash
docker run -d \
  --name secretservice \
  -p 80:80 -p 443:443 \
  -v ss-data:/data \
  -e SS_DOMAIN=secretservice.example.com \
  secretservice
```

### Behind Reverse Proxy

```bash
docker run -d \
  --name secretservice \
  -p 8080:8080 \
  -v ss-data:/data \
  -e SS_PROXY=true \
  -e SS_PORT=8080 \
  secretservice
```

### Docker Compose

```yaml
services:
  secretservice:
    build: .
    restart: unless-stopped
    ports:
      - "8080:8080"
    volumes:
      - ss-data:/data
    environment:
      - SS_PROXY=true
      - SS_PORT=8080
      - SS_DOMAIN=secretservice.example.com

volumes:
  ss-data:
```

## Configuration

Configure using environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| SS_DOMAIN | Domain name for SSL | secretservice.au |
| SS_PATH | Directory for file storage | /mnt/cache |
| SS_PORT | HTTP port for proxy mode | 80 |
| SS_SSL | HTTPS port for standalone mode | 443 |
| SS_PROXY | Set to "true" if behind a reverse proxy | false |
| SS_CHUNK | Chunk size for file uploads in bytes | 1048576 (1MB) |

## Deployment

### Standalone (with ACME SSL)

```bash
SS_DOMAIN=secretservice.example.com SS_PATH=/var/lib/secretservice ./secretservice
```

### Behind Reverse Proxy

```bash
SS_PROXY=true SS_PORT=8080 ./secretservice
```

### OpenRC (Gentoo/Alpine)

```bash
cp etc/init.d/secretservice /etc/init.d/
cp etc/conf.d/secretservice /etc/conf.d/
rc-update add secretservice default
rc-service secretservice start
```

## Features

- [x] End-to-end AES-GCM encryption for text and files
- [x] Short URLs for easy sharing
- [x] Automatic expiration
- [x] WebSocket-based file uploads/downloads
- [x] Multiple secrets per submission
- [x] Self-hosted
- [x] Docker support

## Security Model

### Long URLs (zero-knowledge)

The server never receives the encryption key. The URL format is:

```
https://domain.com/<parentID>/<IV>#<key>
```

The key is in the URL fragment (`#`) which browsers never send to the server. The server cannot decrypt any data.

### Short URLs

Both the key and IV are sent to the server for storage. The short URL redirects to the full long URL on first access.

## API Format

```json
{
  "short": true,
  "expiry": 24,
  "secrets": [
    {"type": "text", "data": "<encrypted>", "length": 100},
    {"type": "file", "data": "<encrypted_filename>", "length": 1048576}
  ],
  "key": "<base64url_key>",
  "iv": "<base64url_iv>"
}
```

For long URLs, omit `key` and `iv` from the request.
