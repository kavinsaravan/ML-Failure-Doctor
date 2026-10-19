# Additional GPU failure scenarios — October 18, 2026

Eight new controlled Apple MPS executions extend the original eight-family
benchmark. Each completed real forward/backward/optimizer steps and sent real
GPU telemetry to an isolated local backend before the injected failure.
These executions are separate from the original 104 jobs. This is historical
evidence from the removed hardware path, not CUDA/ROCm validation.

## Results

| Measurement | Result |
| --- | --- |
| Executed scenarios with training and live telemetry verified | 8/8 |
| Live AI reports | 8/8, with no recovery retries in this run |
| Requested / serving model | `auto` / `glm-5p3` |
| Model-reviewed root-cause field correctness | 8/8 |
| Model-reviewed recommended-fix acceptability | 6/8 |
| Heuristic category matches | 8/8, all `UNKNOWN_ERROR` |
| Sequential diagnosis function latency | Median 11.86 s; range 10.31–23.03 s |

| Injected cause | Root cause | Remediation |
| --- | --- | --- |
| Forward pass under `torch.no_grad` | Pass | Pass |
| Backward on non-scalar output without upstream gradient | Pass | Pass |
| In-place mutation of a gradient-requiring leaf parameter | Pass | Fail |
| Optimizer constructed with an empty parameter list | Pass | Pass |
| Negative learning rate | Pass | Pass |
| Existing checkpoint containing invalid bytes | Pass | Pass |
| CSV has `feature,label`; code requests `target` | Pass | Pass |
| Zero divided by zero creates a non-finite loss | Pass | Fail |

The project assistant reviewed the primary `root_cause` and `recommended_fix`
fields against the frozen cases and rubric. This is an unblinded model-based
review, not independent human adjudication. It does not separately certify every
evidence or prevention statement. All eight cases remain in the denominator.

## Findings

- The in-place update report correctly identifies the error, but its primary
  remedy uses `.data.add_`, bypassing autograd checks. A local CPU counterexample
  in [data-mutation-validation.json](data-mutation-validation.json) produced
  silently incorrect zero gradients instead of the expected nonzero gradients.
  This risk is also documented in the official
  [PyTorch migration guide](https://pytorch.org/blog/pytorch-0_4_0-migration-guide/).
  The report includes a valid `no_grad` alternative, but unsafe alternatives
  still fail this rubric.
- The non-finite-loss report correctly identifies `0/0` and suggests denominator
  stabilization. It also suggests `nan_to_num` plus logging instead of stopping
  if NaN is acceptable. For this deterministic invalid training loss, that
  alternative can hide the fault without specifying that the optimizer update
  must be skipped, so it fails the frozen remediation criterion. This is a
  reviewer judgment; its conditional wording is retained in the review notes.
- The corrupt-checkpoint diagnosis correctly recommends regenerating/restoring
  a valid checkpoint and explicitly rejects disabling safe loading as a repair.
- The CSV diagnosis provides a correct column mapping. Its unspecified default
  fallback requires application-specific validation; it is noted but not scored
  as prescribing a concrete fabricated label.

The classifier returns `UNKNOWN_ERROR` for all eight novel failure types.
Matching those broad expected categories does not establish specific diagnosis
accuracy; the model review scores the actual explanation fields separately.

## Timing and limitations

The timing is measured around the production diagnosis function called by the
CLI evaluator. AI timing includes the Fireworks/network request; it excludes
dashboard interactions, backend HTTP transport, and queueing outside this call.
The same cases took less than the 1 ms timer resolution in rules mode. These are
eight sequential exploratory observations, not load testing or an SLA estimate.
See [latency.json](latency.json) for both modes and all raw observations.

These are eight deliberately instrumented cases, one per added family. Some
errors occur in Python/framework validation after GPU computation rather than
in a GPU kernel. Provenance is recorded per case. The dataset was frozen before
inference, but was designed by the project assistant familiar with this system;
it is not an independently constructed held-out evaluation. No production
diagnosis prompt or model setting was tuned for these results. Keep these scores
separate from the original repeated-case benchmark and do not infer general
accuracy from eight successes.

## Reproduce

The current collector supports CUDA/ROCm only. To audit the historical
collection, use the source revision and [collection patch](collection-code.patch)
in the [execution manifest](execution-summary.json). For new supported-hardware
runs, follow the [current measurement protocol](../../README.md).

Choose new output paths; collection and evaluation refuse to overwrite evidence.
From `backend/`, evaluate using the saved cases:

```bash
go run ./cmd/evaluate -cases ../evaluation/runs/mps-extended-8/cases.json \
  -out /tmp/crashlens-extended-new-rules -mode rules

# Uses configured Fireworks credentials and inference credits.
FIREWORKS_MODEL=auto go run ./cmd/evaluate \
  -cases ../evaluation/runs/mps-extended-8/cases.json \
  -out /tmp/crashlens-extended-new-ai -mode fireworks
```

The
[execution manifest](execution-summary.json) preserves dataset/collector hashes,
and [collection patch](collection-code.patch) records the collector changes
relative to `a635250`. Case-level judgments are in
[reviews.csv](reviews.csv), arithmetic in
[scores.json](scores.json), and the pre-inference
criteria in [review-rubric.json](review-rubric.json).

Intermediate rule predictions and blank review templates were removed; their
[classification summary](classification-summary.json) remains. The complete
original run is recoverable from Git commit `259a5b51d5d77e405a687689b67af7af0701e466`.
