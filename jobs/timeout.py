#!/usr/bin/env python3
"""
Timeout Failure Job
Simulates a job that times out during execution.
"""

import time
import sys

def main():
    print("Starting long-running training job...")
    print("Initializing model: GPT-2-XL...")
    print("Warning: This is a large model, training may take several hours")
    time.sleep(1)

    print("Loading dataset...")
    print("Dataset size: 50GB")
    time.sleep(1)

    print("\nEpoch 1/100")
    print("Processing batch 1/10000...")
    time.sleep(0.5)
    print("Processing batch 2/10000...")
    print("Average time per batch: 45 seconds")
    time.sleep(0.5)

    print("\nEstimated time remaining: 125 hours")
    time.sleep(0.3)

    # Simulate timeout
    print("\nERROR: Job execution timeout")
    print("TimeoutError: Job exceeded maximum runtime limit of 2 hours")
    print("Current runtime: 2h 0m 15s")
    print("Maximum allowed: 2h 0m 0s")
    print("\nTraceback (most recent call last):")
    print('  File "runner.py", line 156, in execute')
    print("    result = train_loop()")
    print("TimeoutError: deadline exceeded")

    return 1

if __name__ == "__main__":
    sys.exit(main())
