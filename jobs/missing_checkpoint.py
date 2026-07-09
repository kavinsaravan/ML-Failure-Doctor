#!/usr/bin/env python3
"""
Missing Checkpoint Failure Job
Simulates a failure when trying to resume from a non-existent checkpoint.
"""

import time
import sys

def main():
    print("Starting training job (resume mode)...")
    print("Initializing model...")
    time.sleep(1)

    print("Loading dataset...")
    time.sleep(1)

    print("Attempting to resume from checkpoint...")
    print("Checkpoint path: ./checkpoints/model_epoch_50.pt")
    time.sleep(0.5)

    # Simulate missing checkpoint error
    print("\nERROR: Checkpoint not found")
    print("FileNotFoundError: [Errno 2] No such file or directory: './checkpoints/model_epoch_50.pt'")
    print("\nTraceback (most recent call last):")
    print('  File "train.py", line 78, in load_checkpoint')
    print('    checkpoint = torch.load(checkpoint_path)')
    print("FileNotFoundError: Checkpoint file does not exist")

    return 1

if __name__ == "__main__":
    sys.exit(main())
