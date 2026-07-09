#!/usr/bin/env python3
"""
GPU OOM Failure Job
Simulates a GPU Out of Memory error during training.
"""

import time
import sys

def main():
    print("Starting training with large batch size...")
    print("Initializing model: ResNet-152...")
    time.sleep(1)

    print("Loading dataset...")
    print("Batch size: 512")
    time.sleep(1)

    print("Allocating GPU memory...")
    print("Allocated: 8192 MiB")
    print("Allocated: 12288 MiB")
    print("Allocated: 14848 MiB")
    time.sleep(0.5)

    print("\nEpoch 1/10")
    print("Processing batch 1/50...")
    time.sleep(0.3)
    print("Processing batch 2/50...")
    time.sleep(0.3)

    # Simulate OOM error
    print("\nERROR: HIP error: out of memory")
    print("RuntimeError: HIP out of memory. Tried to allocate 2048 MiB")
    print("  Current allocated: 15360 MiB")
    print("  Total available: 16384 MiB")
    print("\nTraceback (most recent call last):")
    print('  File "train.py", line 45, in train_step')
    print("    output = model(batch)")
    print("RuntimeError: HIP error: out of memory")

    return 1

if __name__ == "__main__":
    sys.exit(main())
