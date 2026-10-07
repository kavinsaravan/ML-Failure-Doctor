# Example notebooks

Create a private workspace in the dashboard first. Notebook configuration prompts
for the reachable backend URL and your saved CrashLens key. Optional Fireworks
credentials enable AI diagnoses billed to your account; otherwise reports use rules.
Keys are never included in saved notebook source or outputs. Clear any output that
your own code prints before sharing a notebook.

- Root CrashLens_Demo.ipynb: small real CUDA training and controlled failures.
- examples/CrashLens_E2E_Test.ipynb: CUDA training, an oversized allocation failure,
  a missing dependency, and authenticated workload listing.
- SDK 01_quickstart.ipynb: real CPU computation and a missing-file failure.
- SDK 02_magic_commands.ipynb: tracked IPython cells and an actual import failure.
- SDK 03_pytorch_training.ipynb: small training on CUDA/ROCm, MPS, or CPU, with
  device selection printed explicitly.

Installation uses the Git repository; the package is not claimed to be published
on PyPI. CUDA tests require a GPU runtime. CPU runs do not validate GPU support.
OOM examples attempt one allocation larger than total CUDA memory rather than
consuming memory in an unbounded loop. Failed allocations may leave sampled usage
low. Training uses generated data, so these are functional tests, not evidence of
production scale or measured diagnosis accuracy.

Run cells in order, use the workload IDs returned by your runs, and inspect their
reports in your own dashboard. For a strict hardware check use
scripts/validate_gpu.py from the repository root. See the main README for details.
