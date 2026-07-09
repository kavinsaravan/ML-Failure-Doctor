#!/usr/bin/env python3
"""
Missing Checkpoint Failure Job
Simulates a failure when trying to resume from a non-existent checkpoint.
"""

import time
import sys

def main():
    print("Starting training job (resume mode)...")
    print("Runtime: ROCm / HIP backend")
    print("Device: AMD Radeon MI210")
    time.sleep(1)

    print("\nInitializing model: ResNet-50...")
    print("Model parameters: 25.6M")
    time.sleep(0.5)

    print("\nLoading dataset...")
    print("Dataset: ImageNet (train split)")
    time.sleep(1)

    print("\nResuming from checkpoint: /checkpoints/latest.pt")
    print("Checkpoint path: /checkpoints/latest.pt")
    time.sleep(0.5)

    # Simulate missing checkpoint error
    print("\nFileNotFoundError: checkpoint not found")
    print("[Errno 2] No such file or directory: '/checkpoints/latest.pt'")
    print("\nTraceback (most recent call last):")
    print('  File "train.py", line 123, in resume_training')
    print('    checkpoint = torch.load(checkpoint_path, map_location="cuda")')
    print('  File "/opt/rocm/lib/python3.9/site-packages/torch/serialization.py", line 791, in load')
    print('    with _open_file_like(f, "rb") as opened_file:')
    print("FileNotFoundError: [Errno 2] No such file or directory: '/checkpoints/latest.pt'")

    return 1

if __name__ == "__main__":
    sys.exit(main())
