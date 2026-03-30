#!/usr/bin/env python3
"""Example: Cost-Based Model Routing - nxuskit

## nxusKit Features Demonstrated
- Provider factory for multiple model tiers (cheap/standard/premium)
- Unified chat interface enabling transparent model switching
- Task complexity classification for intelligent routing
- Consistent token tracking across providers for cost estimation

## Interactive Modes
- `--verbose` or `-v`: Show raw request/response data
- `--step` or `-s`: Pause at each step with explanations

## Why This Pattern Matters
Not every request needs the most expensive model. Simple lookups can use a fast,
cheap model while complex reasoning tasks warrant a premium one. Cost-routing
lets you optimize spend without sacrificing quality where it matters.

Usage:
    export ANTHROPIC_API_KEY="your-key-here"
    python main.py
    python main.py --verbose    # Show routing decisions and costs
    python main.py --step       # Step through with explanations

Or with Ollama (no API key needed):
    python main.py ollama
"""

import os
import sys
import time
from dataclasses import dataclass
from enum import Enum
from typing import Dict, List, Optional, Tuple

# Add shared python module to path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), "../../../shared/python"))

from interactive import InteractiveConfig, StepAction
from nxuskit import LLMError, Message, Provider


class Complexity(Enum):
    """Task complexity levels."""

    SIMPLE = "simple"
    MEDIUM = "medium"
    COMPLEX = "complex"


@dataclass
class ModelTier:
    """A model tier with cost metadata."""

    name: str
    complexity: Complexity
    cost_per_1k_input: float  # USD per 1K input tokens
    cost_per_1k_output: float  # USD per 1K output tokens
    provider_factory: callable  # returns provider or None
    description: str


def estimate_cost(tier: ModelTier, input_tokens: int, output_tokens: int) -> float:
    """Estimate cost in USD for a request."""
    input_cost = (input_tokens / 1000.0) * tier.cost_per_1k_input
    output_cost = (output_tokens / 1000.0) * tier.cost_per_1k_output
    return input_cost + output_cost


def classify_complexity(task: str) -> Complexity:
    """Classify task complexity using simple heuristics.

    In production, you might use a small classifier model or keyword matching
    tuned to your domain. This heuristic is intentionally simple for the example.
    """
    task_lower = task.lower()

    # Complex indicators: multi-step reasoning, code generation, analysis
    complex_signals = [
        "explain",
        "analyze",
        "compare",
        "design",
        "architect",
        "implement",
        "debug",
        "optimize",
        "trade-off",
        "tradeoff",
        "pros and cons",
        "step by step",
        "algorithm",
    ]

    # Simple indicators: short lookups, definitions, yes/no
    simple_signals = [
        "what is",
        "define",
        "translate",
        "convert",
        "list",
        "name",
        "how many",
        "true or false",
        "yes or no",
    ]

    complex_score = sum(1 for s in complex_signals if s in task_lower)
    simple_score = sum(1 for s in simple_signals if s in task_lower)

    # Also consider length: long prompts tend to be more complex
    word_count = len(task.split())
    if word_count > 50:
        complex_score += 2
    elif word_count < 10:
        simple_score += 1

    if complex_score > simple_score:
        return Complexity.COMPLEX
    if simple_score > complex_score:
        return Complexity.SIMPLE
    return Complexity.MEDIUM


def build_tiers(force_ollama: bool = False) -> Dict[Complexity, ModelTier]:
    """Build model tiers based on available credentials.

    When force_ollama is True (or no cloud keys are set), all tiers use Ollama
    with the same model -- useful for local development and testing.
    """
    anthropic_key = os.environ.get("ANTHROPIC_API_KEY")

    use_cloud = not force_ollama and (anthropic_key is not None)

    if use_cloud:
        tiers = {
            Complexity.SIMPLE: ModelTier(
                name="claude-haiku-4-5-20251001",
                complexity=Complexity.SIMPLE,
                cost_per_1k_input=0.00025,
                cost_per_1k_output=0.00125,
                provider_factory=lambda: Provider.claude(
                    model="claude-haiku-4-5-20251001",
                    api_key=anthropic_key,
                ),
                description="Fast and cheap -- ideal for lookups and simple Q&A",
            ),
            Complexity.MEDIUM: ModelTier(
                name="claude-haiku-4-5-20251001",
                complexity=Complexity.MEDIUM,
                cost_per_1k_input=0.00025,
                cost_per_1k_output=0.00125,
                provider_factory=lambda: Provider.claude(
                    model="claude-haiku-4-5-20251001",
                    api_key=anthropic_key,
                ),
                description="Balanced cost/quality for moderate tasks",
            ),
            Complexity.COMPLEX: ModelTier(
                name="claude-haiku-4-5-20251001",
                complexity=Complexity.COMPLEX,
                cost_per_1k_input=0.00025,
                cost_per_1k_output=0.00125,
                provider_factory=lambda: Provider.claude(
                    model="claude-haiku-4-5-20251001",
                    api_key=anthropic_key,
                ),
                description="Premium model for reasoning (Haiku demo)",
            ),
        }
    else:
        # All Ollama -- cost is effectively zero (local compute)
        for_local = lambda: Provider.ollama(model="llama3")  # noqa: E731
        tiers = {
            Complexity.SIMPLE: ModelTier(
                name="llama3 (simple)",
                complexity=Complexity.SIMPLE,
                cost_per_1k_input=0.0,
                cost_per_1k_output=0.0,
                provider_factory=for_local,
                description="Local model for simple tasks",
            ),
            Complexity.MEDIUM: ModelTier(
                name="llama3 (medium)",
                complexity=Complexity.MEDIUM,
                cost_per_1k_input=0.0,
                cost_per_1k_output=0.0,
                provider_factory=for_local,
                description="Local model for medium tasks",
            ),
            Complexity.COMPLEX: ModelTier(
                name="llama3 (complex)",
                complexity=Complexity.COMPLEX,
                cost_per_1k_input=0.0,
                cost_per_1k_output=0.0,
                provider_factory=for_local,
                description="Local model for complex tasks",
            ),
        }

    return tiers


