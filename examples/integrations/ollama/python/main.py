#!/usr/bin/env python3
"""Example: Ollama Local Integration - nxuskit

## nxusKit Features Demonstrated
- Provider.ollama() for local model access (no API key required)
- Unified chat interface working with local models
- Server connectivity checks and graceful error handling
- Consistent response types (ChatResponse, TokenUsage) across local/cloud

## Interactive Modes
- `--verbose` or `-v`: Show raw request/response data
- `--step` or `-s`: Pause at each step with explanations

## Why This Pattern Matters
Ollama runs models locally, eliminating API costs and latency for development
and privacy-sensitive workloads. nxusKit wraps Ollama with the exact same
interface as cloud providers, so code written against Ollama works unchanged
with Claude or OpenAI.

Usage:
    # Start Ollama first:
    ollama serve

    # Then run:
    python main.py
    python main.py --verbose    # Show request/response details
    python main.py --step       # Step through with explanations

    # Use a specific model:
    python main.py llama3
    python main.py mistral
    python main.py phi3
"""

import os
import sys
import time
from typing import Optional

# Add shared python module to path (integration = one level deeper)
sys.path.insert(0, os.path.join(os.path.dirname(__file__), "../../../shared/python"))

from interactive import InteractiveConfig, StepAction
from nxuskit import LLMError, Message, NetworkError, Provider


OLLAMA_BASE_URL = "http://localhost:11434"


def check_ollama_server() -> bool:
    """Check if the Ollama server is reachable.

    Makes a lightweight HTTP request to the Ollama API root.
    """
    import urllib.request
    import urllib.error

    try:
        req = urllib.request.Request(OLLAMA_BASE_URL, method="GET")
        with urllib.request.urlopen(req, timeout=3) as resp:
            return resp.status == 200
    except (urllib.error.URLError, OSError):
        return False


def list_available_models() -> Optional[list]:
    """Fetch the list of locally available models from Ollama.

    Returns a list of model name strings, or None if the request fails.
    """
    import json
    import urllib.request
    import urllib.error

    try:
        req = urllib.request.Request(f"{OLLAMA_BASE_URL}/api/tags", method="GET")
        with urllib.request.urlopen(req, timeout=5) as resp:
            data = json.loads(resp.read().decode())
            models = data.get("models", [])
            return [m.get("name", "unknown") for m in models]
    except (urllib.error.URLError, OSError, json.JSONDecodeError):
        return None


def main() -> int:
    config = InteractiveConfig.from_args()

    print("=== Ollama Local Integration Example ===")
    print()

    # Parse optional model name from args
    args = [a for a in sys.argv[1:] if not a.startswith("-")]
    model_name = args[0] if args else "llama3"

    # Step: Check server connectivity
    if (
        config.step_pause(
            "Checking Ollama server connectivity...",
            [
                f"Ollama API endpoint: {OLLAMA_BASE_URL}",
                "No API key required -- Ollama runs locally",
                "Make sure 'ollama serve' is running",
            ],
        )
        == StepAction.QUIT
    ):
        return 0

    print(f"Checking Ollama server at {OLLAMA_BASE_URL}...")
    if not check_ollama_server():
        print()
        print("ERROR: Ollama server is not reachable.")
        print()
        print("To fix this:")
        print("  1. Install Ollama: https://ollama.ai")
        print("  2. Start the server: ollama serve")
        print(f"  3. Pull a model: ollama pull {model_name}")
        print("  4. Re-run this example")
        return 1

    print("  Server is running.")
    print()

    # Step: List available models
    if (
        config.step_pause(
            "Listing locally available models...",
            [
                "Ollama stores downloaded models locally",
                "Each model can be used without any API key",
            ],
        )
        == StepAction.QUIT
    ):
        return 0

    models = list_available_models()
    if models:
        print("Available models:")
        for m in models:
            marker = " <-- selected" if model_name in m else ""
            print(f"  - {m}{marker}")
    else:
        print("Could not list models (API may not support /api/tags).")
    print()

    # Step: Create provider
    if (
        config.step_pause(
            f"Creating Ollama provider with model '{model_name}'...",
            [
                "nxusKit: Provider.ollama() needs only a model name",
                "Same interface as Provider.claude() or Provider.openai()",
                "Timeout defaults to 30s (local models can be slow on first load)",
            ],
        )
        == StepAction.QUIT
    ):
        return 0

    # nxusKit: Provider.ollama() -- no API key, just a model name
    provider = Provider.ollama(model=model_name, timeout=60.0)

    print(f"Provider: {provider.provider_name}")
    print(f"Model:    {provider.model}")
    print()

    # --- Demo: Basic chat ---
    if (
        config.step_pause(
            "Sending a chat request to the local model...",
            [
                "nxusKit: provider.chat() works identically for local and cloud",
                "Response includes content, model name, and token usage",
            ],
        )
        == StepAction.QUIT
    ):
        return 0

    print("--- Basic Chat ---")
    messages = [
        Message.system("You are a helpful assistant running locally via Ollama."),
        Message.user("What are the advantages of running LLMs locally?"),
    ]

    config.print_request(
        "POST",
        f"{OLLAMA_BASE_URL}/api/chat",
        {"model": model_name, "messages": [m.__dict__ for m in messages]},
    )

    try:
        start = time.time()
        response = provider.chat(messages, temperature=0.7, max_tokens=300)
        elapsed_ms = int((time.time() - start) * 1000)

        config.print_response(
            200,
            elapsed_ms,
            {"content": response.content[:200], "model": response.model},
        )

        print(f"Response ({elapsed_ms}ms):")
        print(response.content)
        print()
        print(f"Model:  {response.model}")
        print(
            f"Tokens: {response.usage.input_tokens} in, "
            f"{response.usage.output_tokens} out, "
            f"{response.usage.total_tokens} total"
        )

    except NetworkError as e:
        print(f"Network error: {e}")
        print(f"Is the model '{model_name}' pulled? Run: ollama pull {model_name}")
        return 1
    except LLMError as e:
        print(f"Error ({e.provider}): {e}")
        return 1
    print()

    # --- Demo: Follow-up conversation ---
    if (
        config.step_pause(
            "Sending a follow-up message...",
            [
                "Multi-turn conversations work the same as with cloud providers",
                "Pass the full message history for context",
            ],
        )
        == StepAction.QUIT
    ):
        return 0

    print("--- Follow-up Conversation ---")
    messages.append(Message.assistant(response.content))
    messages.append(Message.user("Summarize that in one sentence."))

    try:
        start = time.time()
        followup = provider.chat(messages, temperature=0.5, max_tokens=100)
        elapsed_ms = int((time.time() - start) * 1000)

        print(f"Follow-up ({elapsed_ms}ms):")
        print(followup.content)
        print()
        print(f"Tokens: {followup.usage.total_tokens}")

    except LLMError as e:
        print(f"Error: {e}")
        return 1

    print()
    print("Done. Ollama integration working.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
