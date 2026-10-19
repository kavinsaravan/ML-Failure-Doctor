# Evaluation results and measurement protocol

Measure execution count, category classification, root-cause correctness, and
user impact separately. Current collectors support NVIDIA CUDA and AMD ROCm.
Use a GPU-compatible PyTorch build; both use the `cuda` device namespace.
CPU and simulated telemetry cannot pass hardware validation.

## Historical results

**Root-cause correctness: 93.27% — 97 of 104 cases passed the model review.**
Recommended-fix acceptability was **75.96% — 79 of 104 cases**. These scores
come from an unblinded project-assistant review of eight controlled scenario
families, each repeated 13 times; independent human review remains pending.

| Historical evaluation | Recorded result | Evidence |
| --- | --- | --- |
| Original execution benchmark | 104 tracked jobs, eight families repeated 13 times | [Manifest](runs/mps-104/execution-summary.json) |
| Category classification | 104/104 expected category matches, including 39 `UNKNOWN_ERROR` cases | [Results](runs/mps-104/classification-summary.json) |
| Fireworks `auto`, including recovery | 104/104 AI reports, served by `glm-5p3` | [Run](runs/mps-104/inference-summary.json) |
| Model-reviewed root-cause correctness | **97/104 (93.27%)** | [Scores](runs/mps-104/final/scores.json) |
| Model-reviewed recommended-fix acceptability | **79/104 (75.96%)** | [Rubric and per-case notes](runs/mps-104/README.md) |
| Additional eight-scenario benchmark | Eight jobs; model-reviewed root cause 8/8, remediation 6/8 | [Results](runs/mps-extended-8/README.md) |
| Sequential diagnosis latency on eight added cases | Median 11.86 s, range 10.31–23.03 s | [Observations](runs/mps-extended-8/latency.json) |

Reviews were model-based and unblinded, covering primary `root_cause` and
`recommended_fix` fields only. Independent human validation remains pending.
Repeated controlled scenarios do not establish general diagnostic accuracy,
production users, or concurrent-job capacity. New runs do not establish usage
before their collection date. No actual CUDA/ROCm hardware benchmark is recorded
here yet.

The repository keeps two compact benchmark folders containing datasets, final
predictions, completed reviews, rubrics, scores, source patches, and manifests.
Shards, probes, blank reviews, and intermediate results are preserved in Git
commit `259a5b51d5d77e405a687689b67af7af0701e466`, rather than the current tree.
The 104-case [inference summary](runs/mps-104/inference-summary.json) retains
failure/recovery counts, and its [benchmark report](runs/mps-104/README.md)
includes an archive command to recover the full intermediate run.

## Real CUDA/ROCm execution count

From the repository root on a compatible GPU host, choose a new output path:

```bash
python scripts/validate_gpu.py --device cuda
python scripts/collect_gpu_cases.py --device cuda --local-backend --repeats 13 \
  --out evaluation/runs/cuda-original-new/cases.json
```

The validator needs an accessible backend and a workspace key in
`CRASHLENS_API_KEY`; set `CRASHLENS_URL` as needed. The collector's
`--local-backend` option builds an isolated temporary API/database and does not
contact a deployed service. Collection makes no Fireworks requests. On AMD,
install ROCm-compatible PyTorch and use the same `--device cuda` argument.

The original suite has eight scenario families; repeating each 13 times creates
104 sequential small controlled jobs. Each job performs forward, backward, and
optimizer steps, verifies real telemetry reached the API while running, and
then triggers a controlled failure. Repetitions are not independent causes or
concurrent-job capacity. Records include local workload ID, trial, hardware
platform, logs, metrics, timestamps, and verification flags. IDs are local to
each isolated database, not globally unique between runs.

Use `--suite extended` for disabled gradient tracking, non-scalar backward,
in-place leaf mutation, empty optimizer, negative learning rate, corrupt
checkpoint, missing CSV column, and non-finite loss. Some are Python/framework
validation errors after GPU work rather than GPU kernel errors.

```bash
python scripts/collect_gpu_cases.py --device cuda --local-backend \
  --suite extended --out evaluation/runs/cuda-extended-new/cases.json
```

`--suite original` is the default; `--suite all` runs both suites. Preserve source
revision, any uncommitted collector changes, and dataset hashes with new runs.
Collectors and evaluators refuse to overwrite existing evidence.

## Diagnosis evaluation

From `backend/`, use the newly collected dataset and new output directories:

```bash
go run ./cmd/evaluate -cases ../evaluation/runs/cuda-original-new/cases.json \
  -out ../evaluation/runs/cuda-original-new/rules -mode rules

# Uses inference credits; credentials belong in the environment, not arguments.
FIREWORKS_MODEL=auto go run ./cmd/evaluate \
  -cases ../evaluation/runs/cuda-original-new/cases.json \
  -out ../evaluation/runs/cuda-original-new/fireworks -mode fireworks
```

AI evaluation requires `FIREWORKS_API_KEY`. The default model is `auto`; pin a
specific tool-capable model with `FIREWORKS_MODEL` if desired. Reports record
requested/serving models. Preserve all predictions, fallbacks, retries, serving
models, dataset hashes, source revision, and run date. Routing pools can change.
Expected labels and review rubrics are not passed to inference.

Category matches score the production heuristic classifier in both modes, not
AI explanation correctness. A fallback is not a successful AI report.

Have an independent reviewer complete every row of generated `review.csv`:

- `root_cause_correct=1`: identifies the specific observed cause, stays consistent
  with logs/metrics, and avoids a contradictory causal history. A broad category
  alone is insufficient when a specific cause is observable. Otherwise use `0`.
- `remediation_correct=1`: offers a concrete fix addressing the cause without
  unsafe advice. Otherwise use `0`.
- Record reviewer identity and supporting notes. Leave unreviewed judgments
  blank and keep failures/fallbacks in the denominator. Label model reviews
  explicitly rather than presenting them as independent human review.

From the repository root:

```bash
python scripts/score_diagnoses.py \
  --predictions evaluation/runs/cuda-original-new/fireworks/predictions.json \
  --reviews evaluation/runs/cuda-original-new/fireworks/review.csv \
  --out evaluation/runs/cuda-original-new/fireworks/review-summary.json
```

The scorer suppresses accuracy for incomplete reviews. Freeze a diverse held-out
dataset before tuning to evaluate generalization; include ambiguous and
unsupported failures. Report denominator, response coverage, scenario diversity,
hardware, date, and reviewer provenance. 
