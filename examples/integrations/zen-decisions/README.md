# ZEN Decision Tables -- Personality Variants & Hit Policies

Demonstrates ZEN JSON Decision Model (JDM) evaluation through three scenarios: a maze-navigating rat with interchangeable personality variants, a potion recipe selector using collect hit policy, and a food truck planner combining decision tables with expression nodes. This is the first nxusKit example built on the ZEN decision engine.

> Evaluate business decision tables in Go, Rust, and Python using the ZEN engine — from pricing rules to eligibility checks, with first-hit and collect hit policies.

**Scenarios**: `maze-rat` · `potion` · `food-truck`

## Edition

**Pro** — requires a Pro (or trial) entitlement for ZEN evaluation APIs.

## What this demonstrates

**Difficulty: Starter** 🟢 · ZEN

- **Summary:** ZEN decision table evaluation via nxusKit SDK
- **Scenario:** Evaluate business decision tables using the ZEN engine
- **`tech_tags` in manifest:** `ZEN` — example id **`zen-decisions`** in `conformance/examples_manifest.json`.

## Prerequisites

- **SDK:** Use an installed SDK tree (`NXUSKIT_SDK_DIR`, `NXUSKIT_LIB_PATH` as needed); `test-examples.sh` resolves Go/Rust/Python deps from that tree only — see [README.md](../../../README.md), `scripts/setup-sdk.sh`, and `scripts/test-examples.sh`.
- **Languages in this example:** go, rust (paths under this directory; Python may live under a sibling `python/` or shared reference per **Language Implementations**).

## Key nxusKit Features Demonstrated

| Feature | Description | Rust | Go |
|---------|-------------|------|-----|
| **ZEN Evaluate** | Stateless JDM evaluation | `nxuskit_zen_evaluate()` | `nxuskit-go.ZenEvaluate()` |
| **First Hit Policy** | Return first matching rule | `"hitPolicy": "first"` | `"hitPolicy": "first"` |
| **Collect Hit Policy** | Return all matching rules | `"hitPolicy": "collect"` | `"hitPolicy": "collect"` |
| **Expression Nodes** | Computed fields in pipeline | `expressionNode` | `expressionNode` |
| **Personality Variants** | Same input, different models | Multiple JDM files | Multiple JDM files |
| **Result Freeing** | Memory management for results | `nxuskit_zen_free_result()` | Automatic (Go wrapper) |

## Real-World Application

Pricing rules, eligibility determination, policy evaluation.

## Technologies

ZEN

## Pipeline Architecture

### Maze Rat (First Hit Policy)

```
                    ┌──────────────────┐
                    │  cautious JDM    │──> action: sniff_around
                    └──────────────────┘
┌───────────┐       ┌──────────────────┐
│   Input    │──┬──>│  greedy JDM      │──> action: go_right
│ (shared)   │  │   └──────────────────┘
└───────────┘  │   ┌──────────────────┐
                └──>│  explorer JDM    │──> action: go_right
                    └──────────────────┘
```

### Potion (Collect Hit Policy)

```
┌───────────┐    ┌──────────────────┐    ┌──────────────────┐
│   Input    │──>│  Decision Table   │──>│  All Matching     │
│            │   │  (collect policy) │   │  Recipes          │
└───────────┘    └──────────────────┘    └──────────────────┘
```

### Food Truck (Decision + Expression Pipeline)

```
┌───────────┐    ┌──────────────────┐    ┌──────────────────┐    ┌──────────┐
│   Input    │──>│  Decision Table   │──>│  Expression Node  │──>│  Output   │
│            │   │  (location/price) │   │  (menu/restock)   │   │          │
└───────────┘    └──────────────────┘    └──────────────────┘    └──────────┘
```

## Build

Attach an **installed SDK** (`NXUSKIT_SDK_DIR`). See the repository [README.md](../../../README.md) and `scripts/test-examples.sh`.

```bash
# From `/examples/integrations/zen-decisions`:
cd rust && cargo build
cd go && make build
```

## Run

### Rust

```bash
cd rust
cargo run -- --scenario maze-rat
cargo run -- --scenario potion --verbose
cargo run -- --scenario food-truck --step
```

### Go

```bash
cd go
make build
./bin/zen-decisions --scenario maze-rat
./bin/zen-decisions --scenario potion --verbose
./bin/zen-decisions --scenario food-truck --step
```

Or directly:

```bash
cd go
go run . --scenario maze-rat
```

## Scenarios

### Maze Rat (Personality Variants)

A maze-navigating rat with three interchangeable personality variants, each encoded as a separate JDM file. All share the same input schema but produce different actions.

**Input fields**: `scent_strength` (1-10), `dead_ends_nearby` (0-5), `hunger_level` (1-10)

**Output fields**: `action` (go_left, go_right, backtrack, sniff_around, rest), `confidence` (0.0-1.0)

| Personality | Strategy | Key Trait |
|-------------|----------|-----------|
| **Cautious** | Avoids dead ends, backtracks when risky | High dead_ends triggers backtrack |
| **Greedy** | Always follows scent, ignores dead ends | Strong scent always means go_right |
| **Explorer** | Seeks new paths, ignores scent | Prefers go_left to explore unknown areas |

