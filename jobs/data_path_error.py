#!/usr/bin/env python3
"""
Data Path Error Failure Job
Simulates a failure when training data file is not found.
"""

import time
import sys

def main():
    print("Starting training job...")
    print("Runtime: ROCm / HIP backend")
    print("Device: AMD Radeon MI100")
    time.sleep(0.5)

    print("\nInitializing model: BERT-base...")
    print("Model parameters: 110M")
    time.sleep(1)

    print("\nLoading dataset from /data/train.csv")
    print("Dataset path: /data/train.csv")
    print("Expected format: CSV with columns [text, label]")
    time.sleep(0.5)

    # Simulate file not found error
    print("\nFileNotFoundError: Dataset path /data/train.csv does not exist")
    print("[Errno 2] No such file or directory: '/data/train.csv'")
    print("\nTraceback (most recent call last):")
    print('  File "train.py", line 67, in load_dataset')
    print('    df = pd.read_csv(data_path)')
    print('  File "/usr/local/lib/python3.9/site-packages/pandas/io/parsers/readers.py", line 912, in read_csv')
    print('    return _read(filepath_or_buffer, kwds)')
    print("FileNotFoundError: [Errno 2] No such file or directory: '/data/train.csv'")
    print("\nPlease verify:")
    print("  1. Dataset path is correct")
    print("  2. Dataset has been downloaded and preprocessed")
    print("  3. Mount points are properly configured")
    print("  4. File permissions allow read access")

    return 1

if __name__ == "__main__":
    sys.exit(main())
