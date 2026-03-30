#!/usr/bin/env python3
"""Example: Multi-Provider Comparison - nxuskit

## nxusKit Features Demonstrated
- Provider abstraction layer (LLMProvider protocol)
- Unified response structure across different providers
- Provider-agnostic error handling
- Concurrent request execution (with ThreadPoolExecutor)

## Interactive Modes
- `--verbose` or `-v`: Show raw request/response data
- `--step` or `-s`: Pause at each step with explanations

## Why This Pattern Matters
Running the same prompt across providers enables A/B testing, cost comparison,
and fallback strategies. nxusKit's unified interface makes this trivial -
all providers return the same ChatResponse type with normalized token usage.

Usage:
    export ANTHROPIC_API_KEY="your-key-here"
    export OPENAI_API_KEY="your-key-here"
    python main.py
    python main.py --verbose    # Show request/response details
    python main.py --step       # Step through with explanations
"""

import os
import sys
import time
from concurrent.futures import ThreadPoolExecutor, as_completed
from typing import Any, Dict

# Add shared python module to path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), "../../../shared/python"))

from interactive import InteractiveConfig, StepAction
from nxuskit import LLMError, Message, Provider


def main():
    # Parse interactive mode flags
    config = InteractiveConfig.from_args()

    print("=== Multi-Provider Comparison Example ===")
    print()

    question = "In one sentence, what makes Python unique among programming languages?"

    # Step: Setting up providers
    if (
        config.step_pause(
            "Creating multiple LLM providers...",
            [
                "nxusKit: Each provider uses the same factory pattern",
                "Providers are created independently and can fail gracefully",
                "All providers implement the same LLMProvider protocol",
            ],
        )
        == StepAction.QUIT
    ):
        return 0

    # nxusKit: Create providers using the factory pattern
    providers = create_providers()

    if not providers:
        print("No providers available!")
        print("\nTo use this example, configure at least one provider:")
        print("  - export ANTHROPIC_API_KEY='...' for Claude")
        print("  - export OPENAI_API_KEY='...' for OpenAI")
        print("  - Start Ollama locally (ollama serve)")
        return 1

    # Step: Building requests
    if (
        config.step_pause(
            "Building identical requests for each provider...",
            [
                "nxusKit: Request structure is provider-agnostic",
                "Same message types work across all providers",
                "Only the model name differs between providers",
            ],
        )
        == StepAction.QUIT
    ):
        return 0

    print(f"Question: {question}")
    print()
    print("=" * 80)

    messages = [Message.user(question)]

    # Step: Sending concurrent requests
    if (
        config.step_pause(
            "Sending concurrent requests to all providers...",
            [
                "nxusKit: Unified interface enables concurrent execution",
                "ThreadPoolExecutor runs all requests in parallel",
                "Each request returns the same ChatResponse type",
            ],
        )
        == StepAction.QUIT
    ):
        return 0

    # nxusKit: Unified interface enables easy concurrent execution
    results = run_concurrent_requests(providers, messages, config)

    # Display results
    for name, result in results.items():
        print()
        if "error" in result:
            print(f"{name}: Error - {result['error']}")
        else:
            # nxusKit: All providers return same ChatResponse structure
            response = result["response"]
            print(f"{name} ({response.model})")
            print(f"{response.content}")
            print(f"Tokens: {response.usage.total_tokens}")

    print()
    print("=" * 80)
    return 0


def create_providers() -> Dict[str, Any]:
    """Create all available providers."""
    providers = {}

    # Try Claude
    if api_key := os.environ.get("ANTHROPIC_API_KEY"):
        # nxusKit: Provider factory with explicit configuration
        providers["Claude"] = {
            "provider": Provider.claude(
                model="claude-haiku-4-5-20251001", api_key=api_key
            ),
            "model": "claude-haiku-4-5-20251001",
            "url": "https://api.anthropic.com/v1/messages",
        }

    # Try OpenAI
    if api_key := os.environ.get("OPENAI_API_KEY"):
        providers["OpenAI"] = {
            "provider": Provider.openai(model="gpt-4o", api_key=api_key),
            "model": "gpt-4o",
            "url": "https://api.openai.com/v1/chat/completions",
        }

    # Try Ollama (always attempt - it's local)
    try:
        # nxusKit: Local provider needs no API key
        providers["Ollama"] = {
            "provider": Provider.ollama(model="llama3"),
            "model": "llama3",
            "url": "http://localhost:11434/api/chat",
        }
    except Exception:
        pass  # Ollama not available

    return providers


def run_concurrent_requests(
    providers: Dict, messages, config: InteractiveConfig
) -> Dict[str, Any]:
    """Run requests concurrently using ThreadPoolExecutor."""
    results = {}

    def make_request(name: str, provider_info: Dict) -> tuple:
        """Make a single provider request."""
        try:
            # Verbose: Show request
            config.print_request(
                "POST",
                provider_info["url"],
                {"messages": [{"role": "user", "content": messages[0].content}]},
            )

            start = time.time()
            # nxusKit: Same chat() interface for all providers
            response = provider_info["provider"].chat(
                messages,
                temperature=0.5,
                max_tokens=100,
            )
            elapsed_ms = int((time.time() - start) * 1000)

            # Verbose: Show response
            config.print_response(
                200,
                elapsed_ms,
                {"content": response.content, "model": response.model},
            )

            return name, {"response": response}
        except LLMError as e:
            return name, {"error": str(e)}

    # nxusKit: ThreadPoolExecutor for concurrent provider calls
    with ThreadPoolExecutor(max_workers=len(providers)) as executor:
        futures = {
            executor.submit(make_request, name, info): name
            for name, info in providers.items()
        }

        for future in as_completed(futures):
            name, result = future.result()
            results[name] = result

    return results


if __name__ == "__main__":
    sys.exit(main())
