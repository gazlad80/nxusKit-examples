#!/usr/bin/env python3
# ruff: noqa: E402
"""Solver What-If Example — nxusKit SDK

Demonstrates push/pop scoping, incremental solving, and what-if analysis:
  1. Load a scenario with variables, constraints, and objectives
  2. Solve the base model
  3. For each what-if scenario: push scope, add constraints, solve, compare, pop
  4. Summarize the impact of each temporary change

Usage:
  python3 main.py [--scenario wedding] [--verbose] [--step]
"""

from __future__ import annotations

import argparse
import json
import sys
import time
from dataclasses import dataclass
from pathlib import Path

EXAMPLES_DIR = Path(__file__).resolve().parents[3]
EXAMPLE_DIR = Path(__file__).resolve().parents[1]
SCENARIOS_DIR = EXAMPLE_DIR / "scenarios"
sys.path.insert(0, str(EXAMPLES_DIR / "shared" / "python"))

from interactive import InteractiveConfig, StepAction
from nxuskit.solver import SolverError, SolverLibraryNotFoundError, SolverSession
from nxuskit.solver_types import (
    ConstraintDef,
    ConstraintType,
    DomainDef,
    ObjectiveDef,
    ObjectiveDirection,
    SolveResult,
    SolveStatus,
    SolverConfig,
    SolverExplanation,
    SolverValue,
    VariableDef,
    VariableType,
)


@dataclass
class ScenarioSummary:
    name: str
    status: str
    objective_value: float | None = None


def list_scenarios() -> list[str]:
    if not SCENARIOS_DIR.is_dir():
        return []
    return sorted(
        item.name
        for item in SCENARIOS_DIR.iterdir()
        if item.is_dir() and (item / "problem.json").exists()
    )


def load_problem(scenario: str) -> dict:
    path = SCENARIOS_DIR / scenario / "problem.json"
    if not path.exists():
        available = list_scenarios()
        print(f"Error: scenario '{scenario}' not found at {path}")
        if available:
            print(f"Available scenarios: {', '.join(available)}")
        return {}
    with open(path) as handle:
        return json.load(handle)


def parse_variable(data: dict) -> VariableDef:
    domain = DomainDef.from_dict(data["domain"]) if "domain" in data else None
    return VariableDef(
        name=data["name"],
        var_type=VariableType(data["var_type"]),
        domain=domain,
        label=data.get("label", ""),
    )


def parse_constraint(data: dict) -> ConstraintDef:
    return ConstraintDef(
        name=data.get("name", ""),
        constraint_type=ConstraintType(data["constraint_type"]),
        variables=data.get("variables", []),
        parameters=data.get("parameters") or {},
        weight=data.get("weight"),
        label=data.get("label", ""),
    )


def parse_objective(data: dict) -> ObjectiveDef:
    return ObjectiveDef(
        name=data["name"],
        direction=ObjectiveDirection(data["direction"]),
        expression=data.get("expression", ""),
        variable=data.get("variable", ""),
        weight=data.get("weight"),
        label=data.get("label", ""),
        priority=data.get("priority"),
    )


def value_as_float(value: SolverValue) -> float:
    raw = value.value
    if isinstance(raw, bool):
        return 1.0 if raw else 0.0
    return float(raw)


def format_value(value: SolverValue) -> str:
    raw = value.value
    if isinstance(raw, float) and raw.is_integer():
        return str(int(raw))
    return str(raw)


def extract_assignments(result: SolveResult) -> dict[str, float]:
    return {
        name: value_as_float(value)
        for name, value in sorted(result.assignments.items())
    }


def print_assignments(result: SolveResult, indent: str = "  ") -> None:
    if not result.assignments:
        return
    print("Assignments:")
    for name, value in sorted(result.assignments.items()):
        print(f"{indent}{name} = {format_value(value)}")


def print_stats(result: SolveResult, config: InteractiveConfig) -> None:
    if not config.is_verbose():
        return
    stats = result.stats
    print(
        "[nxusKit] Solve stats: "
        f"{stats.solve_time_ms}ms, "
        f"{stats.num_variables} vars, "
        f"{stats.num_constraints} constraints"
    )


def print_delta(
    base: dict[str, float],
    variant: dict[str, float],
    indent: str = "    ",
) -> None:
    all_names = sorted(set(base) | set(variant))
    any_delta = False
    for name in all_names:
        before = base.get(name, 0.0)
        after = variant.get(name, 0.0)
        diff = after - before
        if abs(diff) <= 0.001:
            continue
        sign = "+" if diff > 0 else ""
        print(f"{indent}{name}: {before:.1f} -> {after:.1f} ({sign}{diff:.1f})")
        any_delta = True
    if not any_delta:
        print(f"{indent}(no changes from base)")


def print_explanation(explanation: SolverExplanation | None) -> None:
    if explanation is None:
        print("  (no explanation available)")
        return
    if explanation.unsat_core_labels:
        print(f"  Unsat core: {', '.join(explanation.unsat_core_labels)}")
    if explanation.binding_constraints:
        print(f"  Binding constraints: {', '.join(explanation.binding_constraints)}")


