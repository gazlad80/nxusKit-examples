#!/usr/bin/env python3
"""Example: Streaming Chat - nxuskit

## nxusKit Features Demonstrated
- Unified streaming interface across all providers (Iterator[StreamChunk])
- Python iterator-based streaming (idiomatic Python pattern)
- Structured chunk types with delta content and metadata
- Streaming utilities (collect_stream, stream_with_callback)

## Interactive Modes
- `--verbose` or `-v`: Show raw SSE chunks as they arrive
- `--step` or `-s`: Pause at each step with explanations

## Why This Pattern Matters
Streaming enables real-time response display, reducing perceived latency.
nxusKit normalizes the different streaming formats from Claude, OpenAI,
and Ollama into a consistent iterator-based interface.

Usage:
    export ANTHROPIC_API_KEY="your-key-here"
    python main.py
    python main.py --verbose    # Show SSE chunks
    python main.py --step       # Step through with explanations

Or with Ollama (no API key needed):
    python main.py ollama
"""

import os
import sys
import time

# Add shared python module to path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), "../../../shared/python"))

from interactive import InteractiveConfig, StepAction
from nxuskit import LLMError, Message, Provider, stream_with_callback


def main():
    # Parse interactive mode flags
    config = InteractiveConfig.from_args()

    print("=== Streaming Chat Example ===")
    print()

    # Step: Checking API keys
    if (
        config.step_pause(
            "Checking for API keys...",
            [
                "Reads ANTHROPIC_API_KEY, OPENAI_API_KEY from environment",
                "Falls back to Ollama if no cloud API key is set",
            ],
        )
        == StepAction.QUIT
    ):
        return 0

    # Determine provider from command line (default: claude)
    args = [a for a in sys.argv[1:] if not a.startswith("-")]
    provider_name = args[0] if args else "claude"

    # nxusKit: Same provider factory as non-streaming
    provider, model = create_provider(provider_name)
    if provider is None:
        return 1

    # Step: Creating provider
    if (
        config.step_pause(
            f"Creating {provider_name} provider...",
            [
                "nxusKit: Same builder pattern as non-streaming",
                "Streaming is just a different method call on the provider",
            ],
        )
        == StepAction.QUIT
    ):
        return 0

    # Step: Building request
    if (
        config.step_pause(
            "Building streaming request...",
            [
                "nxusKit: Same message types work for streaming and non-streaming",
                "The provider handles stream: true parameter automatically",
            ],
        )
        == StepAction.QUIT
    ):
        return 0

    messages = [
        Message.user("Write a short poem about Python programming."),
    ]

    # Verbose: Show request
    config.print_request(
        "POST",
        get_provider_url(provider_name),
        {
            "messages": [{"role": "user", "content": messages[0].content}],
            "stream": True,
        },
    )

    # Step: Starting stream
    if (
        config.step_pause(
            "Starting streaming request...",
            [
                "nxusKit: Unified streaming API - returns Iterator[StreamChunk]",
                "Server-Sent Events (SSE) arrive as they're generated",
                "Python iteration provides natural backpressure",
            ],
        )
        == StepAction.QUIT
    ):
        return 0

    print(f"Streaming response from {provider.provider_name}:")
    print()

    try:
        # nxusKit: Unified streaming API - returns Iterator[StreamChunk]
        start = time.time()
        stream = provider.chat_stream(
            messages,
            temperature=0.8,
            max_tokens=300,
        )

        # nxusKit: Standard Python iteration - process chunks as they arrive
        full_content = []
        chunk_count = 0
        for chunk in stream:
            chunk_count += 1
            # nxusKit: Normalized chunk structure - delta contains new content
            if chunk.delta:
                # Verbose: Show each chunk
                config.print_stream_chunk(chunk_count, chunk.delta)

                print(chunk.delta, end="", flush=True)
                full_content.append(chunk.delta)

        elapsed_ms = int((time.time() - start) * 1000)
        print("\n")

        # Verbose: Show stream completion
        config.print_stream_done(elapsed_ms, chunk_count)

        # Note: Token usage in streaming requires collecting the final response
        # or using stream_with_callback for accumulated usage

    except LLMError as e:
        print(f"\nError ({e.provider}): {e}")
        return 1

    # Demonstrate stream_with_callback utility
    print("--- Using stream_with_callback utility ---")
    print()

    def on_chunk(chunk):
        """nxusKit: Callback receives each StreamChunk"""
        if chunk.delta:
            print(chunk.delta, end="", flush=True)

    try:
        # nxusKit: stream_with_callback provides accumulated content
        result = stream_with_callback(
            provider.chat_stream(
                [Message.user("Count from 1 to 5 briefly.")],
                max_tokens=100,
            ),
            on_chunk,
        )
        print(f"\n\nCollected content: {result[:50]}...")

    except LLMError as e:
        print(f"\nError: {e}")
        return 1

    return 0


def create_provider(provider_name: str):
    """Create a provider based on available configuration."""
    if provider_name == "claude":
        api_key = os.environ.get("ANTHROPIC_API_KEY")
        if not api_key:
            print("Error: ANTHROPIC_API_KEY not set")
            print("\nTo fix this, either:")
            print("  1. export ANTHROPIC_API_KEY='your-key-here'")
            print("  2. Run: python main.py ollama")
            return None, None
        return Provider.claude(
            model="claude-haiku-4-5-20251001", api_key=api_key
        ), "claude-haiku-4-5-20251001"

    elif provider_name == "openai":
        api_key = os.environ.get("OPENAI_API_KEY")
        if not api_key:
            print("Error: OPENAI_API_KEY not set")
            return None, None
        return Provider.openai(model="gpt-4o", api_key=api_key), "gpt-4o"

    elif provider_name == "ollama":
        return Provider.ollama(model="llama3"), "llama3"

    else:
        print(f"Unknown provider: {provider_name}")
        print("Supported: claude, openai, ollama")
        return None, None


def get_provider_url(provider_name: str) -> str:
    """Get the API URL for verbose output based on provider name."""
    urls = {
        "claude": "https://api.anthropic.com/v1/messages",
        "openai": "https://api.openai.com/v1/chat/completions",
        "ollama": "http://localhost:11434/api/chat",
    }
    return urls.get(provider_name, "https://api.example.com/chat")


if __name__ == "__main__":
    sys.exit(main())