def route_and_execute(
    task: str,
    tiers: Dict[Complexity, ModelTier],
    config: InteractiveConfig,
) -> Optional[Tuple[Complexity, ModelTier, object, float]]:
    """Classify a task, route to the appropriate tier, and execute.

    Returns (complexity, tier, response, estimated_cost) or None on failure.
    """
    complexity = classify_complexity(task)
    tier = tiers[complexity]

    print(f"  Complexity: {complexity.value}")
    print(f"  Routed to:  {tier.name} -- {tier.description}")

    provider = tier.provider_factory()
    if provider is None:
        print("  [ERROR] Could not create provider for this tier")
        return None

    messages = [
        Message.system("Be concise. Answer in 2-3 sentences maximum."),
        Message.user(task),
    ]

    config.print_request(
        "POST",
        f"https://api.{provider.provider_name}.com/v1/chat",
        {"model": tier.name, "complexity": complexity.value},
    )

    start = time.time()
    try:
        response = provider.chat(messages, temperature=0.5, max_tokens=200)
    except LLMError as e:
        print(f"  [ERROR] {e.provider}: {e}")
        return None

    elapsed_ms = int((time.time() - start) * 1000)

    config.print_response(
        200,
        elapsed_ms,
        {"content": response.content[:100], "model": response.model},
    )

    cost = estimate_cost(
        tier,
        response.usage.input_tokens,
        response.usage.output_tokens,
    )

    return complexity, tier, response, cost


def main() -> int:
    config = InteractiveConfig.from_args()

    print("=== Cost-Based Model Routing Example ===")
    print()

    # Step: Explain routing strategy
    if (
        config.step_pause(
            "Setting up model tiers...",
            [
                "Simple tasks -> cheap/fast model (Haiku or local)",
                "Medium tasks -> balanced model",
                "Complex tasks -> premium model (Sonnet/Opus in production)",
                "nxusKit: Same Provider interface for all tiers",
            ],
        )
        == StepAction.QUIT
    ):
        return 0

    args = [a for a in sys.argv[1:] if not a.startswith("-")]
    force_ollama = "ollama" in args

    tiers = build_tiers(force_ollama=force_ollama)

    print("Model tiers configured:")
    for complexity, tier in sorted(tiers.items(), key=lambda x: x[0].value):
        print(f"  {complexity.value:>8}: {tier.name}")
        print(
            f"           Cost: ${tier.cost_per_1k_input}/1K in, ${tier.cost_per_1k_output}/1K out"
        )
    print()

    # Example tasks spanning different complexity levels
    tasks = [
        ("What is Python?", "Simple lookup / definition"),
        ("List three popular web frameworks.", "Moderate enumeration task"),
        (
            "Compare and analyze the trade-offs between microservices and monolithic "
            "architectures, considering scalability, deployment complexity, and team size.",
            "Complex multi-factor analysis",
        ),
        ("Translate 'hello' to French.", "Simple translation"),
        (
            "Design a step-by-step algorithm for detecting anomalies in time-series "
            "data, explaining the pros and cons of each approach.",
            "Complex algorithmic design",
        ),
    ]

    total_cost = 0.0
    results: List[Tuple[str, str, float]] = []

    for i, (task, description) in enumerate(tasks, 1):
        if (
            config.step_pause(
                f"Task {i}/{len(tasks)}: {description}",
                [
                    f'Task: "{task[:60]}..."',
                    "Classifying complexity and routing to appropriate tier",
                ],
            )
            == StepAction.QUIT
        ):
            return 0

        print(f"--- Task {i}: {description} ---")
        print(f'  Input: "{task[:80]}{"..." if len(task) > 80 else ""}"')

        result = route_and_execute(task, tiers, config)
        if result:
            complexity, tier, response, cost = result
            total_cost += cost
            results.append((complexity.value, tier.name, cost))

            print(f"  Response: {response.content[:120]}...")
            print(f"  Tokens: {response.usage.total_tokens} | Est. cost: ${cost:.6f}")
        print()

    # Summary
    print("=" * 60)
    print("Routing Summary")
    print("=" * 60)
    for complexity_val, model_name, cost in results:
        print(f"  {complexity_val:>8} -> {model_name}: ${cost:.6f}")
    print(f"\n  Total estimated cost: ${total_cost:.6f}")
    print()
    print("Done.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
