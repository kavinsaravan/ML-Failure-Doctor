# Deploy CrashLens

Configure the backend with a private operator CRASHLENS_API_KEY,
APP_ENV=production, ACCESS_MODE=private, and a persistent DATABASE_PATH. Keep one
backend process per SQLite database. Preserve the data volume across redeploys;
it contains keys, ownership, logs, metrics, and reports.

Set frontend NEXT_PUBLIC_API_URL to the public HTTPS backend URL and rebuild.
Set backend ALLOWED_ORIGINS to the exact frontend origin without a path or trailing
slash. Never put keys in NEXT_PUBLIC variables. Docker setup is in `.env.example`
and `docker-compose.yml`; run `docker compose up --build -d`.

Users create their own workspaces from the dashboard and save a key once. The
same key connects their SDK or MCP client. No email/password recovery exists.
Set ALLOW_WORKSPACE_CREATION=false to use only operator issuance. The operator
can list, revoke, and replace keys through scripts/manage_keys.py but cannot read
other owners' workloads. See [access and operations](README.md#access-and-operations).

Users supply their own Fireworks credentials and tool-capable model per diagnosis
request. Missing credentials produce rules reports. Only the operator legacy
workspace may use backend FIREWORKS_API_KEY/FIREWORKS_MODEL, and both must be
explicitly configured. New diagnoses may incur charges; saved AI reports are reused.

External workloads run on user hardware. Apple MPS requires native macOS, while
CUDA/ROCm require supported GPU hosts. Dashboard template jobs are simulated
examples executed on the backend. Bound execution using JOB_CONCURRENCY,
JOB_QUEUE_SIZE, and JOB_TIMEOUT_SECONDS. Restart recovery marks interrupted
backend-managed jobs failed; external SDK jobs can continue uploading.

Set TRUSTED_PROXY_CIDRS only for known proxy networks. Forwarded IPs are otherwise
ignored. Public creation has global and per-IP limits. This is a prototype with
no per-user diagnosis quotas and no detection of killed external SDK processes.

Validation commands and limitations are in [README](README.md#validation).
Run scripts/docker_smoke.py for disposable container checks and
scripts/validate_gpu.py on a supported GPU host for real training/live telemetry.
