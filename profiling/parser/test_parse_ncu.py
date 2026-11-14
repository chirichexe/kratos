#!/usr/bin/env python3

from __future__ import annotations

import json
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

from parse_ncu import BOUND_COMPUTE, BOUND_MEMORY, BOUND_MIXED, BOUND_UNKNOWN, classify, parse_file


TESTDATA = Path(__file__).resolve().parent / "testdata"
SCRIPT = Path(__file__).resolve().parent / "parse_ncu.py"


class ParseNcuTests(unittest.TestCase):
    def test_compute_bound_sample(self) -> None:
        summary = parse_file(TESTDATA / "compute_bound.csv")

        self.assertEqual(summary["boundType"], BOUND_COMPUTE)
        self.assertEqual(summary["metrics"]["smThroughput"], "82.4")
        self.assertEqual(summary["metrics"]["dramThroughput"], "31.2")
        self.assertEqual(summary["metrics"]["l2Throughput"], "40")
        self.assertEqual(summary["metrics"]["achievedOccupancy"], "0.71")
        self.assertEqual(summary["metrics"]["memoryStallPct"], "12.5")

    def test_memory_bound_sample(self) -> None:
        summary = parse_file(TESTDATA / "memory_bound.csv")

        self.assertEqual(summary["boundType"], BOUND_MEMORY)
        self.assertEqual(summary["metrics"]["dramThroughput"], "84.5")
        self.assertEqual(summary["metrics"]["memoryStallPct"], "37")

    def test_mixed_sample(self) -> None:
        summary = parse_file(TESTDATA / "mixed.csv")

        self.assertEqual(summary["boundType"], BOUND_MIXED)
        self.assertEqual(summary["metrics"]["smThroughput"], "88")
        self.assertEqual(summary["metrics"]["dramThroughput"], "74")

    def test_invalid_metrics_are_unknown(self) -> None:
        summary = parse_file(TESTDATA / "invalid.csv")

        self.assertEqual(summary["boundType"], BOUND_UNKNOWN)
        self.assertEqual(summary["metrics"], {})

    def test_low_pressure_memory_dominant_falls_back_to_memory(self) -> None:
        # Neither absolute threshold fires (sm=4.55, l2=47.14 are both
        # under 70%), as measured for real on a power-capped GPU, but
        # memory is ~10x busier than SM: a real bottleneck, just a mild
        # one by this metric's normalization.
        self.assertEqual(
            classify({"smThroughput": "4.55", "l2Throughput": "47.14"}),
            BOUND_MEMORY,
        )

    def test_low_pressure_compute_dominant_falls_back_to_compute(self) -> None:
        self.assertEqual(
            classify({"smThroughput": "45", "l2Throughput": "10"}),
            BOUND_COMPUTE,
        )

    def test_low_pressure_balanced_stays_unknown(self) -> None:
        # Neither threshold fires and neither resource dominates the
        # other by DOMINANCE_RATIO: genuinely ambiguous, stays unknown.
        self.assertEqual(
            classify({"smThroughput": "30", "l2Throughput": "25"}),
            BOUND_UNKNOWN,
        )

    def test_incomplete_input_does_not_crash(self) -> None:
        with tempfile.TemporaryDirectory() as tmpdir:
            path = Path(tmpdir) / "incomplete.csv"
            path.write_text("Metric Name,Metric Value\nSM Throughput,\n", encoding="utf-8")

            summary = parse_file(path)

        self.assertEqual(summary["boundType"], BOUND_UNKNOWN)
        self.assertEqual(summary["metrics"], {})

    def test_cli_writes_summary_json(self) -> None:
        with tempfile.TemporaryDirectory() as tmpdir:
            output = Path(tmpdir) / "summary.json"
            subprocess.run(
                [
                    sys.executable,
                    str(SCRIPT),
                    "--input",
                    str(TESTDATA / "compute_bound.csv"),
                    "--output",
                    str(output),
                ],
                check=True,
            )

            summary = json.loads(output.read_text(encoding="utf-8"))

        self.assertEqual(summary["boundType"], BOUND_COMPUTE)
        self.assertIn("metrics", summary)
        self.assertEqual(summary["metrics"]["smThroughput"], "82.4")


if __name__ == "__main__":
    unittest.main()
