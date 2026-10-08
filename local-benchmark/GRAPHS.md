# Complete score history

Regenerate: local-benchmark/.venv/bin/python local-benchmark/tools/plot_complete_history.py
Inputs: every results/*/result.json, ordered by started_at and directory name.
Outputs: score-all-trials.{png,svg}, score-record-highs.{png,svg}, all-trials.csv, record-highs.csv.
Failed/aborted/preparation-failed trials have no score and are shown in a separate failure band.
Rejected but valid candidates remain in the complete history and any record highs.
No stage aggregation or omitted trials. Observed record highs are not a list of adopted changes.

Final:44 trials,35 passed,9 failed/aborted/preparation failures,18 strict record highs. Diagnostic runs are gray, rejected candidates orange; no profiler score is claimed as an adopted gain. Original source commits are retained, with public_source_commit for the documented metadata-only equivalent amendment.