def solve(session: SolverSession) -> SolveResult:
    return session.solve(SolverConfig(produce_explanation=True))


def main() -> int:
    config = InteractiveConfig.from_args()

    parser = argparse.ArgumentParser(description="Solver what-if analysis")
    parser.add_argument(
        "--scenario",
        "-S",
        default="wedding",
        help="Scenario to run (default: wedding)",
    )
    filtered_args = [
        arg for arg in sys.argv[1:] if arg not in ("--verbose", "-v", "--step", "-s")
    ]
    args = parser.parse_args(filtered_args)

    problem = load_problem(args.scenario)
    if not problem:
        return 1

    variables = [parse_variable(item) for item in problem.get("variables", [])]
    constraints = [parse_constraint(item) for item in problem.get("constraints", [])]
    objectives = [parse_objective(item) for item in problem.get("objectives", [])]
    what_if_scenarios = problem.get("what_if_scenarios", [])

    print("========================================")
    print(f"  Solver What-If: {problem['name']}")
    print("========================================")
    print()
    print(problem["description"])
    print()
    print(f"Variables:          {len(variables)}")
    print(f"Hard constraints:   {len(constraints)}")
    print(f"What-if scenarios:  {len(what_if_scenarios)}")
    print()

    if (
        config.step_pause(
            "Creating solver session...",
            [
                "Load the base variables, constraints, and objective into nxusKit",
                "Push/Pop enables reversible what-if experiments without rebuilding",
                "Explanation support is enabled for unsat scenarios",
            ],
        )
        == StepAction.QUIT
    ):
        return 0

    start = time.time()
    summaries: list[ScenarioSummary] = []

    try:
        with SolverSession(config=SolverConfig(timeout_ms=30000)) as session:
            session.add_variables(variables)
            session.add_constraints(constraints)
            if objectives:
                session.set_objective(objectives[0])

            print("----------------------------------------")
            print("  Base Problem")
            print("----------------------------------------")
            print()

            base_result = solve(session)
            print(f"Status: {base_result.status.value}")
            print_assignments(base_result)
            if base_result.objective_value is not None:
                print(f"Objective value: {base_result.objective_value:.2f}")
            print_stats(base_result, config)
            print()

            base_assignments = extract_assignments(base_result)
            base_objective = base_result.objective_value
            summaries.append(
                ScenarioSummary("Base", base_result.status.value, base_objective)
            )

            print("----------------------------------------")
            print("  What-If Analysis")
            print("----------------------------------------")
            print()

            for item in what_if_scenarios:
                name = item["name"]
                description = item["description"]
                additions = [
                    parse_constraint(constraint)
                    for constraint in item.get("additional_constraints", [])
                ]

                if (
                    config.step_pause(
                        f'What-if: "{name}"',
                        [
                            description,
                            "Push saves the base model state",
                            "Temporary constraints are added only for this solve",
                            "Pop restores the original baseline afterward",
                        ],
                    )
                    == StepAction.QUIT
                ):
                    return 0

                print(f'Scenario: "{name}"')
                print(f"  {description}")
                print("  Push scope...")
                session.push()
                try:
                    if additions:
                        session.add_constraints(additions)
                    variant_result = solve(session)
                finally:
                    print("  Pop scope (restoring base model)")
                    session.pop()

                print(f"  Status: {variant_result.status.value}")

                if variant_result.status in {SolveStatus.SAT, SolveStatus.OPTIMAL}:
                    variant_assignments = extract_assignments(variant_result)
                    if variant_result.objective_value is not None:
                        print(
                            f"  Objective value: {variant_result.objective_value:.2f}"
                        )
                    print("  Delta from base:")
                    print_delta(base_assignments, variant_assignments)
                    print_stats(variant_result, config)
                    summaries.append(
                        ScenarioSummary(
                            name,
                            variant_result.status.value,
                            variant_result.objective_value,
                        )
                    )
                else:
                    explanation = variant_result.explanation or session.explanation()
                    print_explanation(explanation)
                    summaries.append(ScenarioSummary(name, variant_result.status.value))

                print()

    except SolverLibraryNotFoundError as exc:
        print(f"Error: {exc}")
        return 1
    except SolverError as exc:
        print(f"Error: solver operation failed: {exc}")
        return 1

    print("========================================")
    print("  Summary")
    print("========================================")
    print()

    name_width = max(len(item.name) for item in summaries)
    print(f"  {'Variant':<{name_width}}  {'Status':>10}  {'Objective':>12}")
    print(f"  {'-' * name_width}  {'-' * 10}  {'-' * 12}")
    for item in summaries:
        objective = (
            "-" if item.objective_value is None else f"{item.objective_value:.2f}"
        )
        print(f"  {item.name:<{name_width}}  {item.status:>10}  {objective:>12}")

    elapsed_ms = int((time.time() - start) * 1000)
    if base_objective is not None:
        print(f"\nBase objective: {base_objective:.2f}")
    print(f"Total time: {elapsed_ms}ms")
    print("Done.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
