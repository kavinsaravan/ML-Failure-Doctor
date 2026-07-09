#!/usr/bin/env python3
"""
Dependency Error Failure Job
Simulates a missing Python package import error.
"""

import time
import sys

def main():
    print("Starting training job...")
    print("Importing dependencies...")
    time.sleep(0.5)

    print("import torch")
    print("import numpy as np")
    print("import transformers")
    time.sleep(0.3)

    # Simulate import error
    print("\nERROR: Failed to import required package")
    print("ModuleNotFoundError: No module named 'transformers'")
    print("\nTraceback (most recent call last):")
    print('  File "train.py", line 7, in <module>')
    print("    from transformers import AutoModel, AutoTokenizer")
    print("ModuleNotFoundError: No module named 'transformers'")
    print("\nPlease install required packages:")
    print("  pip install transformers")

    return 1

if __name__ == "__main__":
    sys.exit(main())
