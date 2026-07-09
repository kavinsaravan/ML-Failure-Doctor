#!/usr/bin/env python3
"""
GPU OOM Failure Job
Simulates a GPU Out of Memory error during training with AMD/ROCm.
"""

import time
import sys

def main():
    print("Loading transformer model...")
    print("Runtime: ROCm / HIP backend")
    time.sleep(1)

    print("Model: GPT-2 Large (774M parameters)")
    print("Backend: ROCm 5.7.0")
    print("Device: AMD Radeon MI250X")
    time.sleep(0.5)

    print("\nStarting training with batch_size=128")
    print("Initializing optimizer...")
    time.sleep(0.5)

    print("\nGPU memory allocation:")
    print("GPU memory used: 18432MB / 24576MB")
    time.sleep(0.3)
    print("GPU memory used: 21504MB / 24576MB")
    time.sleep(0.3)
    print("GPU memory used: 23800MB / 24576MB")
    time.sleep(0.5)

    print("\nEpoch 1/10")
    print("Processing batch 1/100...")
    time.sleep(0.3)
    print("Processing batch 2/100...")
    time.sleep(0.3)

    # Simulate OOM error with AMD/ROCm specifics
    print("\nRuntimeError: HIP out of memory. Tried to allocate 2048 MiB.")
    print("GPU memory used: 23800MB / 24576MB")
    print("\nTraceback (most recent call last):")
    print('  File "train.py", line 87, in train_step')
    print("    loss = model(input_ids, labels=labels)")
    print('  File "/opt/rocm/lib/python3.9/site-packages/torch/nn/modules/module.py", line 1501, in _call_impl')
    print("    return forward_call(*args, **kwargs)")
    print("RuntimeError: HIP error: out of memory")

    return 1

if __name__ == "__main__":
    sys.exit(main())