With the default input (`scent_strength=7, dead_ends_nearby=2, hunger_level=4`):
- Cautious: sniff_around (0.60 confidence) -- strong scent but hesitant
- Greedy: go_right (0.95 confidence) -- immediately follows strong scent
- Explorer: go_right (0.80 confidence) -- few dead ends, worth exploring right

### Potion (Collect Hit Policy)

A potion recipe selector that returns all matching recipes using the collect hit policy. Multiple recipes can match the same ingredients and desired effect.

**Input fields**: `ingredients` (comma-separated list), `desired_effect` (healing, strength, invisibility, speed), `caution_level` (low, medium, high)

**Output fields**: `recipe_name`, `recipe_steps`, `warnings`

With the default input (`ingredients=mushroom,moonstone, desired_effect=healing, caution_level=medium`):
- Healing Tonic (mushroom + healing matches)
- Unknown Brew (default catch-all also matches under collect policy)

### Food Truck (Expression Nodes)

A food truck location and menu planner that chains a decision table with an expression node. The decision table selects location and base pricing; the expression node computes menu adjustments.

**Input fields**: `time_of_day`, `weather`, `nearby_events`, `inventory_level`

**Output fields**: `location`, `price_multiplier`, `menu_adjustment`, `restock_alert`

With the default input (`time_of_day=lunch, weather=sunny, nearby_events=large, inventory_level=medium`):
- Location: Main Street Park (lunch + sunny + large event)
- Price multiplier: 1.3 (premium for large event)
- Menu: full menu (medium inventory)
- Restock alert: false

## Decision Table Concepts

### Hit Policies

| Policy | Behavior | Use Case |
|--------|----------|----------|
| **first** | Return the first matching rule | Prioritized rules, fallback chains |
| **collect** | Return all matching rules | Multiple recommendations, audit trails |

With **first** hit policy, rule order matters -- the first match wins. With **collect**, all matching rules are returned as an array.

### Expression Nodes

Expression nodes compute derived fields using simple expressions:

```json
{
    "type": "expressionNode",
    "content": {
        "expressions": [
            {"key": "output_field", "value": "input_field"},
            {"key": "computed", "value": "if condition then 'a' else 'b'"}
        ]
    }
}
```

Expressions support:
- Field references: `field_name` passes through a value
- Conditionals: `if condition then value1 else value2`
- Comparisons: `==`, `!=`, `>`, `<`, `>=`, `<=`
- String literals: `'single quoted'`

### CLIPS vs ZEN Comparison

| Aspect | CLIPS (Rule Engine) | ZEN (Decision Tables) |
|--------|--------------------|-----------------------|
| **Model format** | `.clp` rule files | `.json` JDM files |
| **Evaluation** | Forward-chaining inference | Table lookup + expressions |
| **State** | Working memory (facts) | Stateless per evaluation |
| **Complexity** | Arbitrary rule logic | Structured table/expression |
| **Best for** | Complex reasoning chains | Configuration-driven decisions |
| **Hit policies** | N/A (all matching rules fire) | first, collect |
| **Session management** | Required (create/destroy) | None (stateless) |

## JDM (JSON Decision Model) Format

The JDM format follows the GoRules specification:

```json
{
    "contentType": "application/vnd.gorules.decision",
    "edges": [
        {"id": "e1", "sourceId": "input", "targetId": "dt1", "type": "edge"}
    ],
    "nodes": [
        {"id": "input", "type": "inputNode", ...},
        {"id": "dt1", "type": "decisionTableNode", "content": {...}},
        {"id": "output", "type": "outputNode", ...}
    ]
}
```

### Supported Node Types

| Type | Description |
|------|-------------|
| `inputNode` | Entry point; passes input data to connected nodes |
| `outputNode` | Exit point; returns the final result |
| `decisionTableNode` | Evaluates input against rules with configurable hit policy |
| `expressionNode` | Computes derived fields from expressions |
| `switchNode` | Routes to different branches based on conditions |

**Note**: `functionNode` (JavaScript) is not supported by the nxusKit ZEN provider.

### Rule Field Formats

- Comparison operators: `> 30`, `<= 10`, `>= 7`, `== 5`, `!= 0`
- String values (in outputs): `"\"cool\""` (double-quoted inside JSON)
- Numeric values: `0.85`, `1.3`
- Boolean values: `true`, `false`
- Empty field: `""` (matches any value / wildcard)

## Interactive Modes

```bash
# Verbose mode -- show raw JSON results
cargo run -- --scenario maze-rat --verbose     # Rust
go run . --scenario maze-rat --verbose         # Go

# Step mode -- pause at each step with explanations
cargo run -- --scenario potion --step          # Rust
go run . --scenario potion --step              # Go

# Combined mode
cargo run -- --scenario food-truck --verbose --step
go run . --scenario food-truck --verbose --step
```

Or use environment variables:

```bash
export NXUSKIT_VERBOSE=1
export NXUSKIT_STEP=1
```

## Testing

```bash
# Rust
cd rust && cargo test

# Go
cd go && go test -v
```

Each scenario includes an `expected-output.json` that describes expected results for regression testing.
