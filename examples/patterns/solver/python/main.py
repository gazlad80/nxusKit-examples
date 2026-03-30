#!/usr/bin/env python3
"""Example: Solver - nxuskit

## nxusKit Features Demonstrated
- SolverSession lifecycle (create, build model, solve, close)
- Variable, constraint, and objective definitions from JSON scenarios
- Incremental solving: satisfaction -> optimization -> multi-objective
- Soft constraints with configurable weights
- What-if analysis via push/pop scoping
- Streaming solver progress via solve_stream()

## Interactive Modes
- `--verbose` or `-v`: Show detailed solver state at each step
- `--step` or `-s`: Pause at each step with explanations

## Scenarios
Reads problem definitions from ../scenarios/<name>/problem.json.
Available scenarios: theme-park, space-colony, fantasy-draft

Usage:
    python main.py --scenario theme-park
    python main.py --scenario space-colony --verbose
    python main.py --scenario fantasy-draft --step
"""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path

from nxuskit.solver import SolverSession
from nxuskit.solver_types import (
    ConstraintDef,
    ConstraintType,
    DomainDef,
    MultiObjectiveMode,
    ObjectiveDef,
    ObjectiveDirection,
    SolverConfig,
    SolveResult,
    VariableDef,
    VariableType,
)

# ── Constants ───────────────────────────────────────────────

SCENARIOS_DIR = Path(__file__).resolve().parent.parent / "scenarios"
SEPARATOR = "=" * 60
THIN_SEP = "-" * 40


# ── Helpers ─────────────────────────────────────────────────


def list_available_scenarios() -> list[str]:
    """Return sorted list of scenario directory names."""
    if not SCENARIOS_DIR.is_dir():
        return []
    return sorted(
        d.name
        for d in SCENARIOS_DIR.iterdir()
        if d.is_dir() and (d / "problem.json").exists()
    )


def load_problem(scenario: str) -> dict:
    """Load and return the problem.json for the given scenario name."""
    path = SCENARIOS_DIR / scenario / "problem.json"
    if not path.exists():
        available = list_available_scenarios()
        print(f"Error: scenario '{scenario}' not found at {path}")
        if available:
            print(f"Available scenarios: {', '.join(available)}")
        else:
            print(f"No scenarios found in {SCENARIOS_DIR}")
        sys.exit(1)
    with open(path) as f:
        return json.load(f)


def parse_variable(data: dict) -> VariableDef:
    """Parse a variable definition from problem JSON."""
    domain = None
    if "domain" in data:
        domain = DomainDef.from_dict(data["domain"])
    return VariableDef(
        name=data["name"],
        var_type=VariableType(data["var_type"]),
        domain=domain,
        label=data.get("label", ""),
    )


def parse_constraint(data: dict, weight: float | None = None) -> ConstraintDef:
    """Parse a constraint definition from problem JSON."""
    return ConstraintDef(
        name=data.get("name", ""),
        constraint_type=ConstraintType(data["constraint_type"]),
        variables=data.get("variables", []),
        parameters=data.get("parameters") or {},
        weight=weight if weight is not None else data.get("weight"),
        label=data.get("label", ""),
    )


def parse_objective(data: dict) -> ObjectiveDef:
    """Parse an objective definition from problem JSON."""
    return ObjectiveDef(
        name=data["name"],
        direction=ObjectiveDirection(data["direction"]),
        expression=data.get("expression", ""),
        variable=data.get("variable", ""),
        weight=data.get("weight"),
        label=data.get("label", ""),
        priority=data.get("priority"),
    )


def print_result(result: SolveResult, verbose: bool = False) -> None:
    """Print solve result in a readable format."""
    print(f"  Status: {result.status.value}")
    if result.assignments:
        print("  Assignments:")
        for name, val in sorted(result.assignments.items()):
            print(f"    {name} = {val.value} ({val.type})")
    if result.objective_value is not None:
        print(f"  Objective value: {result.objective_value}")
    if result.objective_values:
        print("  Objective values:")
        for name, val in sorted(result.objective_values.items()):
            print(f"    {name} = {val}")
    if result.violated_soft_constraints:
        print(f"  Violated soft constraints: {result.violated_soft_constraints}")
    if verbose and result.stats:
        print(
            f"  Stats: {result.stats.solve_time_ms}ms, "
            f"{result.stats.num_variables} vars, "
            f"{result.stats.num_constraints} constraints"
        )


