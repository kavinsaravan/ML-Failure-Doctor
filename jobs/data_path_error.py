#!/usr/bin/env python3
"""
Data Path Error Failure Job
Simulates a failure when training data file is not found.
"""

import time
import sys

def main():
    print("Starting training job...")
    print("Initializing model...")
    time.sleep(1)

    print("Loading dataset from path...")
    print("Data path: /data/imagenet/train")
    time.sleep(0.5)

    # Simulate file not found error
    print("\nERROR: Training data not found")
    print("FileNotFoundError: [Errno 2] No such file or directory: '/data/imagenet/train'")
    print("\nTraceback (most recent call last):")
    print('  File "train.py", line 34, in load_dataset')
    print('    dataset = ImageFolder(data_path)')
    print("FileNotFoundError: No such file or directory: '/data/imagenet/train'")
    print("\nPlease verify:")
    print("  1. Data path is correct")
    print("  2. Dataset has been downloaded")
    print("  3. Mount points are properly configured")

    return 1

if __name__ == "__main__":
    sys.exit(main())
