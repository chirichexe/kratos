#!/usr/bin/env python3
"""Parse Nsight Compute exports into a small workload profile summary."""

from __future__ import annotations

import argparse
import csv
import json
import re
from pathlib import Path
from typing import Any


BOUND_COMPUTE = "compute-bound"
BOUND_MEMORY = "memory-bound"
BOUND_MIXED = "mixed"
BOUND_UNKNOWN = "unknown"

METRIC_ALIASES = {
    "smThroughput": {
        "smthroughput",
        "sm throughput",
        "sm throughput %",
        "sm__throughput.avg.pct_of_peak_sustained_elapsed",
    },
    "dramThroughput": {
        "dramthroughput",
        "dram throughput",
        "dram throughput %",
        "dram__throughput.avg.pct_of_peak_sustained_elapsed",
    },
    "l2Throughput": {
        "l2throughput",
        "l2 throughput",
        "l2 throughput %",
        "lts__throughput.avg.pct_of_peak_sustained_elapsed",
        "lts__t_sectors.avg.pct_of_peak_sustained_elapsed",
    },
    "achievedOccupancy": {
        "achievedoccupancy",
        "achieved occupancy",
        "sm__warps_active.avg.pct_of_peak_sustained_active",
    },
    "memoryStallPct": {
        "memorystallpct",
        "memory stall pct",
        "memory stall %",
        "memory stall percentage",
        "smsp__warp_issue_stalled_long_scoreboard_per_warp_active.pct",
    },
}

ALIAS_TO_METRIC = {
    alias: metric for metric, aliases in METRIC_ALIASES.items() for alias in aliases
}


def parse_file(path: Path) -> dict[str, Any]:
    text = path.read_text(encoding="utf-8")
    metrics = parse_json_metrics(text)
    if not metrics:
        metrics = parse_csv_metrics(text)
    if not metrics:
        metrics = parse_text_metrics(text)

    return {
        "boundType": classify(metrics),
        "metrics": {key: value for key, value in metrics.items() if value is not None},
    }


def parse_json_metrics(text: str) -> dict[str, str]:
    try:
        document = json.loads(text)
    except json.JSONDecodeError:
        return {}

    metrics: dict[str, str] = {}
    for key, value in walk_json(document):
        add_metric(metrics, key, value)
    return metrics


def walk_json(value: Any) -> list[tuple[str, Any]]:
    pairs: list[tuple[str, Any]] = []
    if isinstance(value, dict):
        metric_name = first_present(value, ("metric", "metricName", "name", "Metric Name"))
        metric_value = first_present(value, ("value", "metricValue", "Metric Value", "avg"))
        if metric_name is not None and metric_value is not None:
            pairs.append((str(metric_name), metric_value))

        for key, child in value.items():
            if not isinstance(child, (dict, list)):
                pairs.append((str(key), child))
            else:
                pairs.extend(walk_json(child))
    elif isinstance(value, list):
        for child in value:
            pairs.extend(walk_json(child))
    return pairs


def first_present(document: dict[str, Any], keys: tuple[str, ...]) -> Any:
    for key in keys:
        if key in document:
            return document[key]
    return None


def parse_csv_metrics(text: str) -> dict[str, str]:
    metrics: dict[str, str] = {}
    rows = list(csv.reader(text.splitlines()))
    if not rows:
        return metrics

    header_row_idx: int | None = None
    metric_idx: int | None = None
    value_idx: int | None = None
    for idx, row in enumerate(rows):
        header = [normalize_header(cell) for cell in row]
        found_metric_idx = find_first_index(header, ("metric name", "metric", "name"))
        found_value_idx = find_first_index(header, ("metric value", "value", "avg"))
        if found_metric_idx is not None and found_value_idx is not None:
            header_row_idx = idx
            metric_idx = found_metric_idx
            value_idx = found_value_idx
            break

    data_rows = rows[header_row_idx+1:] if header_row_idx is not None else rows
    for row in data_rows:
        if metric_idx is not None and value_idx is not None:
            if len(row) <= max(metric_idx, value_idx):
                continue
            add_metric(metrics, row[metric_idx], row[value_idx])
            continue

        if len(row) >= 2:
            add_metric(metrics, row[0], row[1])

    return metrics


