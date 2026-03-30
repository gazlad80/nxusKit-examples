#!/usr/bin/env python3
"""Example: CLIPS+LLM Hybrid — nxuskit

## nxusKit Features Demonstrated
- ClipsSession for deterministic rule-based routing (direct FFI)
- LLM for natural language classification and response generation
- Unified workflow: LLM preprocessing → CLIPS rules → LLM postprocessing
- Context manager pattern for session lifecycle

## Why This Pattern Matters
nxusKit's differentiator is not that it wraps CLIPS or LLMs separately —
it's that CLIPS, LLMs, Solver, and BN all speak the same interface.
This example shows what no other Python library can do: a unified
CLIPS + LLM workflow through one SDK.

## Interactive Modes
- `--verbose` or `-v`: Show raw LLM request/response and CLIPS facts
- `--step` or `-s`: Pause at each pipeline stage with explanations

Usage:
    python main.py
    python main.py --verbose
    python main.py ollama    # Use local Ollama instead of Claude
"""

from __future__ import annotations

import json
import os
import sys
from pathlib import Path

# Add shared python module to path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), "../../../shared/python"))

from interactive import InteractiveConfig, StepAction
from nxuskit import ClipsSession, LLMError, Message, Provider


# -- Constants ---------------------------------------------------------------

RULES_FILE = Path(__file__).resolve().parent.parent / "ticket-routing.clp"

SAMPLE_TICKETS = [
    (
        "Security Incident",
        "URGENT: We've detected unauthorized access attempts on our production "
        "database. Multiple failed login attempts from unknown IPs. Need immediate "
        "investigation!",
    ),
    (
        "Infrastructure Issue",
        "Database connection timeouts are causing checkout failures. Customers "
        "are complaining they can't complete purchases. Started after last "
        "night's deployment.",
    ),
    (
        "Application Bug",
        "The login button on the mobile app is not responding. Users have to "
        "force close and reopen the app. Started after the latest update.",
    ),
    (
        "General Inquiry",
        "Hi, I was wondering if you could help me understand how to export "
        "my data? The documentation is a bit unclear.",
    ),
]


def main() -> int:
    config = InteractiveConfig.from_args()

    print("=== CLIPS+LLM Hybrid Demo ===\n")
    print("Pipeline: LLM classifies ticket → CLIPS routes → LLM suggests response\n")

    # Determine provider
    args = [a for a in sys.argv[1:] if not a.startswith("-")]
    provider_name = args[0] if args else "claude"

    provider = create_provider(provider_name)
    if provider is None:
        return 1

    # Verify CLIPS rules exist
    if not RULES_FILE.exists():
        print(f"Error: CLIPS rules not found at {RULES_FILE}")
        print("Run from the repository root or the example directory.")
        return 1

    if (
        config.step_pause(
            "Starting hybrid pipeline...",
            [
                "Step 1: LLM classifies the ticket (category, priority, sentiment)",
                "Step 2: CLIPS applies deterministic routing rules",
                "Step 3: LLM generates suggested response",
            ],
        )
        == StepAction.QUIT
    ):
        return 0

    for label, ticket_text in SAMPLE_TICKETS:
        print(f"=== {label} ===")
        print(f"Ticket: {ticket_text[:80]}...\n")

        if (
            config.step_pause(
                "Analyzing ticket with hybrid pipeline...",
                [
                    "LLM extracts: category, priority, sentiment, security keywords",
                    "CLIPS evaluates routing rules against extracted facts",
                    "LLM generates an empathetic response suggestion",
                ],
            )
            == StepAction.QUIT
        ):
            return 0

        # --- Step 1: LLM Classification ---
        classification = classify_ticket(provider, ticket_text, config)
        if classification is None:
            print("  Classification failed — skipping\n")
            continue

        print("  LLM Classification:")
        for key, val in classification.items():
            print(f"    {key}: {val}")

        # --- Step 2: CLIPS Routing ---
        routing = route_with_clips(classification, config)
        if routing is None:
            print("  CLIPS routing failed — skipping\n")
            continue

        print("\n  CLIPS Routing (deterministic):")
        print(f"    Team: {routing['team']}")
        print(f"    SLA: {routing['sla_hours']} hours")
        print(f"    Escalation: Level {routing['escalation_level']}")

        # --- Step 3: LLM Response Suggestion ---
        suggestion = suggest_response(provider, ticket_text, routing, config)
        if suggestion:
            print(f"\n  Suggested Response:\n    {suggestion[:200]}")

        print(f"\n{'—' * 60}\n")

    print("=== Why Hybrid is Better ===")
    print("- LLM alone: May miss SLA policies, inconsistent routing")
    print("- CLIPS alone: Can't understand natural language input")
    print("- CLIPS + LLM: Best of both — understanding AND policy compliance")

    return 0


