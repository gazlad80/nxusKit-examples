#!/usr/bin/env python3
"""Example: Timeout Configuration - nxuskit

## nxusKit Features Demonstrated
- Provider creation with timeout, connect_timeout, and read_timeout
- Granular timeout control (connect vs. read phases)
- NetworkError on timeout with provider context
- Comparison between tight and relaxed timeouts

## Interactive Modes
- `--verbose` or `-v`: Show raw request/response data
- `--step` or `-s`: Pause at each step with explanations

## Why This Pattern Matters
Network requests can hang indefinitely without timeouts. nxusKit supports
separate connect and read timeouts so you can fail fast on connection issues
while allowing longer read windows for large completions.

Usage:
    export ANTHROPIC_API_KEY="your-key-here"
    python main.py
    python main.py --verbose    # Show timeout details
    python main.py --step       # Step through with explanations

Or with Ollama (no API key needed):
    python main.py ollama
"""

import os
import sys
import time
from typing import Optional

# Add shared python module to path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), "../../../shared/python"))

from interactive import InteractiveConfig, StepAction
from nxuskit import LLMError, Message, NetworkError, Provider


def create_provider_with_timeout(
    provider_name: str,
    *,
    timeout: float = 30.0,
    connect_timeout: Optional[float] = None,
    read_timeout: Optional[float] = None,
):
    """Create a provider with specific timeout settings.

    Returns (provider, description_string) or (None, error_message).
    """
    timeout_desc = f"timeout={timeout}s"
    if connect_timeout is not None or read_timeout is not None:
        timeout_desc = f"connect={connect_timeout}s, read={read_timeout}s"

    if provider_name == "claude":
        api_key = os.environ.get("ANTHROPIC_API_KEY")
        if not api_key:
            # Fall back to Ollama
            print("ANTHROPIC_API_KEY not set -- falling back to Ollama")
            return create_provider_with_timeout(
                "ollama",
                timeout=timeout,
                connect_timeout=connect_timeout,
                read_timeout=read_timeout,
            )
        return (
            Provider.claude(
                model="claude-haiku-4-5-20251001",
                api_key=api_key,
                timeout=timeout,
                connect_timeout=connect_timeout,
                read_timeout=read_timeout,
            ),
            timeout_desc,
        )

    if provider_name == "openai":
        api_key = os.environ.get("OPENAI_API_KEY")
        if not api_key:
            print("Error: OPENAI_API_KEY not set")
            return None, "missing key"
        return (
            Provider.openai(
                model="gpt-4o",
                api_key=api_key,
                timeout=timeout,
                connect_timeout=connect_timeout,
                read_timeout=read_timeout,
            ),
            timeout_desc,
        )

    if provider_name == "ollama":
        return (
            Provider.ollama(
                model="llama3",
                timeout=timeout,
                connect_timeout=connect_timeout,
                read_timeout=read_timeout,
            ),
            timeout_desc,
        )

    print(f"Unknown provider: {provider_name}. Supported: claude, openai, ollama")
    return None, "unknown"


def timed_chat(provider, messages, config: InteractiveConfig, label: str) -> bool:
    """Run a chat call, print timing info, and return True on success."""
    config.print_request(
        "POST",
        f"https://api.{provider.provider_name}.com/v1/chat",
        {"label": label, "messages": [m.__dict__ for m in messages]},
    )

    start = time.time()
    try:
        response = provider.chat(messages, temperature=0.5, max_tokens=150)
        elapsed_ms = int((time.time() - start) * 1000)

        config.print_response(200, elapsed_ms, {"content": response.content[:100]})

        print(f"  Completed in {elapsed_ms}ms")
        print(f"  Response: {response.content[:120]}...")
        print(f"  Tokens: {response.usage.total_tokens}")
        return True

    except NetworkError as e:
        elapsed_ms = int((time.time() - start) * 1000)
        print(f"  NetworkError after {elapsed_ms}ms: {e}")
        return False

    except LLMError as e:
        elapsed_ms = int((time.time() - start) * 1000)
        print(f"  LLMError after {elapsed_ms}ms ({e.provider}): {e}")
        return False


def main() -> int:
    config = InteractiveConfig.from_args()

    print("=== Timeout Configuration Example ===")
    print()

    args = [a for a in sys.argv[1:] if not a.startswith("-")]
    provider_name = args[0] if args else "claude"

    # --- Demo 1: Very short timeout (likely to fail on cloud providers) ---
    if (
        config.step_pause(
            "Demo 1: Very short timeout...",
            [
                "Sets a 0.001s read timeout -- almost certainly too short",
                "Demonstrates that nxusKit raises NetworkError on timeout",
                "Note: Ollama locally may actually succeed with this timeout",
            ],
        )
        == StepAction.QUIT
    ):
        return 0

    print("--- Demo 1: Very short timeout (expect failure) ---")
    short_provider, desc = create_provider_with_timeout(
        provider_name,
        connect_timeout=5.0,
        read_timeout=0.001,
    )
    if short_provider is None:
        return 1

    print(f"  Provider: {short_provider.provider_name}, {desc}")
    messages = [Message.user("Say hello in one sentence.")]

    success = timed_chat(short_provider, messages, config, "short-timeout")
    if success:
        print("  (Surprisingly succeeded with short timeout!)")
    else:
        print("  Expected: timeout triggered NetworkError")
    print()

    # --- Demo 2: Normal timeout (should succeed) ---
    if (
        config.step_pause(
            "Demo 2: Normal timeout...",
            [
                "Uses default 30s timeout -- plenty of headroom",
                "nxusKit: connect_timeout and read_timeout can be set independently",
            ],
        )
        == StepAction.QUIT
    ):
        return 0

    print("--- Demo 2: Normal timeout (expect success) ---")
    normal_provider, desc = create_provider_with_timeout(
        provider_name,
        connect_timeout=10.0,
        read_timeout=30.0,
    )
    if normal_provider is None:
        return 1

    print(f"  Provider: {normal_provider.provider_name}, {desc}")
    messages = [Message.user("Explain timeouts in networking in two sentences.")]

    success = timed_chat(normal_provider, messages, config, "normal-timeout")
    if not success:
        print("  Unexpected failure with normal timeout")
        return 1
    print()

    # --- Demo 3: Total timeout shorthand ---
    if (
        config.step_pause(
            "Demo 3: Total timeout shorthand...",
            [
                "nxusKit: 'timeout' sets both connect and read to the same value",
                "Simpler API when you don't need granular control",
            ],
        )
        == StepAction.QUIT
    ):
        return 0

    print("--- Demo 3: Single timeout parameter ---")
    unified_provider, desc = create_provider_with_timeout(provider_name, timeout=30.0)
    if unified_provider is None:
        return 1

    print(f"  Provider: {unified_provider.provider_name}, {desc}")
    messages = [Message.user("What is a good default HTTP timeout and why?")]

    success = timed_chat(unified_provider, messages, config, "unified-timeout")
    if not success:
        print("  Unexpected failure")
        return 1
    print()

    print("Done. Timeout configuration demonstrated.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
