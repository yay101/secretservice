# Secret Service
Ever wanted to share a file or information via a link that only works once and didn't want to have to pay or sign in to something to do so?
Then this is the service for you.

You can self host this service behind whatever reverse proxy you prefer.

## Build
```
go build -ldflags="-s -w"
```

## Configuration
Configure using environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| SS_DOMAIN | Domain name for SSL (e.g., secretservice.au) | secretservice.au |
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
SS_DOMAIN=secretservice.example.com SS_PROXY=true SS_PORT=8080 ./secretservice
```

### OpenRC (Gentoo/Alpine)
Copy the service files:
```bash
cp etc/init.d/secretservice /etc/init.d/
cp etc/conf.d/secretservice /etc/conf.d/
rc-update add secretservice default
rc-service secretservice start
```

## Features
- [x] End-to-end encryption for text and files
- [x] Short URLs for easy sharing
- [x] Automatic expiration
- [x] WebSocket-based file uploads/downloads
- [x] Multiple secrets per submission
- [x] Self-hosted

## API Format
```json
{
  "short": true,
  "expiry": 24,
  "secrets": [
    {"type": "text", "data": "<encrypted>", "length": 100},
    {"type": "file", "data": "<encrypted_filename>", "length": 1048576}
  ],
  "key": "<base64_key>",
  "iv": "<base64_iv>"
}
```
