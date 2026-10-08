import json
import tempfile
import unittest
from pathlib import Path
from plot_complete_history import load_trials, write_csv

class HistoryTests(unittest.TestCase):
    def test_failed_reported_score_and_rejected_records(self):
        with tempfile.TemporaryDirectory() as temp:
            base = Path(temp)
            records = [
                dict(status='passed', score=100, decision='accepted'),
                dict(status='failed', score=None, reported_score=9999),
                dict(status='passed', score=120, decision='rejected'),
                dict(status='passed', score=120),
                dict(status='aborted', score=None),
            ]
            for i, row in enumerate(records):
                p = base / str(i); p.mkdir()
                row['started_at'] = f'20261008T15000{i}Z'
                (p/'result.json').write_text(json.dumps(row))
            rows = load_trials(base, base/'missing.json')
            self.assertEqual(len(rows), 5)
            self.assertEqual([r['trial'] for r in rows if r['record_high']], [1, 3])
            self.assertEqual(rows[2]['decision'], 'rejected')
            write_csv(base/'all.csv', rows)
            self.assertIn('9999', (base/'all.csv').read_text())
            self.assertFalse(rows[1]['valid'])
    def test_invalid_passed_score_rejected(self):
        with tempfile.TemporaryDirectory() as temp:
            base = Path(temp); (base/'a').mkdir()
            (base/'a/result.json').write_text(json.dumps(dict(started_at='20261008T150000Z',status='passed',score=None)))
            with self.assertRaises(ValueError): load_trials(base, base/'missing.json')

if __name__ == '__main__': unittest.main()
