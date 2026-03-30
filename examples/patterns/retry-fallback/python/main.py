#!/usr/bin/env python3
"""Example: Retry and Provider Fallback - nxuskit

## nxusKit Features Demonstrated
- RetryConfig for configurable exponential backoff
- retry_with_backoff wrapper for automatic retry on transient errors
- Provider failover chain (Claude -> OpenAI -> Ollama)
- Typed exceptions: AuthenticationError, RateLimitError, NetworkError

## Interactive Modes
- `--verbose` or `-v`: Show raw request/response data
- `--step` or `-s`: Pause at each step with explanations

## Why This Pattern Matters
Production LLM applications need resilience. A single provider may be rate-limited,
down, or experiencing latency spikes. Combining per-provider retry with cross-provider
fallback gives you layered reliability without manual intervention.

Usage:
    export ANTHROPIC_API_KEY="your-key-here"
    python main.py
    python main.py --verbose    # Show retry attempts and failover
    python main.py --step       # Step through with explanations
"""

import os
import sys
import time
from dataclasses import dataclass
from typing import List, Optional, Tuple

# Add shared python module to path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), "../../../shared/python"))

from interactive import InteractiveConfig, StepAction
from nxuskit import (
    AuthenticationError,
    LLMError,
    Message,
    NetworkError,
    Provider,
    RateLimitError,
    RetryConfig,
    retry_with_backoff,
)


@dataclass
class ProviderSpec:
    """Specification for a provider in the fallback chain."""

    name: str
    factory: callable  # returns (provider, model_name) or None
    priority: int  # lower = tried first


def build_fallback_chain() -> List[ProviderSpec]:
    """Build an ordered list of provider specs based on available credentials.

    Priority: Claude (1) -> OpenAI (2) -> Ollama (3).
    Providers without credentials are still included -- they will fail at call
    time and the chain will advance to the next one.
    """
    chain: List[ProviderSpec] = []

    # Claude
    def make_claude():
        api_key = os.environ.get("ANTHROPIC_API_KEY")
        if not api_key:
            return None
        return Provider.claude(model="claude-haiku-4-5-20251001", api_key=api_key)

    chain.append(ProviderSpec(name="Claude", factory=make_claude, priority=1))

    # OpenAI
    def make_openai():
        api_key = os.environ.get("OPENAI_API_KEY")
        if not api_key:
            return None
        return Provider.openai(model="gpt-4o", api_key=api_key)

    chain.append(ProviderSpec(name="OpenAI", factory=make_openai, priority=2))

    # Ollama (always available, no key required)
    def make_ollama():
        return Provider.ollama(model="llama3")

    chain.append(ProviderSpec(name="Ollama", factory=make_ollama, priority=3))

    chain.sort(key=lambda s: s.priority)
    return chain


def chat_with_retry(
    provider,
    messages: List[Message],
    config: InteractiveConfig,
    retry_cfg: RetryConfig,
) -> Tuple[str, object]:
    """Call provider.chat with retry_with_backoff.

    Returns (provider_name, ChatResponse).
    """
    print(
        f"  Attempting {provider.provider_name} (max {retry_cfg.max_retries} retries)..."
    )

    config.print_request(
        "POST",
        f"https://api.{provider.provider_name}.com/v1/chat",
        {
            "messages": [m.__dict__ for m in messages],
            "max_retries": retry_cfg.max_retries,
        },
    )

    start = time.time()
    # nxusKit: retry_with_backoff wraps any callable with exponential backoff.
    # It respects LLMError.is_retryable to decide whether to retry.
    response = retry_with_backoff(
        provider.chat,
        messages,
        temperature=0.7,
        max_tokens=300,
        max_retries=retry_cfg.max_retries,
        initial_delay=retry_cfg.initial_delay,
        max_delay=retry_cfg.max_delay,
    )
    elapsed_ms = int((time.time() - start) * 1000)

    config.print_response(
        200,
        elapsed_ms,
        {"content": response.content[:120], "model": response.model},
    )
    return provider.provider_name, response


