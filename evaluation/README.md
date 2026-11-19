# Evaluation Results

**Root-cause correctness: 94.23% — 98 of 104 cases passed the model review.** These scores
come from an unblinded project-assistant review of eight controlled scenario
families, each repeated 13 times; independent human review remains pending.

| Historical evaluation | Recorded result |
| --- | --- |
| Original execution benchmark | 104 tracked jobs, eight families repeated 13 times |
| Category classification | 104/104 expected category matches, including 39 `UNKNOWN_ERROR` cases |
| Fireworks `auto`, including recovery | 104/104 AI reports, served by `glm-5p3` |
| Model-reviewed root-cause correctness | **99/104 (94.23%)** |
| Additional eight-scenario benchmark | Eight jobs; model-reviewed root cause 8/8, remediation 6/8 |
| Sequential diagnosis latency on eight added cases | Median 11.86 s, range 10.31–23.03 s |

Measure execution count, category classification, root-cause correctness, and
user impact separately. Current collectors support NVIDIA CUDA and AMD ROCm.
Use a GPU-compatible PyTorch build; both use the `cuda` device namespace.
CPU and simulated telemetry cannot pass hardware validation.