def create_provider(provider_name: str):
    """Create an LLM provider based on available configuration."""
    if provider_name == "claude":
        api_key = os.environ.get("ANTHROPIC_API_KEY")
        if not api_key:
            print("No ANTHROPIC_API_KEY set — trying Ollama...")
            return Provider.ollama(model="llama3")
        return Provider.claude(model="claude-haiku-4-5-20251001", api_key=api_key)
    elif provider_name == "ollama":
        return Provider.ollama(model="llama3")
    else:
        print(f"Unknown provider: {provider_name}. Use 'claude' or 'ollama'.")
        return None


def classify_ticket(
    provider, ticket_text: str, config: InteractiveConfig
) -> dict | None:
    """Use LLM to classify a support ticket into structured fields."""
    system_prompt = (
        "You are a support ticket classifier. Analyze the ticket and respond "
        "with ONLY a JSON object containing:\n"
        '- "category": one of "security", "infrastructure", "application", "general"\n'
        '- "priority": one of "low", "medium", "high", "critical"\n'
        '- "sentiment": one of "positive", "neutral", "negative", "frustrated"\n'
        '- "has_security_keywords": "yes" or "no"\n'
        "Respond with ONLY the JSON object, no markdown fences."
    )

    config.print_request("POST", "llm://classify", {"ticket": ticket_text[:50]})

    try:
        response = provider.chat(
            [Message.system(system_prompt), Message.user(ticket_text)],
            temperature=0.1,
            max_tokens=200,
        )
        config.print_response(200, 0, {"content": response.content})

        # Strip markdown fences if present
        content = response.content.strip()
        if content.startswith("```"):
            content = content.split("\n", 1)[1] if "\n" in content else content[3:]
            if content.endswith("```"):
                content = content[:-3]
            content = content.strip()

        return json.loads(content)
    except (LLMError, json.JSONDecodeError) as e:
        print(f"  Classification error: {e}")
        return None


def route_with_clips(classification: dict, config: InteractiveConfig) -> dict | None:
    """Use CLIPS rules to route a classified ticket."""
    try:
        # nxusKit: ClipsSession with context manager for automatic cleanup
        with ClipsSession() as clips:
            clips.load_file(str(RULES_FILE))
            clips.reset()

            # Assert the classification as a CLIPS fact
            category = classification.get("category", "general")
            priority = classification.get("priority", "medium")
            sentiment = classification.get("sentiment", "neutral")
            has_security = classification.get("has_security_keywords", "no")

            fact = (
                f"(ticket-classification "
                f"(category {category}) "
                f"(priority {priority}) "
                f"(sentiment {sentiment}) "
                f"(has-security-keywords {has_security}))"
            )
            clips.fact_assert_string(fact)

            # Run inference
            rules_fired = clips.run()

            if config.verbose:
                print(f"  [CLIPS] Rules fired: {rules_fired}")

            # Extract routing decision
            fact_indices = clips.facts_by_template("routing-decision")
            if not fact_indices:
                return {
                    "team": "general-support",
                    "sla_hours": 24,
                    "escalation_level": 0,
                }

            slots_json = clips.fact_slot_values(fact_indices[0])
            slots = json.loads(slots_json)

            # Unwrap typed ClipsValue format
            return {
                "team": _unwrap_clips(slots.get("team", "")),
                "sla_hours": _unwrap_clips_int(slots.get("sla-hours", 24)),
                "escalation_level": _unwrap_clips_int(slots.get("escalation-level", 0)),
            }
    except Exception as e:
        print(f"  CLIPS error: {e}")
        return None


def suggest_response(
    provider, ticket_text: str, routing: dict, config: InteractiveConfig
) -> str | None:
    """Use LLM to generate a suggested response based on routing decision."""
    prompt = (
        f"A support ticket has been routed to the {routing['team']} team "
        f"with a {routing['sla_hours']}-hour SLA (escalation level "
        f"{routing['escalation_level']}). Write a brief, empathetic "
        f"acknowledgment to the customer. Keep it under 3 sentences.\n\n"
        f"Original ticket: {ticket_text}"
    )

    try:
        response = provider.chat(
            [Message.user(prompt)], temperature=0.7, max_tokens=200
        )
        return response.content.strip()
    except LLMError:
        return None


def _unwrap_clips(value) -> str:
    """Extract string from a ClipsValue (may be {"type":"symbol","value":"x"})."""
    if isinstance(value, dict):
        return str(value.get("value", value))
    return str(value)


def _unwrap_clips_int(value) -> int:
    """Extract int from a ClipsValue."""
    if isinstance(value, dict):
        return int(value.get("value", 0))
    return int(value) if value else 0


if __name__ == "__main__":
    sys.exit(main())
