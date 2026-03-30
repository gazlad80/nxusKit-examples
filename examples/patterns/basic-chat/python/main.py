#!/usr/bin/env python3
"""Example: Basic Chat - nxuskit

## nxusKit Features Demonstrated
- Unified provider factory (Provider.claude, Provider.openai, Provider.ollama)
- Type-safe message construction with Message.user(), Message.system()
- Consistent error handling with typed exceptions
- Normalized token tracking across all providers (TokenUsage)

## Interactive Modes
- `--verbose` or `-v`: Show raw request/response data
- `--step` or `-s`: Pause at each step with explanations

## Why This Pattern Matters
This is the foundational pattern for all LLM interactions. nxusKit provides
a consistent API across providers (Claude, OpenAI, Ollama) so you can switch
providers without changing your application code.

Usage:
    export ANTHROPIC_API_KEY="your-key-here"
    python main.py
    python main.py --verbose    # Show request/response details
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
from nxuskit import LLMError, Message, Provider


def main():
    # Parse interactive mode flags
    config = InteractiveConfig.from_args()

    print("=== Basic Chat Example ===")
    print()

    # Step: Checking API keys
    if (
        config.step_pause(
            "Checking for API keys...",
            [
                "Reads ANTHROPIC_API_KEY, OPENAI_API_KEY from environment",
                "Falls back to Ollama if no cloud API key is set",
                "This keeps secrets out of source code",
            ],
        )
        == StepAction.QUIT
    ):
        return 0

    # Determine provider from command line (default: claude)
    # Filter out our flags from sys.argv
    args = [a for a in sys.argv[1:] if not a.startswith("-")]
    provider_name = args[0] if args else "claude"

    # nxusKit: Provider factory creates provider with sensible defaults
    provider, model = create_provider(provider_name)
    if provider is None:
        return 1

    # Step: Creating provider
    if (
        config.step_pause(
            f"Creating {provider_name} provider...",
            [
                "nxusKit: Provider factory with type-safe configuration",
                "The provider abstraction hides provider-specific details",
                "No HTTP connection is made yet - that happens on first request",
            ],
        )
        == StepAction.QUIT
    ):
        return 0

    # Step: Building request
    if (
        config.step_pause(
            "Building chat request...",
            [
                "nxusKit: Type-safe message construction with static methods",
                "Messages support system, user, and assistant roles",
                "Parameters like temperature and max_tokens have sensible defaults",
            ],
        )
        == StepAction.QUIT
    ):
        return 0

    # nxusKit: Type-safe message construction with static methods
    messages = [
        Message.system("You are a helpful programming assistant."),
        Message.user("What is Python and why should I use it?"),
    ]

    # Verbose: Show the request
    config.print_request(
        "POST",
        get_provider_url(provider_name),
        {
            "messages": [m.__dict__ for m in messages],
            "temperature": 0.7,
            "max_tokens": 500,
        },
    )

    # Step: Sending request
    if (
        config.step_pause(
            f"Sending request to {provider.provider_name} API...",
            [
                "nxusKit: Unified chat interface - same pattern for all providers",
                "The request is serialized to JSON and sent via HTTPS",
                "Response is parsed and normalized to a common format",
            ],
        )
        == StepAction.QUIT
    ):
        return 0

    print(f"Sending request to {provider.provider_name}...")
    print()

    try:
        # nxusKit: Unified chat interface - same pattern for all providers
        start = time.time()
        response = provider.chat(
            messages,
            temperature=0.7,
            max_tokens=500,
        )
        elapsed_ms = int((time.time() - start) * 1000)

        # Verbose: Show the response
        config.print_response(
            200,
            elapsed_ms,
            {
                "content": response.content,
                "model": response.model,
                "usage": response.usage.__dict__,
            },
        )

        # Display response
        print(f"Response:\n{response.content}\n")
        print(f"Model: {response.model}")

        # nxusKit: Unified token tracking - same format regardless of provider
        print("Token usage:")
        print(f"  Input: {response.usage.input_tokens} tokens")
        print(f"  Output: {response.usage.output_tokens} tokens")
        print(f"  Total: {response.usage.total_tokens} tokens")

    except LLMError as e:
        # nxusKit: Typed exceptions with provider context
        print(f"Error ({e.provider}): {e}")
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
        # nxusKit: Provider factory with type-safe configuration
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
        # nxusKit: Local provider - no API key required
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
