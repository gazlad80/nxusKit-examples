#!/usr/bin/env python3
"""Example: BN Structure Learning — nxuskit

## nxusKit Features Demonstrated
- BnNetwork lifecycle (create, search structure, learn parameters, infer, close)
- Hill-Climb + BDeu structure learning algorithm
- K2 + BDeu structure learning algorithm
- MLE parameter learning with Laplace smoothing
- Variable Elimination inference on learned models
- Structure comparison between algorithms

## Interactive Modes
- `--verbose` or `-v`: Show detailed structure and parameter info
- `--step` or `-s`: Pause at each learning step with explanations

## Scenarios
Reads CSV data from ../scenarios/<name>/data.csv.
Available scenarios: golf

Usage:
    python main.py --scenario golf
    python main.py --scenario golf --verbose
"""

from __future__ import annotations

import argparse
import os
import sys
from pathlib import Path

# Add shared python module to path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), "../../../shared/python"))

from interactive import InteractiveConfig, StepAction
from nxuskit.bn import BnEvidence, BnNetwork

# -- Constants ---------------------------------------------------------------

SCENARIOS_DIR = Path(__file__).resolve().parent.parent / "scenarios"
SEPARATOR = "=" * 60
THIN_SEP = "-" * 40


def main() -> int:
    config = InteractiveConfig.from_args()

    parser = argparse.ArgumentParser(description="BN Structure Learning Example")
    parser.add_argument(
        "--scenario", "-S", default="golf", help="Scenario name (default: golf)"
    )
    filtered_args = [
        a for a in sys.argv[1:] if a not in ("--verbose", "-v", "--step", "-s")
    ]
    args = parser.parse_args(filtered_args)

    scenario_dir = SCENARIOS_DIR / args.scenario
    csv_path = scenario_dir / "data.csv"

    if not csv_path.exists():
        print(f"Error: Cannot find {csv_path}")
        print(f"Available scenarios: {list_scenarios()}")
        return 1

    print(SEPARATOR)
    print(f"  BN Structure Learning: {args.scenario}")
    print(SEPARATOR)
    print()

    # --- Step 1: Load and inspect CSV data ---
    if (
        config.step_pause(
            "Loading CSV data...",
            [
                "CSV columns become BN variables",
                "Row count affects structure learning quality",
                "Each row is an independent observation",
            ],
        )
        == StepAction.QUIT
    ):
        return 0

    print("--- Step 1: Load CSV Data ---")
    print(f"  File: {csv_path}")
    row_count, columns = inspect_csv(csv_path)
    print(f"  Rows: {row_count}")
    print(f"  Variables ({len(columns)}): {', '.join(columns)}")
    print()

    # --- Step 2: Hill-Climb + BDeu Structure Learning ---
    if (
        config.step_pause(
            "Running Hill-Climb + BDeu structure learning...",
            [
                "Hill-Climb: greedy search — add/remove/reverse edges",
                "BDeu scoring: Bayesian Dirichlet — balances fit with complexity",
                "Result: set of directed edges representing relationships",
            ],
        )
        == StepAction.QUIT
    ):
        return 0

    print("--- Step 2: Hill-Climb + BDeu Structure Learning ---")

    # nxusKit: BnNetwork.create() + search_structure()
    hc_net = BnNetwork.create()
    hc_result = hc_net.search_structure(
        str(csv_path), algorithm="hill_climb", scoring="bdeu", max_parents=3
    )

    hc_edges = hc_result.get("edges", [])
    print("  Algorithm: Hill-Climb + BDeu")
    print(f"  Edges discovered: {len(hc_edges)}")
    print_edges(hc_edges, config)
    print()

    # --- Step 3: K2 + BDeu Structure Learning ---
    if (
        config.step_pause(
            "Running K2 + BDeu structure learning...",
            [
                "K2: requires variable ordering — explores parents in that order",
                "BDeu scoring: Bayesian Dirichlet equivalent uniform prior",
                "Often discovers different structure than Hill-Climb",
            ],
        )
        == StepAction.QUIT
    ):
        return 0

    print("--- Step 3: K2 + BDeu Structure Learning ---")

    k2_net = BnNetwork.create()
    try:
        k2_result = k2_net.search_structure(
            str(csv_path),
            algorithm="k2",
            scoring="bdeu",
            max_parents=3,
            ordering=columns,
        )
        k2_edges = k2_result.get("edges", [])
        print("  Algorithm: K2 + BDeu")
        print(f"  Edges discovered: {len(k2_edges)}")
        print_edges(k2_edges, config)
    except RuntimeError as e:
        print(f"  Warning: K2 structure learning failed: {e}")
        k2_edges = []
    print()

    # --- Step 4: MLE Parameter Learning ---
    if (
        config.step_pause(
            "Learning parameters with MLE...",
            [
                "MLE: Maximum Likelihood Estimation from the data",
                "Laplace smoothing (pseudocount=1.0) prevents zero probabilities",
                "Parameters are conditional probability tables (CPTs)",
            ],
        )
        == StepAction.QUIT
    ):
        return 0

    print("--- Step 4: MLE Parameter Learning ---")
    print("  Structure: Hill-Climb (from Step 2)")

    # nxusKit: learn_mle() populates CPTs from data
    hc_net.learn_mle(str(csv_path), pseudocount=1.0)

    print("  Parameters learned with Laplace smoothing (pseudocount=1.0)")

    if config.verbose:
        variables = hc_net.variables
        print(f"  Variables: {variables}")
        for var in variables:
            states = hc_net.variable_states(var)
            print(f"    {var}: {states}")
    print()

    # --- Step 5: Inference on Learned Model ---
    if (
        config.step_pause(
            "Running inference on learned model...",
            [
                "Variable Elimination: exact inference algorithm",
                "Query: What affects green_speed given weather observations?",
                "Evidence: set observed variables, query unobserved ones",
            ],
        )
        == StepAction.QUIT
    ):
        return 0

    print("--- Step 5: Inference on Learned Model ---")

    try:
        variables = hc_net.variables
    except RuntimeError:
        variables = []

    # Find a good query — use the last variable as target, first as evidence
    if len(variables) >= 2:
        evidence_var = variables[0]
        evidence_states = hc_net.variable_states(evidence_var)
        query_var = variables[-1]

        if evidence_states:
            evidence_val = evidence_states[0]
            print(f"  Evidence: {evidence_var} = {evidence_val}")
            print(f"  Query: P({query_var} | {evidence_var}={evidence_val})")

            # nxusKit: BnEvidence + infer()
            ev = BnEvidence()
            ev.set_discrete(hc_net, evidence_var, evidence_val)
            result = hc_net.infer(ev, algorithm="ve")

            marginal = result.marginal(query_var)
            print(f"  P({query_var}):")
            for state, prob in sorted(marginal.items(), key=lambda x: -x[1]):
                bar = "█" * int(prob * 30)
                print(f"    {state:>15}: {prob:.3f} {bar}")
    print()

    # --- Step 6: Structure Comparison ---
    print("--- Step 6: Structure Comparison ---")
    print(f"  Hill-Climb + BDeu: {len(hc_edges)} edges")
    print(f"  K2 + BDeu:        {len(k2_edges)} edges")

    # Find edges unique to each
    def edge_tuple(e):
        if isinstance(e, list):
            return tuple(e)
        return (e.get("from", ""), e.get("to", ""))

    hc_set = {edge_tuple(e) for e in hc_edges}
    k2_set = {edge_tuple(e) for e in k2_edges}
    shared = hc_set & k2_set
    hc_only = hc_set - k2_set
    k2_only = k2_set - hc_set

    print(f"  Shared edges: {len(shared)}")
    if hc_only:
        print(f"  Hill-Climb only: {sorted(hc_only)}")
    if k2_only:
        print(f"  K2 only: {sorted(k2_only)}")
    print()

    # Cleanup
    hc_net.close()
    k2_net.close()

    print("Done.")
    return 0


def inspect_csv(path: Path) -> tuple[int, list[str]]:
    """Read CSV header and count rows."""
    with open(path) as f:
        header = f.readline().strip()
        columns = [c.strip() for c in header.split(",")]
        row_count = sum(1 for _ in f)
    return row_count, columns


def print_edges(edges: list, config: InteractiveConfig) -> None:
    """Print discovered edges (format: [from, to] pairs or {from, to} dicts)."""
    for edge in edges:
        if isinstance(edge, list) and len(edge) == 2:
            src, dst = edge
        elif isinstance(edge, dict):
            src = edge.get("from", "?")
            dst = edge.get("to", "?")
        else:
            src, dst = "?", "?"
        print(f"    {src} → {dst}")

    if config.verbose and edges:
        config.print_response(200, 0, {"edges": edges})


def list_scenarios() -> list[str]:
    """List available scenario directories."""
    if not SCENARIOS_DIR.exists():
        return []
    return sorted(
        d.name
        for d in SCENARIOS_DIR.iterdir()
        if d.is_dir() and (d / "data.csv").exists()
    )


if __name__ == "__main__":
    sys.exit(main())
