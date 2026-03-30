#!/usr/bin/env python3
"""Example: Bayesian Network Inference - nxuskit

## nxusKit Features Demonstrated
- BnNetwork lifecycle (load BIF model, close)
- BnEvidence creation and discrete observation setting
- Exact inference: Variable Elimination (VE) and Junction Tree (JT)
- Approximate inference: Loopy Belief Propagation (LBP) and Gibbs Sampling
- Algorithm-specific configuration via infer_with_config()
- Streaming Gibbs inference with progressive marginal updates
- BnResult marginal queries and JSON serialization

## Interactive Modes
- `--verbose` or `-v`: Show detailed marginals and network structure
- `--step` or `-s`: Pause at each step with explanations

## Scenarios
Reads model.bif and evidence.json from ../scenarios/<name>/.
Available scenarios: haunted-house, coffee-shop, plant-doctor

Usage:
    python main.py --scenario haunted-house
    python main.py --scenario coffee-shop --verbose
    python main.py --scenario plant-doctor --step
"""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path

from nxuskit.bn import BnEvidence, BnNetwork, BnResult

# -- Constants --------------------------------------------------------

SCENARIOS_DIR = Path(__file__).resolve().parent.parent / "scenarios"
SEPARATOR = "=" * 60
THIN_SEP = "-" * 40

# Algorithms to compare in the main run
ALGORITHMS = [
    ("ve", "Variable Elimination (exact)"),
    ("jt", "Junction Tree (exact)"),
    ("lbp", "Loopy Belief Propagation (approximate)"),
    ("gibbs", "Gibbs Sampling (approximate)"),
]

# Gibbs sampling parameters
GIBBS_NUM_SAMPLES = 10000
GIBBS_BURN_IN = 1000
GIBBS_SEED = 42


# -- Helpers ----------------------------------------------------------


def list_available_scenarios() -> list[str]:
    """Return sorted list of scenario directory names."""
    if not SCENARIOS_DIR.is_dir():
        return []
    return sorted(
        d.name
        for d in SCENARIOS_DIR.iterdir()
        if d.is_dir() and (d / "model.bif").exists()
    )


def load_evidence(scenario_dir: Path) -> dict[str, str]:
    """Load evidence.json from the scenario directory."""
    path = scenario_dir / "evidence.json"
    if not path.exists():
        return {}
    with open(path) as f:
        return json.load(f)


def print_marginals(
    result: BnResult,
    network: BnNetwork,
    variables: list[str],
    verbose: bool = False,
) -> None:
    """Print posterior marginals for all non-evidence variables."""
    for var in sorted(variables):
        marginal = result.marginal(var)
        # Find the most probable state
        best_state = max(marginal, key=marginal.get)
        best_prob = marginal[best_state]
        if verbose:
            # Show full distribution
            dist_parts = [f"{s}={p:.4f}" for s, p in sorted(marginal.items())]
            print(f"    {var}: {', '.join(dist_parts)}")
        else:
            print(f"    {var}: {best_state} ({best_prob:.4f})")


def print_comparison_table(
    algo_results: list[tuple[str, str, dict[str, dict[str, float]]]],
    query_variables: list[str],
) -> None:
    """Print a comparison table of marginals across algorithms."""
    if not algo_results or not query_variables:
        return

    print()
    print(THIN_SEP)
    print("  Algorithm Comparison")
    print(THIN_SEP)
    print()

    # Header
    algo_names = [name for _, name, _ in algo_results]
    col_width = max(len(n) for n in algo_names) + 2
    var_width = max(len(v) for v in query_variables) + 2
    header = f"  {'Variable':<{var_width}}"
    for _, name, _ in algo_results:
        header += f"  {name:>{col_width}}"
    print(header)
    print("  " + "-" * (var_width + (col_width + 2) * len(algo_results)))

    # Rows: show the most probable state and its probability for each algo
    for var in sorted(query_variables):
        row = f"  {var:<{var_width}}"
        for _, _, marginals_dict in algo_results:
            marginal = marginals_dict.get(var, {})
            if marginal:
                best_state = max(marginal, key=marginal.get)
                best_prob = marginal[best_state]
                row += f"  {best_state}={best_prob:.4f}".rjust(col_width + 2)
            else:
                row += f"  {'N/A':>{col_width}}"
        print(row)
    print()


