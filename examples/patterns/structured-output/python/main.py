#!/usr/bin/env python3
"""Example: Structured Output - nxuskit

## nxusKit Features Demonstrated
- ResponseFormat.JSON for requesting structured JSON responses
- Provider-agnostic JSON mode (Claude adds system instruction, OpenAI uses API param)
- Consistent error handling with typed exceptions
- Optional Pydantic validation with graceful dict fallback

## Interactive Modes
- `--verbose` or `-v`: Show raw request/response data
- `--step` or `-s`: Pause at each step with explanations

## Why This Pattern Matters
LLMs produce free-form text by default. Structured output (JSON mode) constrains
the model to return valid JSON, enabling reliable downstream parsing. nxusKit
abstracts the provider-specific mechanisms behind a single ResponseFormat enum.

Usage:
    export ANTHROPIC_API_KEY="your-key-here"
    python main.py
    python main.py --verbose    # Show request/response details
    python main.py --step       # Step through with explanations

Or with Ollama (no API key needed):
    python main.py ollama
"""

import json
import os
import sys
import time
from typing import Any, Dict, List, Optional, Tuple

# Add shared python module to path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), "../../../shared/python"))

from interactive import InteractiveConfig, StepAction
from nxuskit import LLMError, Message, Provider, ResponseFormat

# --- Optional Pydantic support ---
# If Pydantic is installed, we get richer validation and type coercion.
# Otherwise we fall back to plain dict checks.
try:
    from pydantic import BaseModel, ValidationError

    class SentimentResult(BaseModel):
        """Pydantic model for sentiment classification."""

        text: str
        sentiment: str  # positive, negative, neutral
        confidence: float
        keywords: List[str]

    PYDANTIC_AVAILABLE = True
except ImportError:
    PYDANTIC_AVAILABLE = False
    SentimentResult = None  # type: ignore[assignment,misc]


# --- Dict-based validation fallback ---
REQUIRED_FIELDS = {
    "text": str,
    "sentiment": str,
    "confidence": (int, float),
    "keywords": list,
}
VALID_SENTIMENTS = {"positive", "negative", "neutral"}


def validate_dict(data: Dict[str, Any]) -> List[str]:
    """Validate parsed JSON against expected schema, returning a list of errors."""
    errors: List[str] = []
    for field_name, expected_type in REQUIRED_FIELDS.items():
        if field_name not in data:
            errors.append(f"Missing field: {field_name}")
        elif not isinstance(data[field_name], expected_type):
            errors.append(
                f"Field '{field_name}' should be {expected_type}, got {type(data[field_name])}"
            )

    if "sentiment" in data and data["sentiment"] not in VALID_SENTIMENTS:
        errors.append(
            f"Invalid sentiment '{data['sentiment']}'; expected one of {VALID_SENTIMENTS}"
        )

    if "confidence" in data:
        conf = data["confidence"]
        if isinstance(conf, (int, float)) and not (0.0 <= conf <= 1.0):
            errors.append(f"Confidence {conf} out of range [0.0, 1.0]")

    return errors


def display_result(data: Dict[str, Any]) -> None:
    """Pretty-print a classification result."""
    print(f"  Text:       {data.get('text', 'N/A')}")
    print(f"  Sentiment:  {data.get('sentiment', 'N/A')}")
    print(f"  Confidence: {data.get('confidence', 'N/A')}")
    keywords = data.get("keywords", [])
    print(f"  Keywords:   {', '.join(keywords) if keywords else 'none'}")


