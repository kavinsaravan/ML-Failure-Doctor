"""Summarize independent reviews of evaluator outputs; never infer correctness from labels."""
import argparse
import csv
import hashlib
import json
from pathlib import Path


def summarize(predictions, reviews):
    ids = [p['id'] for p in predictions]
    review_ids = [r['id'] for r in reviews]
    if not ids or any(not ident for ident in ids) or len(set(ids)) != len(ids):
        raise ValueError('Predictions must have unique, nonempty case IDs')
    if len(set(review_ids)) != len(review_ids) or set(ids) != set(review_ids):
        raise ValueError('Reviews must contain exactly one row for every prediction')
    by_id = {row['id']: row for row in reviews}
    groups = {'all': predictions}
    sources = {p['report']['source'] for p in predictions}
    if not sources <= {'ai', 'rules'}:
        raise ValueError('Report source must be ai or rules')
    for source in sorted(sources):
        groups[source] = [p for p in predictions if p['report']['source'] == source]
    for family in sorted({p.get('scenario_family', '') for p in predictions} - {''}):
        groups['family:' + family] = [p for p in predictions if p.get('scenario_family') == family]
    result = {}
    for group, cases in groups.items():
        scores = {}
        for field in ('root_cause_correct', 'remediation_correct'):
            values = []
            for case in cases:
                row = by_id[case['id']]
                value = row.get(field, '').strip()
                if value not in ('', '0', '1'):
                    raise ValueError(f'{case["id"]}: {field} must be blank, 0, or 1')
                if value:
                    if not row.get('reviewer', '').strip():
                        raise ValueError(f'{case["id"]}: a reviewer is required')
                    values.append(int(value))
            complete = len(values) == len(cases)
            scores[field] = {
                'cases': len(cases), 'reviewed': len(values), 'correct': sum(values),
                'complete': complete,
                'accuracy_percent': 100 * sum(values) / len(cases) if complete else None,
            }
        result[group] = scores
    return {
        'review_scores': result,
        'ai_report_coverage_percent': 100 * len(groups.get('ai', [])) / len(predictions),
        'limitations': 'Correctness comes from reviewer judgments, not automatic category matches. '
                       'Incomplete review sets have no accuracy score. Repeated scenario families '
                       'are correlated; execution count is not independent diagnostic diversity.',
    }


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--predictions', type=Path, required=True)
    parser.add_argument('--reviews', type=Path, required=True)
    parser.add_argument('--out', type=Path, required=True)
    args = parser.parse_args()
    raw = args.predictions.read_bytes()
    reviews_raw = args.reviews.read_bytes()
    with args.reviews.open(newline='') as file:
        result = summarize(json.loads(raw), list(csv.DictReader(file)))
    result['predictions_sha256'] = hashlib.sha256(raw).hexdigest()
    result['reviews_sha256'] = hashlib.sha256(reviews_raw).hexdigest()
    args.out.parent.mkdir(parents=True, exist_ok=True)
    with args.out.open('x') as file:
        file.write(json.dumps(result, indent=2) + '\n')
    print(json.dumps(result, indent=2))


if __name__ == '__main__':
    main()
