# CrashLens dashboard

The Next.js dashboard connects to the Go API. Set `NEXT_PUBLIC_API_URL` in
`.env.local` for development, or in your host's build environment for deployment.
It is a public backend URL, never a secret. Rebuild after changing it.

```bash
npm ci
npm run dev
```

Open `/dashboard` to create a workspace or reconnect with a saved key. Add your
own Fireworks key and tool-capable model for AI diagnosis. Keys remain in memory;
refresh requires reconnecting. The backend must allow this frontend's exact
origin through `ALLOWED_ORIGINS`.

```bash
npm run lint
npm test
npx tsc --noEmit --incremental false
npm run build
```

The dashboard's simulated example jobs execute on the backend. Actual user
workloads execute on their own machines with the Python SDK. See the
[root README](../README.md) for setup and deployment.