def classify_text(
    provider,
    text: str,
    config: InteractiveConfig,
) -> Optional[Dict[str, Any]]:
    """Send a classification request using JSON mode and validate the result."""
    prompt = (
        "Classify the sentiment of the following text. "
        "Return ONLY a JSON object with these fields:\n"
        '  "text": the original text,\n'
        '  "sentiment": one of "positive", "negative", or "neutral",\n'
        '  "confidence": a float between 0.0 and 1.0,\n'
        '  "keywords": a list of relevant keywords.\n\n'
        f"Text: {text}"
    )

    messages = [
        Message.system(
            "You are a sentiment analysis engine. Always respond with valid JSON."
        ),
        Message.user(prompt),
    ]

    config.print_request(
        "POST",
        f"https://api.{provider.provider_name}.com/v1/chat",
        {"response_format": "json", "messages": [m.__dict__ for m in messages]},
    )

    start = time.time()
    # nxusKit: ResponseFormat.JSON tells the provider to constrain output to valid JSON
    response = provider.chat(
        messages,
        temperature=0.2,
        max_tokens=300,
        response_format=ResponseFormat.JSON,
    )
    elapsed_ms = int((time.time() - start) * 1000)

    config.print_response(200, elapsed_ms, {"content": response.content[:200]})

    # Parse raw JSON
    try:
        data = json.loads(response.content)
    except json.JSONDecodeError as exc:
        print(f"  [ERROR] Invalid JSON from model: {exc}")
        print(f"  Raw content: {response.content[:300]}")
        return None

    # Validate with Pydantic if available, otherwise dict checks
    if PYDANTIC_AVAILABLE:
        try:
            result = SentimentResult(**data)
            return result.model_dump()
        except ValidationError as exc:
            print(f"  [WARN] Pydantic validation failed: {exc}")
            print("  Falling back to dict validation...")

    errors = validate_dict(data)
    if errors:
        for err in errors:
            print(f"  [WARN] {err}")
    return data


def create_provider(provider_name: str) -> Tuple[Any, Optional[str]]:
    """Create a provider, returning (provider, model) or (None, None)."""
    if provider_name == "claude":
        api_key = os.environ.get("ANTHROPIC_API_KEY")
        if not api_key:
            print("ANTHROPIC_API_KEY not set -- trying Ollama instead")
            return Provider.ollama(model="llama3"), "llama3"
        return Provider.claude(
            model="claude-haiku-4-5-20251001", api_key=api_key
        ), "claude-haiku-4-5-20251001"

    if provider_name == "openai":
        api_key = os.environ.get("OPENAI_API_KEY")
        if not api_key:
            print("Error: OPENAI_API_KEY not set")
            return None, None
        return Provider.openai(model="gpt-4o", api_key=api_key), "gpt-4o"

    if provider_name == "ollama":
        return Provider.ollama(model="llama3"), "llama3"

    print(f"Unknown provider: {provider_name}. Supported: claude, openai, ollama")
    return None, None


def main() -> int:
    config = InteractiveConfig.from_args()

    print("=== Structured Output (JSON Mode) Example ===")
    print()

    if PYDANTIC_AVAILABLE:
        print("Pydantic detected -- using model-based validation")
    else:
        print("Pydantic not installed -- using dict-based validation")
    print()

    # Step: Explain JSON mode
    if (
        config.step_pause(
            "About to request JSON-formatted responses...",
            [
                "nxusKit: ResponseFormat.JSON constrains the model to output valid JSON",
                "Claude: appends a JSON instruction to the system message",
                "OpenAI: sets response_format={'type': 'json_object'}",
            ],
        )
        == StepAction.QUIT
    ):
        return 0

    args = [a for a in sys.argv[1:] if not a.startswith("-")]
    provider_name = args[0] if args else "claude"

    provider, model = create_provider(provider_name)
    if provider is None:
        return 1

    print(f"Using provider: {provider.provider_name} ({model})")
    print()

    # Sample texts to classify
    samples = [
        "I absolutely love this new feature, it makes everything so much easier!",
        "The server has been down for three hours and nobody is responding to tickets.",
        "The package arrived on Tuesday as expected.",
    ]

    for i, text in enumerate(samples, 1):
        if (
            config.step_pause(
                f"Classifying sample {i}/{len(samples)}...",
                [f'Text: "{text[:60]}..."', "Sending JSON-mode request to provider"],
            )
            == StepAction.QUIT
        ):
            return 0

        print(f"--- Sample {i} ---")
        print(f'  Input: "{text}"')

        try:
            result = classify_text(provider, text, config)
            if result:
                display_result(result)
        except LLMError as e:
            print(f"  [ERROR] {e.provider}: {e}")
        print()

    print("Done. All samples classified.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
