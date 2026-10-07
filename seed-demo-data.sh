#!/usr/bin/env bash
# Seed explicitly simulated jobs into the connected workspace, never a fixed host.
set -euo pipefail
: "${CRASHLENS_URL:?Set CRASHLENS_URL to your backend URL}"
: "${CRASHLENS_API_KEY:?Set CRASHLENS_API_KEY to your workspace key}"
python3 - <<'INNER'
import json, os, urllib.request
url = os.environ["CRASHLENS_URL"].rstrip("/")
for template in ("gpu_oom", "successful", "dependency_error", "timeout", "missing_checkpoint", "data_path_error"):
    request = urllib.request.Request(url + "/workloads/run", method="POST",
        data=json.dumps({"template": template, "type": "ML_JOB"}).encode(),
        headers={"Content-Type": "application/json", "Authorization": "Bearer " + os.environ["CRASHLENS_API_KEY"]})
    with urllib.request.urlopen(request, timeout=15) as response:
        result = json.load(response)
    print(f"Queued simulated {template}: workload {result['workload_id']}")
print("Open your dashboard to view the simulated jobs. This does not test GPU hardware.")
INNER
