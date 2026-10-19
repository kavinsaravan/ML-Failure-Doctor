import importlib.util
from pathlib import Path
import unittest


spec = importlib.util.spec_from_file_location(
    'score_diagnoses', Path(__file__).resolve().parents[1] / 'score_diagnoses.py')
scorer = importlib.util.module_from_spec(spec)
spec.loader.exec_module(scorer)


class ReviewScoringTests(unittest.TestCase):
    def setUp(self):
        self.predictions = [{'id': 'a', 'report': {'source': 'ai'}},
                            {'id': 'b', 'report': {'source': 'rules'}}]
        self.reviews = [{'id': 'a', 'root_cause_correct': '1',
                         'remediation_correct': '0', 'reviewer': 'Reviewer A'},
                        {'id': 'b', 'root_cause_correct': '',
                         'remediation_correct': '', 'reviewer': ''}]

    def test_partial_reviews_cannot_produce_overall_accuracy(self):
        result = scorer.summarize(self.predictions, self.reviews)
        self.assertIsNone(result['review_scores']['all']['root_cause_correct']['accuracy_percent'])
        self.assertEqual(result['ai_report_coverage_percent'], 50)
        self.assertEqual(result['review_scores']['ai']['remediation_correct']['accuracy_percent'], 0)

    def test_complete_reviews_count_incorrect_cases_in_denominator(self):
        self.reviews[1].update(root_cause_correct='0', remediation_correct='1', reviewer='Reviewer A')
        result = scorer.summarize(self.predictions, self.reviews)
        self.assertEqual(result['review_scores']['all']['root_cause_correct']['accuracy_percent'], 50)
        self.assertEqual(result['review_scores']['all']['remediation_correct']['accuracy_percent'], 50)

    def test_duplicate_missing_and_unknown_reviews_are_rejected(self):
        for rows in [self.reviews[:1], [self.reviews[0], self.reviews[0]],
                     self.reviews + [{'id': 'extra'}]]:
            with self.assertRaises(ValueError):
                scorer.summarize(self.predictions, rows)

    def test_scores_require_a_reviewer_and_binary_judgment(self):
        self.reviews[0]['reviewer'] = ''
        with self.assertRaises(ValueError):
            scorer.summarize(self.predictions, self.reviews)
        self.reviews[0].update(reviewer='Reviewer A', root_cause_correct='probably')
        with self.assertRaises(ValueError):
            scorer.summarize(self.predictions, self.reviews)

    def test_empty_ids_and_invalid_sources_are_rejected(self):
        self.predictions[0]['id'] = ''
        with self.assertRaises(ValueError):
            scorer.summarize(self.predictions, self.reviews)
        self.predictions[0].update(id='a', report={'source': 'all'})
        with self.assertRaises(ValueError):
            scorer.summarize(self.predictions, self.reviews)

    def test_family_scores_preserve_repeated_trial_denominators(self):
        for prediction in self.predictions:
            prediction['scenario_family'] = 'oom'
        self.reviews[1].update(root_cause_correct='0', remediation_correct='1', reviewer='Reviewer A')
        result = scorer.summarize(self.predictions, self.reviews)
        score = result['review_scores']['family:oom']['root_cause_correct']
        self.assertEqual(score['cases'], 2)
        self.assertEqual(score['accuracy_percent'], 50)


if __name__ == '__main__':
    unittest.main()
