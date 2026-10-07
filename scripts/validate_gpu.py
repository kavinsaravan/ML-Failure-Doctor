"""Require real CUDA/ROCm/MPS training and live telemetry; never pass on CPU/simulation."""
import argparse
import json
import os
from pathlib import Path
import sys
import time

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "crashlens-sdk"))

def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--api-url", default=os.getenv("CRASHLENS_URL", "http://localhost:8080"))
    parser.add_argument("--device", choices=["auto", "cuda", "mps"], default="auto")
    args = parser.parse_args()
    try:
        import torch
        import requests
        from crashlens import WorkloadTracker
    except ImportError as error:
        print(f"Hardware validation unavailable: install GPU-compatible PyTorch and the SDK ({error}).", file=sys.stderr)
        return 2
    cuda_available = torch.cuda.is_available()
    mps_available = hasattr(torch.backends, "mps") and torch.backends.mps.is_available()
    device = args.device
    if device == "auto":
        device = "cuda" if cuda_available else "mps"
    if not (cuda_available if device == "cuda" else mps_available):
        print(f"Hardware validation unavailable: no {device} GPU. CPU fallback is refused.", file=sys.stderr)
        return 2
    if device == "mps" and os.getenv("PYTORCH_ENABLE_MPS_FALLBACK") == "1":
        print("Unset PYTORCH_ENABLE_MPS_FALLBACK for hardware validation.", file=sys.stderr)
        return 2
    backend = "MPS" if device == "mps" else ("ROCm" if torch.version.hip else "CUDA")
    sources = ("torch.mps",) if device == "mps" else ("torch.cuda", "nvidia-ml-py")
    synchronize = torch.mps.synchronize if device == "mps" else torch.cuda.synchronize
    tracker = WorkloadTracker(args.api_url, api_key=os.getenv("CRASHLENS_API_KEY"))
    observed_live = False
    with tracker.track(f"Real {backend} GPU validation") as ident:
        model = torch.nn.Linear(1024, 512).to(device)
        optimizer = torch.optim.SGD(model.parameters(), lr=.01)
        inputs = torch.randn(64, 1024, device=device)
        deadline = time.monotonic() + 10
        next_check = time.monotonic() + 3
        while time.monotonic() < deadline:
            optimizer.zero_grad(); loss = model(inputs).square().mean(); loss.backward(); optimizer.step()
            synchronize()
            if time.monotonic() >= next_check:
                response = requests.get(f"{tracker.api_url}/workloads/{ident}", headers=tracker.headers, timeout=10)
                response.raise_for_status(); workload = response.json()
                samples = json.loads(workload.get("gpu_metrics") or "[]")
                observed_live |= workload["status"] == "running" and any(
                    sample.get("source") in sources and sample.get("gpu_memory_used_mb", 0) > 0 for sample in samples
                )
                next_check = time.monotonic() + 1
            time.sleep(.01)
        if not observed_live:
            raise RuntimeError("No real GPU readings reached the API during execution")
    print(f"PASS: real {backend} training and live GPU metrics, workload {ident}")
    return 0

if __name__ == "__main__":
    raise SystemExit(main())