def parse_text_metrics(text: str) -> dict[str, str]:
    metrics: dict[str, str] = {}
    for line in text.splitlines():
        parts = line.strip().split()
        if len(parts) < 2:
            continue

        canonical = canonical_metric_name(parts[0])
        if canonical is None or canonical in metrics:
            continue

        matches = re.findall(r"[-+]?\d+(?:,\d{3})*(?:\.\d+)?", " ".join(parts[1:]))
        if not matches:
            continue
        add_metric(metrics, parts[0], matches[-1])
    return metrics


def normalize_header(value: str) -> str:
    return value.strip().lower()


def find_first_index(values: list[str], candidates: tuple[str, ...]) -> int | None:
    for candidate in candidates:
        if candidate in values:
            return values.index(candidate)
    return None


def add_metric(metrics: dict[str, str], name: str, value: Any) -> None:
    canonical = canonical_metric_name(name)
    if canonical is None or canonical in metrics:
        return

    parsed = parse_number(value)
    if parsed is None:
        return
    metrics[canonical] = format_number(parsed)


def canonical_metric_name(name: str) -> str | None:
    normalized = normalize_metric_name(name)
    if normalized in ALIAS_TO_METRIC:
        return ALIAS_TO_METRIC[normalized]

    compact = normalized.replace(" ", "")
    for alias, metric in ALIAS_TO_METRIC.items():
        if compact == alias.replace(" ", ""):
            return metric
    return None


def normalize_metric_name(name: str) -> str:
    return re.sub(r"\s+", " ", name.strip().lower())


def parse_number(value: Any) -> float | None:
    if isinstance(value, bool):
        return None
    if isinstance(value, (int, float)):
        return float(value)

    match = re.search(r"[-+]?\d+(?:,\d{3})*(?:\.\d+)?", str(value))
    if match is None:
        return None
    try:
        return float(match.group(0).replace(",", ""))
    except ValueError:
        return None


def format_number(value: float) -> str:
    return f"{value:.6f}".rstrip("0").rstrip(".")


def classify(metrics: dict[str, str]) -> str:
    values = {key: parse_number(value) for key, value in metrics.items()}
    sm = values.get("smThroughput")
    dram = values.get("dramThroughput")
    l2 = values.get("l2Throughput")
    memory_stall = values.get("memoryStallPct")

    if sm is None and dram is None and l2 is None and memory_stall is None:
        return BOUND_UNKNOWN

    memory_values = [value for value in (dram, l2) if value is not None]
    memory_peak = max(memory_values) if memory_values else None

    compute_pressure = sm is not None and sm >= 70.0
    throughput_pressure = memory_peak is not None and memory_peak >= 70.0
    stall_pressure = memory_stall is not None and memory_stall >= 30.0
    memory_pressure = throughput_pressure or stall_pressure

    if compute_pressure and memory_pressure:
        return BOUND_MIXED
    if memory_pressure:
        return BOUND_MEMORY
    if compute_pressure:
        return BOUND_COMPUTE
    return BOUND_UNKNOWN


def write_summary(summary: dict[str, Any], path: Path) -> None:
    path.write_text(json.dumps(summary, indent=2, sort_keys=True) + "\n", encoding="utf-8")


def main() -> int:
    parser = argparse.ArgumentParser(description="Parse Nsight Compute output into JSON summary")
    parser.add_argument("--input", required=True, type=Path, help="Path to Nsight Compute CSV or JSON export")
    parser.add_argument("--output", required=True, type=Path, help="Path to write summary JSON")
    args = parser.parse_args()

    summary = parse_file(args.input)
    write_summary(summary, args.output)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