def step_pause(step: bool, message: str) -> None:
    """If step mode is active, print the message and wait for Enter."""
    if step:
        print(f"  {message}")
        input("  [Press Enter to continue...]")
        print()


# -- Main Logic -------------------------------------------------------


def run(scenario: str, verbose: bool = False, step: bool = False) -> int:
    """Run the Bayesian inference example for the given scenario."""
    scenario_dir = SCENARIOS_DIR / scenario
    model_path = scenario_dir / "model.bif"
    evidence_data = load_evidence(scenario_dir)

    # -- Print scenario summary ----------------------------------------
    print(SEPARATOR)
    print(f"  Bayesian Inference Example: {scenario}")
    print(SEPARATOR)
    print()

    step_pause(step, "Loading the Bayesian Network model from BIF file...")

    # -- Load network --------------------------------------------------
    with BnNetwork.load(str(model_path)) as net:
        variables = net.variables
        num_vars = net.num_variables

        print(f"  Model:     {model_path.name}")
        print(f"  Variables: {num_vars}")
        print(f"  Evidence:  {len(evidence_data)} observations")
        print()

        if verbose:
            print("  Network variables:")
            for var in sorted(variables):
                states = net.variable_states(var)
                print(f"    {var}: {states}")
            print()

        # -- Evidence summary ------------------------------------------
        if evidence_data:
            print("  Observed evidence:")
            for var, state in sorted(evidence_data.items()):
                print(f"    {var} = {state}")
            print()

        # Determine query variables (non-evidence variables)
        evidence_vars = set(evidence_data.keys())
        query_variables = [v for v in variables if v not in evidence_vars]

        if verbose:
            print(
                f"  Query variables ({len(query_variables)}): "
                f"{', '.join(sorted(query_variables))}"
            )
            print()

        step_pause(step, "Setting evidence observations on the network...")

        # -- Create evidence -------------------------------------------
        with BnEvidence() as ev:
            for var, state in evidence_data.items():
                ev.set_discrete(net, var, state)

            if verbose:
                print(f"  Evidence set: {len(evidence_data)} discrete observations")
                print()

            algo_results: list[tuple[str, str, dict[str, dict[str, float]]]] = []
            results_summary: list[tuple[str, str]] = []

            # -- Step 1: Variable Elimination --------------------------
            print(THIN_SEP)
            print("  Step 1: Variable Elimination (exact inference)")
            print(THIN_SEP)
            step_pause(
                step,
                "VE systematically sums out variables to compute exact posteriors.",
            )

            with net.infer(ev, "ve") as result:
                print()
                print_marginals(result, net, query_variables, verbose)
                marginals = {v: result.marginal(v) for v in query_variables}
                algo_results.append(("ve", "VE", marginals))
                results_summary.append(("VE (exact)", "ok"))
            print()

            # -- Step 2: Junction Tree ---------------------------------
            print(THIN_SEP)
            print("  Step 2: Junction Tree (exact inference)")
            print(THIN_SEP)
            step_pause(
                step, "JT builds a clique tree for efficient exact message passing."
            )

            with net.infer(ev, "jt") as result:
                print()
                print_marginals(result, net, query_variables, verbose)
                marginals = {v: result.marginal(v) for v in query_variables}
                algo_results.append(("jt", "JT", marginals))
                results_summary.append(("JT (exact)", "ok"))
            print()

            # -- Step 3: Loopy Belief Propagation ----------------------
            print(THIN_SEP)
            print("  Step 3: Loopy Belief Propagation (approximate)")
            print(THIN_SEP)
            step_pause(
                step, "LBP iterates messages on a factor graph until convergence."
            )

            try:
                with net.infer_with_config(
                    ev, "lbp", {"max_iterations": 100}
                ) as result:
                    print()
                    print_marginals(result, net, query_variables, verbose)
                    marginals = {v: result.marginal(v) for v in query_variables}
                    algo_results.append(("lbp", "LBP", marginals))
                    results_summary.append(("LBP (approx)", "ok"))
            except Exception as e:
                print(f"    LBP inference failed: {e}")
                results_summary.append(("LBP (approx)", f"error: {e}"))
            print()

            # -- Step 4: Gibbs Sampling --------------------------------
            print(THIN_SEP)
            print("  Step 4: Gibbs Sampling (approximate, MCMC)")
            print(THIN_SEP)
            step_pause(
                step,
                f"Gibbs draws {GIBBS_NUM_SAMPLES} samples with "
                f"burn-in={GIBBS_BURN_IN}, seed={GIBBS_SEED}.",
            )

            try:
                with net.infer(
                    ev,
                    "gibbs",
                    num_samples=GIBBS_NUM_SAMPLES,
                    burn_in=GIBBS_BURN_IN,
                    seed=GIBBS_SEED,
                ) as result:
                    print()
                    print_marginals(result, net, query_variables, verbose)
                    marginals = {v: result.marginal(v) for v in query_variables}
                    algo_results.append(("gibbs", "Gibbs", marginals))
                    results_summary.append(("Gibbs (MCMC)", "ok"))

                    if verbose:
                        print()
                        print("  Full result JSON:")
                        result_json = json.loads(result.to_json())
                        json_str = json.dumps(result_json, indent=2, sort_keys=True)
                        print(f"    {json_str[:500]}...")
            except Exception as e:
                print(f"    Gibbs inference failed: {e}")
                results_summary.append(("Gibbs (MCMC)", f"error: {e}"))
            print()

            # -- Streaming demonstration -------------------------------
            print(THIN_SEP)
            print("  Bonus: Streaming Gibbs inference")
            print(THIN_SEP)
            step_pause(
                step,
                "Streaming shows progressive convergence of Gibbs sampling.",
            )

            try:
                chunk_count = 0
                for chunk in net.infer_stream(
                    ev,
                    num_samples=GIBBS_NUM_SAMPLES,
                    burn_in=GIBBS_BURN_IN,
                    seed=GIBBS_SEED,
                    chunk_size=1000,
                ):
                    chunk_count += 1
                    if chunk.is_final:
                        print(
                            f"    Stream complete: {chunk.iteration}/{chunk.total} "
                            f"iterations"
                        )
                    elif verbose:
                        print(f"    Chunk {chunk.iteration}/{chunk.total}")
                if verbose:
                    print(f"    Total stream chunks: {chunk_count}")
                results_summary.append(("Streaming", "ok"))
            except Exception as e:
                print(f"    Streaming not available: {e}")
                results_summary.append(("Streaming", "unavailable"))
            print()

        # -- Algorithm comparison table --------------------------------
        print_comparison_table(algo_results, query_variables)

        # -- Summary ---------------------------------------------------
        print(SEPARATOR)
        print("  Summary")
        print(SEPARATOR)
        print()
        for label, status in results_summary:
            print(f"  {label:30s} {status}")
        print()
        print("  Network closed. All resources released.")
        print(SEPARATOR)

    return 0


def main() -> int:
    """Entry point: parse arguments and run the Bayesian inference example."""
    parser = argparse.ArgumentParser(
        description="Bayesian Network Inference Example - nxuskit",
    )
    parser.add_argument(
        "--scenario",
        default="haunted-house",
        help="Scenario name (directory under ../scenarios/). Default: haunted-house",
    )
    parser.add_argument(
        "--verbose",
        "-v",
        action="store_true",
        help="Show detailed marginals and network structure",
    )
    parser.add_argument(
        "--step",
        "-s",
        action="store_true",
        help="Pause at each step with explanations",
    )
    args = parser.parse_args()

    # FR-031: validate scenario name and list available on error
    available = list_available_scenarios()
    if args.scenario not in available:
        print(f"Error: unknown scenario '{args.scenario}'")
        if available:
            print(f"Available scenarios: {', '.join(available)}")
        else:
            print(f"No scenarios found in {SCENARIOS_DIR}")
        return 1

    return run(args.scenario, verbose=args.verbose, step=args.step)


if __name__ == "__main__":
    sys.exit(main())
