# Historical 104-job benchmark and model review

This is a **model-based review by the project assistant**, not an independent
human review. No new Fireworks requests were made. The existing predictions
were frozen by SHA-256 before the judgments were written.

## Results

| Measurement | Correct / reviewed | Score |
| --- | --- | --- |
| Root-cause field correctness | 97 / 104 | 93.27% |
| Recommended-fix acceptability | 79 / 104 | 75.96% |

Each case completed GPU training and live telemetry on Apple MPS before an
injected failure. This hardware path has since been removed; these results do
not validate the current CUDA/ROCm version.

Every case received separate binary judgments and supporting notes. All 104
final reports are AI reports from `auto`, served by `glm-5p3`.
[inference-summary.json](inference-summary.json) records failed attempts and
recovery; all original intermediate output is recoverable from Git commit
`259a5b51d5d77e405a687689b67af7af0701e466`. Category matches and report
availability were not used as correctness scores.

| Scenario family | Root cause | Remediation |
| --- | --- | --- |
| allocation-256 | 13/13 | 1/13 |
| allocation-384 | 12/13 | 0/13 |
| checkpoint | 12/13 | 13/13 |
| dataset | 8/13 | 13/13 |
| dependency | 13/13 | 13/13 |
| dtype-or-device | 13/13 | 13/13 |
| matmul-shape | 13/13 | 13/13 |
| reshape-size | 13/13 | 13/13 |

## What was reviewed

I read every `root_cause` and `recommended_fix` field against the known injected
cause and the recorded exception, GPU training, telemetry, and tensor parameters.
The logs, hardware flags, and tensor parameters remain in [cases.json](cases.json).
The original rubric was frozen in [final/review-rubric.json](final/review-rubric.json)
before judgments. Per-case reasons are retained in [final/reviews.csv](final/reviews.csv).

Root cause passes only when the explanation identifies the cause without a
material factual contradiction. Minor wording errors and ambiguous references
to later evaluation work are noted, rather than automatically treated as denial
of the earlier GPU training. Explicit claims that training or meaningful GPU
work had not occurred fail when logs show completed forward/backward/optimizer
steps. Notes identify these interpretation choices for a second reviewer.

A recommended fix passes when it contains at least one specific remedy for the
injected cause and no unsafe advice. Extra ineffective snippets do not erase an
otherwise valid primary fix under this rubric, but are noted. Suggesting disabled
MPS memory limits fails even when caveated or offered alongside safe alternatives.

The evidence and prevention fields have **not** been separately certified. This
is a scoped review of the primary explanation and remediation fields, not proof
that every statement in each complete report is correct.

## Failures

Seven root-cause judgments failed:

- `mps-dataset-trial-001`: Identifies missing absent_training_dataset.csv in the process working directory and provides restoration/path correction as a concrete remedy. Root additionally claims failure preceded training, computation, or meaningful workload/GPU work, contradicting the recorded completed GPU forward/backward/optimizer steps; root fails the consistency criterion. It simultaneously acknowledges those completed steps, making the before-any-training claim internally contradictory. The optional misindentation suspicion in the fix is unsupported, but a valid file-restoration remedy is provided.
- `mps-dataset-trial-006`: Identifies missing absent_training_dataset.csv in the process working directory and provides restoration/path correction as a concrete remedy. Root additionally claims failure preceded training, computation, or meaningful workload/GPU work, contradicting the recorded completed GPU forward/backward/optimizer steps; root fails the consistency criterion.
- `mps-dataset-trial-008`: Identifies missing absent_training_dataset.csv in the process working directory and provides restoration/path correction as a concrete remedy. Root additionally claims failure preceded training, computation, or meaningful workload/GPU work, contradicting the recorded completed GPU forward/backward/optimizer steps; root fails the consistency criterion. The optional default-dataset fallback would require application-specific validation; the explicit restore/correct-path remedy is valid.
- `mps-checkpoint-trial-009`: Identifies missing absent_resume_checkpoint.pt at the requested path; restoring/providing the file or correcting its path addresses the injected FileNotFoundError. Root narrative incorrectly asserts failure preceded any meaningful GPU work, contradicting the verified forward/backward/optimizer steps already in the log; root score fails despite identifying the missing file.
- `mps-dataset-trial-010`: Identifies missing absent_training_dataset.csv in the process working directory and provides restoration/path correction as a concrete remedy. Root additionally claims failure preceded training, computation, or meaningful workload/GPU work, contradicting the recorded completed GPU forward/backward/optimizer steps; root fails the consistency criterion.
- `mps-dataset-trial-012`: Identifies missing absent_training_dataset.csv in the process working directory and provides restoration/path correction as a concrete remedy. Root additionally claims failure preceded training, computation, or meaningful workload/GPU work, contradicting the recorded completed GPU forward/backward/optimizer steps; root fails the consistency criterion.
- `mps-allocation-384-trial-013`: Root identifies the limit breach but incorrectly asserts the fixed 419430400-byte request is driven by width/batch; the collector uses a constant allocation independent of those training dimensions. Fix suggests disabling MPS safety limits despite cautions, violating the remediation rubric. Claims about total/free machine memory overinterpret process-scoped/recommended-working-set telemetry.

Twenty-five remediation judgments failed because the recommended fix suggests
disabling MPS memory safety limits. The earlier literal text search flagged only
20; case-by-case reading found five additional phrasings (such as `or 0.0` and
`e.g., 0.0 disables`). All 26 allocation cases were read; one warned about zero
without recommending disabling it and provided a concrete safe allocation fix.

## Interpretation

The root-cause score exceeds 90% **on this controlled dataset under this model
review**. It does not independently validate a general 90% diagnostic-accuracy
claim. The 104 trials repeat eight known scenario families on Apple MPS and
were not a diverse held-out test. The reviewer knows the project and expected
causes; some temporal and causal statements require judgment. Independent human
adjudication may change the score.

The 100+ execution claim remains supported for sequential local controlled GPU
jobs; it does not establish production users or concurrent capacity.

## Reproduce scoring

From the repository root:

```bash
python3 scripts/score_diagnoses.py \
  --predictions evaluation/runs/mps-104/final/predictions.json \
  --reviews evaluation/runs/mps-104/final/reviews.csv \
  --out /tmp/crashlens-model-review-new-summary.json
```

Use a new output filename; the scorer refuses to overwrite evidence. It
reproduces the arithmetic, not the subjective judgments. The preserved
[final/scores.json](final/scores.json) includes hashes of the unchanged predictions
and completed review CSV. Independent human review is still pending.

## Preserved evidence

- [Execution manifest](execution-summary.json): hardware, job count, timestamps, and dataset/source hashes.
- [Dataset](cases.json): original logs, metrics, labels, and hardware verification flags.
- [Final predictions](final/predictions.json): complete 104-case AI reports, unchanged byte-for-byte.
- [Classification summary](classification-summary.json): original heuristic category results.
- [Inference and recovery summary](inference-summary.json) and [two sequential recovery records](sequential-recovery.jsonl).
- [Collection patch](collection-code.patch) and [evaluation patch](evaluation-code.patch): historical changes against the recorded source revision.

Blank review templates, per-family copies of judgments, shards, debug probes,
and intermediate combined results were removed from the current tree. They
remain in Git commit `259a5b51d5d77e405a687689b67af7af0701e466`. To export the original
intermediate run from the repository root:

```bash
git archive 259a5b51d5d77e405a687689b67af7af0701e466 evaluation/runs/verification-a635250 > /tmp/crashlens-intermediates.tar
```
