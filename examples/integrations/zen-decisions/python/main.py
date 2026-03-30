#!/usr/bin/env python3
"""Example: ZEN Decision Tables — nxuskit

## nxusKit Features Demonstrated
- zen_evaluate() for JSON Decision Model (JDM) evaluation
- Decision tables with "first" hit policy (maze-rat personality variants)
- Decision tables with "collect" hit policy (potion recipe selector)
- Expression nodes for computed outputs (food-truck planner)
- Multiple scenario comparison

## Interactive Modes
- `--verbose` or `-v`: Show raw JSON payloads and evaluation details
- `--step` or `-s`: Pause at each evaluation with explanations

## Scenarios
- maze-rat: Personality variant comparison (cautious, greedy, explorer)
- potion: Collect hit policy — returns all matching recipes
- food-truck: Decision + expression pipeline

Usage:
    python main.py                      # Default: maze-rat
    python main.py --scenario potion
    python main.py --scenario food-truck --verbose
"""

from __future__ import annotations

import argparse
import json
import os
import sys
from pathlib import Path

# Add shared python module to path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), "../../../shared/python"))

from interactive import InteractiveConfig, StepAction
from nxuskit import zen_evaluate

# -- Constants ---------------------------------------------------------------

SCENARIOS_DIR = Path(__file__).resolve().parent.parent / "scenarios"
SEPARATOR = "=" * 50


def main() -> int:
    config = InteractiveConfig.from_args()

    parser = argparse.ArgumentParser(description="ZEN Decision Table Example")
    parser.add_argument(
        "--scenario",
        "-S",
        default="maze-rat",
        choices=["maze-rat", "potion", "food-truck"],
        help="Scenario to run (default: maze-rat)",
    )
    # Filter out interactive flags before parsing
    filtered_args = [
        a for a in sys.argv[1:] if a not in ("--verbose", "-v", "--step", "-s")
    ]
    args = parser.parse_args(filtered_args)

    # Validate scenario directory
    scenario_dir = SCENARIOS_DIR / args.scenario
    if not scenario_dir.exists():
        available = [
            d.name
            for d in SCENARIOS_DIR.iterdir()
            if d.is_dir() and (d / "input.json").exists()
        ]
        print(f"Error: Scenario '{args.scenario}' not found.")
        print(f"Available: {', '.join(available)}")
        return 1

    # Run the selected scenario
    runners = {
        "maze-rat": run_maze_rat,
        "potion": run_potion,
        "food-truck": run_food_truck,
    }

    try:
        return runners[args.scenario](scenario_dir, config)
    except Exception as e:
        print(f"Error: {e}")
        return 1


def run_maze_rat(scenario_dir: Path, config: InteractiveConfig) -> int:
    """Compare personality variants with first-hit decision tables."""
    input_data = _load_json(scenario_dir / "input.json")

    print(f"{SEPARATOR}")
    print("ZEN Decision Tables: Maze Rat")
    print(f"{SEPARATOR}")
    print("Personality variant comparison with first-hit policy\n")

    print("Input:")
    for key, val in input_data.items():
        print(f"  {key}: {val}")
    print()

    personalities = [
        ("cautious", "decision-model.json"),
        ("greedy", "greedy.json"),
        ("explorer", "explorer.json"),
    ]

    results: list[tuple[str, dict]] = []

    for name, filename in personalities:
        model_path = scenario_dir / filename
        if not model_path.exists():
            print(f"  {name}: model file not found, skipping")
            continue

        if (
            config.step_pause(
                f"Evaluating {name} personality...",
                [
                    f"Loading JDM: {filename}",
                    "First-hit policy: first matching rule determines action",
                ],
            )
            == StepAction.QUIT
        ):
            return 0

        model = _load_json(model_path)

        # nxusKit: zen_evaluate — evaluate JDM against input
        result = zen_evaluate(model, input_data)
        results.append((name, result))

        print(f"--- {name} ---")
        _print_result(result, config)
        print()

    # Comparison table
    if results:
        print("--- Personality Comparison ---")
        print(f"  {'Personality':<12} {'Action':<18} {'Confidence':>10}")
        print(f"  {'-' * 42}")
        for name, result in results:
            action = result.get("action", "?")
            confidence = result.get("confidence", 0)
            print(f"  {name:<12} {action:<18} {confidence:>10.2f}")

    return 0


def run_potion(scenario_dir: Path, config: InteractiveConfig) -> int:
    """Evaluate potion recipes with collect hit policy."""
    input_data = _load_json(scenario_dir / "input.json")
    model = _load_json(scenario_dir / "decision-model.json")

    print(f"{SEPARATOR}")
    print("ZEN Decision Tables: Potion Recipes")
    print(f"{SEPARATOR}")
    print("Collect hit policy — returns all matching recipes\n")

    print("Input:")
    for key, val in input_data.items():
        print(f"  {key}: {val}")
    print()

    if (
        config.step_pause(
            "Evaluating potion recipes...",
            [
                "Collect hit policy returns ALL matching rules",
                "Multiple recipes can match the same input",
            ],
        )
        == StepAction.QUIT
    ):
        return 0

    result = zen_evaluate(model, input_data)

    if isinstance(result, list):
        print(f"Matching recipes: {len(result)}\n")
        for i, recipe in enumerate(result, 1):
            print(f"  Recipe {i}:")
            _print_result(recipe, config, indent=4)
    else:
        print("Result:")
        _print_result(result, config)

    return 0


def run_food_truck(scenario_dir: Path, config: InteractiveConfig) -> int:
    """Evaluate food truck decision + expression pipeline."""
    input_data = _load_json(scenario_dir / "input.json")
    model = _load_json(scenario_dir / "decision-model.json")

    print(f"{SEPARATOR}")
    print("ZEN Decision Tables: Food Truck Planner")
    print(f"{SEPARATOR}")
    print("Decision table + expression node pipeline\n")

    print("Input:")
    for key, val in input_data.items():
        print(f"  {key}: {val}")
    print()

    if (
        config.step_pause(
            "Evaluating food truck decision...",
            [
                "Decision table selects location and price multiplier",
                "Expression node computes menu adjustment and restock alert",
                "Pipeline: input → decision → expression → output",
            ],
        )
        == StepAction.QUIT
    ):
        return 0

    result = zen_evaluate(model, input_data)

    print("Decision output:")
    _print_result(result, config)

    return 0


def _load_json(path: Path) -> dict:
    """Load and parse a JSON file."""
    with open(path) as f:
        return json.load(f)


def _print_result(result: dict, config: InteractiveConfig, indent: int = 2) -> None:
    """Print result fields with indentation."""
    prefix = " " * indent
    if isinstance(result, dict):
        for key, value in result.items():
            print(f"{prefix}{key}: {value}")
    else:
        print(f"{prefix}{result}")

    if config.verbose:
        print(f"\n{prefix}[verbose] Raw: {json.dumps(result, indent=2)}")


if __name__ == "__main__":
    sys.exit(main())