# ── Main ────────────────────────────────────────────────────


def run(scenario: str, verbose: bool = False, step: bool = False) -> int:
    """Run the constraint solver example for the given scenario."""
    problem = load_problem(scenario)

    # ── Print problem summary ───────────────────────────────
    print(SEPARATOR)
    print(f"  Constraint Solver Example: {problem['name']}")
    print(SEPARATOR)
    print()
    print(f"  {problem['description']}")
    print()

    variables = [parse_variable(v) for v in problem["variables"]]
    constraints = [parse_constraint(c) for c in problem["constraints"]]
    objectives = [parse_objective(o) for o in problem.get("objectives", [])]
    soft_constraints_data = problem.get("soft_constraints", [])
    what_if_scenarios = problem.get("what_if_scenarios", [])

    print(f"  Variables:        {len(variables)}")
    print(f"  Constraints:      {len(constraints)}")
    print(f"  Objectives:       {len(objectives)}")
    print(f"  Soft constraints: {len(soft_constraints_data)}")
    print(f"  What-if scenarios: {len(what_if_scenarios)}")
    print()

    if verbose:
        print("  Variables:")
        for v in variables:
            domain_str = ""
            if v.domain:
                parts = []
                if v.domain.min is not None:
                    parts.append(f"min={v.domain.min}")
                if v.domain.max is not None:
                    parts.append(f"max={v.domain.max}")
                domain_str = f" [{', '.join(parts)}]" if parts else ""
            print(f"    {v.name}: {v.var_type.value}{domain_str}")
        print()

    if step:
        input("  [Press Enter to continue...]")
        print()

    # ── Create solver session ───────────────────────────────
    print(THIN_SEP)
    print("  Creating solver session...")
    print(THIN_SEP)

    results_summary: list[tuple[str, str]] = []

    with SolverSession(config=SolverConfig(timeout_ms=30000)) as session:
        # ── Add variables ───────────────────────────────────
        session.add_variables(variables)
        if verbose:
            print(f"  Added {len(variables)} variables")
        print()

        # ── Step 1: Satisfaction ────────────────────────────
        print(THIN_SEP)
        print("  Step 1: Satisfaction (hard constraints only)")
        print(THIN_SEP)
        if step:
            print(
                "  Adding hard constraints and checking if a feasible solution exists."
            )
            input("  [Press Enter to continue...]")

        session.add_constraints(constraints)
        if verbose:
            print(f"  Added {len(constraints)} hard constraints:")
            for c in constraints:
                print(f"    {c.name}: {c.label}")

        result = session.solve()
        print_result(result, verbose)
        results_summary.append(("Satisfaction", result.status.value))
        print()

        # ── Step 2: Optimization (single objective) ────────
        if objectives:
            print(THIN_SEP)
            print("  Step 2: Optimization (single objective)")
            print(THIN_SEP)
            if step:
                print(
                    f"  Setting objective: {objectives[0].name} "
                    f"({objectives[0].direction.value})"
                )
                input("  [Press Enter to continue...]")

            session.set_objective(objectives[0])
            if verbose:
                print(
                    f"  Objective: {objectives[0].name} - "
                    f"{objectives[0].direction.value} "
                    f"{objectives[0].expression or objectives[0].variable}"
                )

            result = session.solve()
            print_result(result, verbose)
            results_summary.append(("Optimization", result.status.value))
            print()

        # ── Step 3: Multi-objective ────────────────────────
        if len(objectives) > 1:
            print(THIN_SEP)
            print("  Step 3: Multi-objective (weighted mode)")
            print(THIN_SEP)
            if step:
                print(f"  Adding {len(objectives)} objectives in weighted mode.")
                input("  [Press Enter to continue...]")

            # Add all objectives for multi-objective solve
            for obj in objectives:
                session.add_objective(obj)
                if verbose:
                    w = obj.weight if obj.weight is not None else 1.0
                    print(
                        f"  Added objective: {obj.name} "
                        f"({obj.direction.value}, weight={w})"
                    )

            result = session.solve(
                config=SolverConfig(
                    multi_objective_mode=MultiObjectiveMode.WEIGHTED,
                )
            )
            print_result(result, verbose)
            results_summary.append(("Multi-objective", result.status.value))
            print()
        else:
            results_summary.append(("Multi-objective", "skipped (single objective)"))

        # ── Step 4: Soft constraints ───────────────────────
        if soft_constraints_data:
            print(THIN_SEP)
            print("  Step 4: Soft constraints")
            print(THIN_SEP)
            if step:
                print(
                    f"  Adding {len(soft_constraints_data)} soft constraints "
                    "with weights."
                )
                input("  [Press Enter to continue...]")

            soft_constraints = [
                parse_constraint(sc, weight=sc.get("weight", 1.0))
                for sc in soft_constraints_data
            ]
            session.add_constraints(soft_constraints)
            if verbose:
                for sc in soft_constraints:
                    print(f"  Soft: {sc.name} (weight={sc.weight})")

            result = session.solve()
            print_result(result, verbose)
            results_summary.append(("Soft constraints", result.status.value))
            print()
        else:
            results_summary.append(("Soft constraints", "skipped (none defined)"))

        # ── Step 5: What-if analysis ───────────────────────
        for i, scenario_data in enumerate(what_if_scenarios):
            print(THIN_SEP)
            print(f"  Step 5{chr(ord('a') + i)}: What-if - {scenario_data['name']}")
            print(THIN_SEP)
            if step:
                print(f"  {scenario_data['description']}")
                input("  [Press Enter to continue...]")

            if verbose:
                print(f"  Description: {scenario_data['description']}")

            # Push scope for what-if exploration
            session.push()
            if verbose:
                print("  Pushed solver scope")

            additional = [
                parse_constraint(c)
                for c in scenario_data.get("additional_constraints", [])
            ]
            session.add_constraints(additional)
            if verbose:
                for c in additional:
                    print(f"  Added temporary constraint: {c.name} - {c.label}")

            result = session.solve()
            print_result(result, verbose)
            results_summary.append(
                (f"What-if: {scenario_data['name']}", result.status.value)
            )

            # Pop scope to restore previous state
            session.pop()
            if verbose:
                print("  Popped solver scope (restored previous state)")
            print()

        # ── Streaming demonstration ────────────────────────
        print(THIN_SEP)
        print("  Bonus: Streaming solve progress")
        print(THIN_SEP)
        if step:
            print("  Re-solving with streaming to observe optimization progress.")
            input("  [Press Enter to continue...]")

        try:
            chunk_count = 0
            for chunk in session.solve_stream():
                chunk_count += 1
                if chunk.is_final:
                    print(
                        f"  Stream complete: status={chunk.status}, "
                        f"objective={chunk.objective_value}"
                    )
                elif verbose:
                    print(
                        f"  Iteration {chunk.iteration}: "
                        f"objective={chunk.objective_value}, "
                        f"elapsed={chunk.elapsed_ms}ms"
                    )
            if verbose:
                print(f"  Total stream chunks: {chunk_count}")
        except Exception as e:
            print(f"  Streaming not available: {e}")
            results_summary.append(("Streaming", "unavailable"))
        else:
            results_summary.append(("Streaming", "ok"))
        print()

    # ── Summary ─────────────────────────────────────────────
    print(SEPARATOR)
    print("  Summary")
    print(SEPARATOR)
    for label, status in results_summary:
        print(f"  {label:30s} {status}")
    print()
    print("  Session closed. All resources released.")
    print(SEPARATOR)

    return 0


def main() -> int:
    """Entry point: parse arguments and run the solver example."""
    parser = argparse.ArgumentParser(
        description="Constraint Solver Example - nxuskit",
    )
    parser.add_argument(
        "--scenario",
        default="theme-park",
        help="Scenario name (directory under ../scenarios/). Default: theme-park",
    )
    parser.add_argument(
        "--verbose",
        "-v",
        action="store_true",
        help="Show detailed solver state at each step",
    )
    parser.add_argument(
        "--step",
        "-s",
        action="store_true",
        help="Pause at each step with explanations",
    )
    args = parser.parse_args()

    # FR-031: validate scenario name
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
