# Production Deployment

## Quick Setup

**Backend environment (native deployment):**

Enter these values in your hosting platform’s environment settings. The examples
below use `.env` syntax; for a shell launch, export each variable before starting
the backend. Native `go run .` does not automatically load a `.env` file.

```env
CRASHLENS_API_KEY=your_secure_operator_key  # Keep private
APP_ENV=production
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

Requires Docker and Docker Compose. From a terminal:

```bash
git clone https://github.com/kavinsaravan/ML-Failure-Doctor.git
cd ML-Failure-Doctor
cp .env.example .env  # Configure your operator key and deployment settings.
docker compose up --build -d
docker compose logs -f
```

Open `http://localhost:3000` for the dashboard. The API listens on
`http://localhost:8080`; its public health check is `/health`.
Use `docker compose down` to stop the stack or `docker compose restart` to restart it.

Compose fixes `DATABASE_PATH` at `/app/data/crashlens.db` and mounts the
`crashlens-data` named volume at `/app/data`. The native path above does not apply
to Compose. Changing `DATABASE_PATH` in the root `.env` does not override this
Compose setting.

## Local Development

Requires Go 1.26+, Node.js 20.9+, and a C compiler with CGO enabled for SQLite
(Xcode Command Line Tools on macOS or GCC/build tools on Linux). Python and any
GPU-compatible PyTorch installation belong on the machine running your workloads.

From the repository root, start the backend:

```bash
cd backend
go mod download
export CRASHLENS_API_KEY="your_secure_operator_key"
go run .
```

In another terminal, start the frontend:

```bash
cd frontend
npm ci
# For a remote API, set NEXT_PUBLIC_API_URL in frontend/.env.local first.
npm run dev
```

Open `http://localhost:3000`, create a workspace, and save its key. For SDK runs
against this local API, set `CRASHLENS_URL=http://localhost:8080` or pass that URL
as `api_url=`. Colab needs a backend URL it can reach; its `localhost` is separate
from your development machine.

`NEXT_PUBLIC_API_URL` is embedded at build time. Rebuild the frontend after changing
it for a deployed or Docker stack. It must be reachable from the user's browser;
use the public backend URL, not Docker's internal `http://backend:8080` hostname.

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

### Operator Key Revocation

Users obtain keys through workspace signup. To revoke a key, use the existing
operator-only API directly; a separate CLI is not required. Set `CRASHLENS_URL`
to your backend and `CRASHLENS_API_KEY` to the operator key in your shell:

```bash
# Find the key ID (this lists metadata, never the saved secrets).
curl --fail -H "Authorization: Bearer $CRASHLENS_API_KEY" "$CRASHLENS_URL/api-keys"

# Revoke that key immediately.
curl --fail -X DELETE -H "Authorization: Bearer $CRASHLENS_API_KEY" "$CRASHLENS_URL/api-keys/KEY_ID"
```

**Security:**
- Keys stored as SHA-256 hashes only
- Operator cannot access other workspace's workloads
- See [Authentication & Multi-Tenancy](README.md#authentication--multi-tenancy)

## AI Diagnosis

Users provide their own Fireworks key in the dashboard or SDK. Diagnosis defaults
to the `auto` serverless router; an explicit tool-capable model can override it.
Keys are sent per request and are not stored. Without a request key, the backend
returns a rule-based diagnosis. This also applies to operator requests: backend
`FIREWORKS_API_KEY` and `FIREWORKS_MODEL` variables do not fund API diagnoses.
The separate offline evaluator may still read these variables for explicitly
requested paid benchmark runs.

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
python scripts/validate_gpu.py --device cuda  # NVIDIA CUDA or AMD ROCm PyTorch build
```

## Troubleshooting

**Database locked**: Only one backend process per database file

**CORS errors**: Verify `ALLOWED_ORIGINS` matches frontend URL exactly (no trailing slash)

**Keys not persisting**: Check `DATABASE_PATH` volume mount and permissions

**AI diagnosis fails**: Verify the user supplied a valid Fireworks key in the dashboard or SDK. Model selection defaults to `auto`; explicit overrides must support tool calling.
