#!/usr/bin/env python3
"""
Timeout Failure Job
Simulates a job that stalls and times out during execution.
"""

import time
import sys

def main():
    print("Starting training job...")
    print("Runtime: ROCm / HIP backend")
    print("Device: AMD Radeon MI250")
    time.sleep(0.5)

    print("\nInitializing model: LLaMA-7B...")
    print("Model parameters: 7B")
    print("Precision: FP16")
    time.sleep(1)

    print("\nStarting dataloader...")
    print("Dataset: Custom text corpus")
    print("Batch size: 64")
    print("Number of workers: 8")
    time.sleep(1)

    print("\nEpoch 1/50")
    print("Processing batch 1/5000...")
    time.sleep(0.5)
    print("Batch 1 completed (12.3s)")
    time.sleep(0.3)

    print("\nWaiting for batch...")
    print("Loading batch 2/5000...")
    time.sleep(1)
    print("Dataloader: waiting for data preprocessing...")
    time.sleep(1)
    print("Dataloader: still waiting...")
    time.sleep(1)
    print("Dataloader: no progress detected...")
    time.sleep(1)

    # Simulate stalled job timeout
    print("\nNo progress detected for 60 seconds")
    print("Training appears to be stalled")
    print("\nTimeoutError: Job execution timeout")
    print("Job exceeded maximum idle time of 60 seconds")
    print("Current runtime: 0h 2m 45s")
    print("Last progress: 0h 1m 15s ago")
    print("\nTraceback (most recent call last):")
    print('  File "train.py", line 234, in train_epoch')
    print("    batch = next(dataloader_iterator)")
    print('  File "/usr/local/lib/python3.9/site-packages/torch/utils/data/dataloader.py", line 681, in __next__')
    print("    data = self._next_data()")
    print("TimeoutError: DataLoader worker timeout - possible deadlock in data preprocessing")

    return 1

if __name__ == "__main__":
    sys.exit(main())
