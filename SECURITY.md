# Access and execution policy

Set `CRASHLENS_API_KEY` and `APP_ENV=production` before deploying. Production
startup rejects a missing key. Development without a key is open access.
Generate a key with `openssl rand -hex 32` and provide it through backend secrets.
Never put a key in `NEXT_PUBLIC_` variables or commit it to the repository.

With a key configured, `ACCESS_MODE=private` (default) protects workload lists,
logs, metrics, summaries, diagnoses, job execution, and mutations. Only health is
public. `ACCESS_MODE=demo` explicitly permits public workload reads; logs may
contain sensitive data, so use synthetic demo workloads. All mutations, template
execution, and paid diagnosis still require the configured key in demo mode.

The dashboard asks for the key, validates it against `/session`, and sends a
Bearer header. The key stays in browser memory and must be entered again after
refresh. SDK and MCP clients use `CRASHLENS_API_KEY`. Use HTTPS on deployed hosts.

Set `ALLOWED_ORIGINS` to exact frontend origins. Forwarded client IP headers are
ignored by default. Set `TRUSTED_PROXY_CIDRS` only to actual reverse-proxy networks;
the limiter walks the forwarding chain from the trusted peer toward the client.
Changing connection ports does not reset an IP's limit. Limiter entries expire
only after inactivity, and the map has a fixed capacity.

Template jobs use a fixed worker pool, bounded queue, and execution deadline.
Defaults: two workers, 16 queued jobs, 300-second deadline. Deadline/shutdown
cancellation kills the process group on Linux and macOS. The runner retains the
last 256 KiB of logs and 300 metric samples. Request bodies are limited to 1 MiB.
Only predefined template names are accepted; client-supplied script paths are
never executed. Recovery fails interrupted server-managed jobs without changing
SDK workloads executing on other hosts.

This is a single-key application, not a multi-tenant authorization system.
