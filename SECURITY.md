# Security Notice

## ⚠️ CRITICAL: Set API Key in Production

**The CrashLens backend MUST have `CRASHLENS_API_KEY` set in production environments.**

Without this key, anyone can:
- Delete all workloads
- Create/modify workloads
- Spam AI diagnosis and burn Fireworks credits

### Quick Fix (Do This Now!)

1. **Generate a secure API key:**
   ```bash
   openssl rand -hex 32
   ```

2. **Set it in Railway:**
   - Go to your Railway project
   - Settings → Variables
   - Add: `CRASHLENS_API_KEY=<your_generated_key>`
   - Redeploy

3. **For SDK/Jupyter users, pass the key:**
   ```python
   tracker = WorkloadTracker(
       "https://backend-url",
       api_key=os.getenv("CRASHLENS_API_KEY")
   )
   ```

**⚠️ IMPORTANT: Never use `NEXT_PUBLIC_` for the API key!**
- `NEXT_PUBLIC_` variables are compiled into the client JavaScript bundle
- Anyone visiting your site can read them in browser devtools
- For public dashboards, use server-side routes (see below)

### What's Protected

**With API key set, these routes require authentication:**
- `POST /workloads` - Create workload (SDK/Jupyter use)
- `PUT /workloads/{id}` - Update workload (SDK/Jupyter use)
- `DELETE /workloads/{id}` - Delete workload (destructive)
- `DELETE /workloads/clear` - Clear all workloads (destructive)

**Public routes (safe for dashboard):**
- `GET /health` - Health check
- `GET /workloads` - List workloads (read-only)
- `GET /workloads/{id}` - Get workload (read-only)
- `GET /workloads/{id}/logs` - Get logs (read-only)
- `GET /workloads/{id}/metrics` - Get metrics (read-only)
- `GET /summary` - Get statistics (read-only)
- `POST /workloads/run` - Run template job (safe - only whitelisted templates)
- `POST /workloads/{id}/diagnose` - AI diagnosis (consider rate limiting)

### For Public Dashboards

If you want a public demo where strangers can click buttons:

1. **Keep `run` and `diagnose` public** (they're now safe)
2. **Add rate limiting** to prevent abuse:
   ```go
   // Example: 10 requests per minute per IP
   ```
3. **Protect destructive operations** via Next.js server route:
   ```typescript
   // app/api/clear/route.ts
   export async function POST() {
     const apiKey = process.env.CRASHLENS_API_KEY; // Server-only
     return fetch(`${backendUrl}/workloads/clear`, {
       method: 'DELETE',
       headers: { 'Authorization': `Bearer ${apiKey}` }
     });
   }
   ```

### Development Mode

If `CRASHLENS_API_KEY` is not set, the API runs in **development mode** with NO authentication.
This is fine for local testing but **NEVER acceptable in production**.

### Migration Guide

If you're migrating from an older version without auth:

1. Set `CRASHLENS_API_KEY` in production
2. Update SDK calls to include the key:
   ```python
   tracker = WorkloadTracker(
       "https://backend-url",
       api_key=os.getenv("CRASHLENS_API_KEY")
   )
   ```

3. Update frontend API client to include Authorization header

## Previously Fixed Vulnerabilities

### 1. RCE via script_path (Fixed in v0.2.0)

**Issue:** The `/workloads/run` endpoint accepted arbitrary `script_path` values, allowing code execution via `python3 "-c<code>"`.

**Fix:** Removed `script_path` parameter. Only predefined templates are accepted.

**Before:**
```json
POST /workloads/run
{"script_path": "-cimport os; os.system('rm -rf /')"}
```

**After:**
```json
POST /workloads/run
{"template": "gpu_oom"}  // Only whitelisted templates allowed
```

## Reporting Security Issues

If you find a security vulnerability, please email [your-email] instead of creating a public issue.

## Security Checklist for Production

- [ ] `CRASHLENS_API_KEY` is set
- [ ] API key is at least 32 characters (use `openssl rand -hex 32`)
- [ ] API key is stored in environment variables, not code
- [ ] Frontend/SDK includes API key in requests
- [ ] HTTPS is enabled (Railway does this automatically)
- [ ] CORS allowlist matches your actual frontend domains

## Additional Hardening (Optional)

- Add rate limiting (e.g., 100 requests/minute per IP)
- Enable request logging for audit trail
- Set up monitoring/alerts for failed auth attempts
- Use separate API keys for different clients/environments
