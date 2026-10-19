# Deployment and real workloads

1. Copy `.env.example` to `.env` and configure `CRASHLENS_API_KEY`,
   `APP_ENV=production`, and `ACCESS_MODE=private`.
2. Set `NEXT_PUBLIC_API_URL` to the backend URL reachable from the browser and
   `ALLOWED_ORIGINS` to the exact dashboard origin. Rebuild when the public URL changes.
3. Start with `docker compose up --build -d`. SQLite lives in the named data volume. Run one backend instance per database.
4. Open the dashboard and enter the API key. It remains in memory until refresh.
5. Configure SDK and MCP clients with the same key.

```python
import os
from crashlens import WorkloadTracker

tracker = WorkloadTracker(
    os.environ["CRASHLENS_URL"],
    api_key=os.environ["CRASHLENS_API_KEY"],
)
with tracker.track("Model training"):
    train_model()
```

The SDK uploads current logs, runtime, and GPU samples every two seconds, followed
by a terminal update. Dashboard pages poll every three seconds. No GPU readings
are invented by the SDK when no GPU is available. Notebook tracking uses the same
publisher. Reporting errors do not replace training errors.

Set `JOB_CONCURRENCY`, `JOB_QUEUE_SIZE`, and `JOB_TIMEOUT_SECONDS` for server-run
template jobs. Queue overflow returns 503 with Retry-After; IP throttling returns
429. On restart, lost server-managed work is marked failed with an interruption
reason. External SDK jobs remain running and can resume uploading to the restarted
API. SDK kernel/process kills still need external liveness monitoring.

Use `TRUSTED_PROXY_CIDRS` only when the actual proxy network is known. Do not trust
all networks merely to enable forwarded headers. See [access policy](SECURITY.md).

Validate containers without modifying the application's data:

```bash
python3 scripts/docker_smoke.py
```

On a NVIDIA/AMD host with GPU-compatible PyTorch and the SDK installed:

```bash
export CRASHLENS_URL=https://your-backend.example
export CRASHLENS_API_KEY=your-key
python3 scripts/validate_gpu.py
```

The hardware test runs a small training loop and requires real GPU readings to
reach the API before completion. It refuses CPU fallback and simulated readings.
