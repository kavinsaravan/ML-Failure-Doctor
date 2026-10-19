# Security Notice

## ⚠️ CRITICAL: Set API Key in Production

**The CrashLens backend MUST have `CRASHLENS_API_KEY` set in production environments.**

Without this key, anyone can:
- Execute arbitrary Python code on your server
- Delete all workloads
- Create/modify workloads

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

3. **Update your frontend to include the key:**
   ```typescript
   // In frontend/lib/api.ts or equivalent
   const API_KEY = process.env.NEXT_PUBLIC_CRASHLENS_API_KEY;

   fetch(url, {
     headers: {
       'Authorization': `Bearer ${API_KEY}`,
       'Content-Type': 'application/json'
     }
   })
   ```

### What's Protected

**With API key set, these routes require authentication:**
- `POST /workloads` - Create workload
- `POST /workloads/run` - Run job (CRITICAL - prevents RCE)
- `PUT /workloads/{id}` - Update workload
- `DELETE /workloads/{id}` - Delete workload
- `DELETE /workloads/clear` - Clear all workloads
- `POST /workloads/{id}/diagnose` - Run diagnosis

**Public (read-only) routes:**
- `GET /health`
- `GET /workloads`
- `GET /workloads/{id}`
- `GET /workloads/{id}/logs`
- `GET /workloads/{id}/metrics`
- `GET /summary`

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
