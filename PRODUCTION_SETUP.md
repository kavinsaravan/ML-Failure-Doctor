# Production Deployment

## Quick Setup

**Backend environment (native deployment):**

Enter these values in your hosting platform’s environment settings. The examples
below use `.env` syntax; for a shell launch, export each variable before starting
the backend. Native `go run .` does not automatically load a `.env` file.

```env
CRASHLENS_API_KEY=your_secure_operator_key  # Keep private
APP_ENV=production
ACCESS_MODE=private
DATABASE_PATH=/var/lib/crashlens/crashlens.db  # Must persist across restarts
PORT=8080
ALLOWED_ORIGINS=https://crashlens.example.com  # No trailing slash
```

**Frontend environment:**
```env
NEXT_PUBLIC_API_URL=https://api.crashlens.example.com
```

For a native backend, create `/var/lib/crashlens` and make it writable by the
backend’s service user. The database directory must persist across restarts and
redeployments.

**Docker:**
```bash
cp .env.example .env  # Edit with your values
docker compose up --build -d
```

Compose fixes `DATABASE_PATH` at `/app/data/crashlens.db` and mounts the
`crashlens-data` named volume at `/app/data`. The native path above does not apply
to Compose. Changing `DATABASE_PATH` in the root `.env` does not override this
Compose setting.

## Critical Requirements

- **Database persistence**: Mount a volume for `DATABASE_PATH` - contains all keys, workloads, logs, and metrics
- **One process per database**: Run only one backend instance per SQLite file
- **HTTPS required**: Use HTTPS in production for credential security
- **Never expose secrets**: Don't put keys in `NEXT_PUBLIC_` variables

## Workspace Management

**Self-service creation** (enabled by default):
- Users create workspaces from the dashboard
- Receive a key once (no recovery)
- Disable with `ALLOW_WORKSPACE_CREATION=false`

### Operator Key Management

```bash
python scripts/manage_keys.py --api-url https://YOUR-BACKEND issue Alice
python scripts/manage_keys.py --api-url https://YOUR-BACKEND list
python scripts/manage_keys.py --api-url https://YOUR-BACKEND revoke KEY_ID

# Rotate key (preserve workloads)
python scripts/manage_keys.py --api-url https://YOUR-BACKEND issue Alice --owner-id OWNER_ID
python scripts/manage_keys.py --api-url https://YOUR-BACKEND revoke OLD_KEY_ID
```

**Security:**
- Keys stored as SHA-256 hashes only
- Operator cannot access other workspace's workloads
- See [Authentication & Multi-Tenancy](README.md#authentication--multi-tenancy)

## AI Diagnosis

**User credentials (recommended):**
- Users provide their own Fireworks key + model in dashboard or SDK
- Credentials sent per-request, never stored

**Operator credentials (legacy workspace only):**
```env
FIREWORKS_API_KEY=your_key
FIREWORKS_MODEL=accounts/fireworks/models/llama-v3p3-70b-instruct
# Both required; only works for operator workspace
```

## Resource Limits

```env
JOB_CONCURRENCY=2        # Max simultaneous demo jobs
JOB_QUEUE_SIZE=16        # Max queued jobs
JOB_TIMEOUT_SECONDS=300  # Demo job timeout
```

## Security

**Reverse proxy:**
```env
TRUSTED_PROXY_CIDRS=10.0.0.0/8  # Only if using trusted proxy
```

**Rate limits** (workspace creation):
- 3/min per IP
- 10/min globally

## Validation

```bash
# Health check
curl https://your-backend/health

# Docker smoke test
python scripts/docker_smoke.py

# GPU validation (on GPU host)
python scripts/validate_gpu.py
python scripts/validate_gpu.py --device mps  # macOS only
```

## Troubleshooting

**Database locked**: Only one backend process per database file

**CORS errors**: Verify `ALLOWED_ORIGINS` matches frontend URL exactly (no trailing slash)

**Keys not persisting**: Check `DATABASE_PATH` volume mount and permissions

**AI diagnosis fails**: Verify both `FIREWORKS_API_KEY` and `FIREWORKS_MODEL` are set (operator), or user provided valid BYOK credentials
