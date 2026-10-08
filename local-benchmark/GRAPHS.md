# Complete score history

Regenerate: python3 local-benchmark/tools/plot_complete_history.py
Inputs: every results/*/result.json, ordered by started_at and directory name.
Outputs: score-all-trials.{png,svg}, score-record-highs.{png,svg}, all-trials.csv, record-highs.csv.
Failed/aborted/preparation-failed trials have no score and are shown in a separate failure band.
Rejected but valid candidates remain in the complete history and any record highs.
No stage aggregation or omitted trials. Observed record highs are not a list of adopted changes.