def chat_with_fallback(
    chain: List[ProviderSpec],
    messages: List[Message],
    config: InteractiveConfig,
    retry_cfg: RetryConfig,
) -> Optional[Tuple[str, object]]:
    """Try each provider in the chain, with per-provider retry.

    Returns (provider_name, ChatResponse) on the first success, or None.
    """
    errors: List[Tuple[str, str]] = []

    for spec in chain:
        provider = spec.factory()
        if provider is None:
            print(f"  Skipping {spec.name} (no credentials)")
            errors.append((spec.name, "no credentials"))
            continue

        try:
            return chat_with_retry(provider, messages, config, retry_cfg)
        except AuthenticationError as e:
            print(f"  {spec.name}: auth failed ({e})")
            errors.append((spec.name, f"auth: {e}"))
        except RateLimitError as e:
            print(f"  {spec.name}: rate limited ({e})")
            errors.append((spec.name, f"rate limit: {e}"))
        except NetworkError as e:
            print(f"  {spec.name}: network error ({e})")
            errors.append((spec.name, f"network: {e}"))
        except LLMError as e:
            print(f"  {spec.name}: LLM error ({e})")
            errors.append((spec.name, f"error: {e}"))

    print("\nAll providers failed:")
    for name, reason in errors:
        print(f"  - {name}: {reason}")
    return None


def main() -> int:
    config = InteractiveConfig.from_args()

    print("=== Retry and Provider Fallback Example ===")
    print()

    # Step: Explain retry configuration
    if (
        config.step_pause(
            "Configuring retry behavior...",
            [
                "nxusKit: RetryConfig controls max retries, initial delay, and backoff curve",
                "retry_with_backoff wraps any provider.chat call transparently",
                "Only errors with is_retryable=True trigger retries (e.g. RateLimitError, NetworkError)",
            ],
        )
        == StepAction.QUIT
    ):
        return 0

    # nxusKit: RetryConfig with explicit exponential backoff settings
    retry_cfg = RetryConfig(
        max_retries=2,
        initial_delay=0.5,
        max_delay=10.0,
        exponential_base=2.0,
        jitter=True,
    )

    print(
        f"Retry config: max_retries={retry_cfg.max_retries}, "
        f"initial_delay={retry_cfg.initial_delay}s, "
        f"max_delay={retry_cfg.max_delay}s, jitter={retry_cfg.jitter}"
    )
    print()

    # Step: Build fallback chain
    if (
        config.step_pause(
            "Building provider fallback chain...",
            [
                "Chain order: Claude -> OpenAI -> Ollama",
                "Each provider is tried with retry before falling back",
                "Ollama is the final fallback (runs locally, no API key needed)",
            ],
        )
        == StepAction.QUIT
    ):
        return 0

    chain = build_fallback_chain()
    available = [s.name for s in chain if s.factory() is not None]
    print(f"Fallback chain: {' -> '.join(s.name for s in chain)}")
    print(
        f"Available: {', '.join(available) if available else 'none (install Ollama as last resort)'}"
    )
    print()

    # --- Demonstrate single-provider retry ---
    if (
        config.step_pause(
            "Demo 1: Single provider with retry...",
            [
                "Calls the first available provider with retry_with_backoff",
                "Transient failures (rate limits, network blips) are retried automatically",
            ],
        )
        == StepAction.QUIT
    ):
        return 0

    print("--- Demo 1: Single provider with retry ---")
    messages = [
        Message.system("You are a concise assistant."),
        Message.user("Name three benefits of retry logic in distributed systems."),
    ]

    first_provider = None
    for spec in chain:
        first_provider = spec.factory()
        if first_provider is not None:
            break

    if first_provider is None:
        print("No providers available. Set ANTHROPIC_API_KEY or start Ollama.")
        return 1

    try:
        name, response = chat_with_retry(first_provider, messages, config, retry_cfg)
        print(f"\n  Provider: {name}")
        print(f"  Response: {response.content[:200]}")
        print(f"  Tokens:   {response.usage.total_tokens}")
    except LLMError as e:
        print(f"  Failed after retries: {e}")
    print()

    # --- Demonstrate full failover chain ---
    if (
        config.step_pause(
            "Demo 2: Full failover chain...",
            [
                "Tries each provider in sequence with per-provider retry",
                "First success wins; remaining providers are skipped",
            ],
        )
        == StepAction.QUIT
    ):
        return 0

    print("--- Demo 2: Full provider failover chain ---")
    messages = [
        Message.user("What is exponential backoff and why is it important?"),
    ]

    result = chat_with_fallback(chain, messages, config, retry_cfg)
    if result:
        name, response = result
        print(f"\n  Succeeded with: {name}")
        print(f"  Response: {response.content[:200]}")
        print(f"  Tokens:   {response.usage.total_tokens}")
    else:
        print("  All providers exhausted.")
        return 1

    print()
    print("Done.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
