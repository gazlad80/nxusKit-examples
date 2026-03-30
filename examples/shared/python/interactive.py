"""
Interactive mode utilities for nxusKit examples.

Provides two debugging modes:
- Verbose mode (--verbose or -v): Shows raw HTTP request/response data
- Step mode (--step or -s): Pauses at each API call with explanations

Usage:
    from interactive import InteractiveConfig, StepAction

    config = InteractiveConfig.from_args()

    # Step mode: pause with explanation
    if config.step_pause("Creating provider...", [
        "This initializes the HTTP client",
        "API key is validated on first request",
    ]) == StepAction.QUIT:
        return

    # Verbose mode: show request
    config.print_request("POST", "https://api.example.com/chat", request_dict)

    # ... make request ...

    # Verbose mode: show response
    config.print_response(200, elapsed_ms, response_dict)

Environment Variables:
    NXUSKIT_VERBOSE=1: Enable verbose mode (alternative to --verbose)
    NXUSKIT_STEP=1: Enable step mode (alternative to --step)
    NXUSKIT_VERBOSE_LIMIT=N: Max characters before truncation (default: 2000)
"""

import argparse
import json
import os
import re
import sys
from dataclasses import dataclass, field
from enum import Enum
from typing import Any


class StepAction(Enum):
    """Result of a step pause, indicating user's choice."""

    CONTINUE = "continue"  # User pressed Enter, proceed to next step
    QUIT = "quit"  # User typed 'q', exit program gracefully
    SKIP = "skip"  # User typed 's', disable step mode and continue


def _is_tty() -> bool:
    """Check if stdin is a terminal (TTY)."""
    return sys.stdin.isatty()


@dataclass
class InteractiveConfig:
    """Configuration for interactive debugging modes.

    Created once at program start from CLI args and environment variables.
    """

    verbose: bool = False
    step: bool = False
    verbose_limit: int = 2000
    _is_tty: bool = field(default_factory=_is_tty)
    _step_skipped: bool = field(default=False, repr=False)

    @classmethod
    def from_args(cls) -> "InteractiveConfig":
        """Parse configuration from CLI args and environment variables.

        CLI flags take precedence over environment variables.
        """
        if "--help" in sys.argv or "-h" in sys.argv:
            mod = sys.modules.get("__main__")
            doc = (getattr(mod, "__doc__", None) or "").strip()
            if doc:
                print(doc)
            else:
                print("Usage: see the example source file (no module docstring).")
            sys.exit(0)

        parser = argparse.ArgumentParser(add_help=False)
        parser.add_argument(
            "--verbose", "-v", action="store_true", help="Enable verbose output"
        )
        parser.add_argument(
            "--step", "-s", action="store_true", help="Enable step-through mode"
        )

        # Parse known args only (don't fail on example-specific args)
        args, _ = parser.parse_known_args()

        # Check environment variables as fallback
        verbose = args.verbose or os.environ.get("NXUSKIT_VERBOSE") == "1"
        step = args.step or os.environ.get("NXUSKIT_STEP") == "1"

        # Parse verbose limit from environment
        verbose_limit = 2000
        limit_str = os.environ.get("NXUSKIT_VERBOSE_LIMIT")
        if limit_str:
            try:
                verbose_limit = max(100, min(100000, int(limit_str)))
            except ValueError:
                pass

        is_tty = _is_tty()

        # Warn if step mode requested in non-TTY environment
        if step and not is_tty:
            print(
                "[nxusKit] Warning: Step mode disabled (not a TTY). Use --verbose for debugging.",
                file=sys.stderr,
            )

        return cls(
            verbose=verbose,
            step=step and is_tty,  # Auto-disable step mode in non-TTY
            verbose_limit=verbose_limit,
            _is_tty=is_tty,
            _step_skipped=False,
        )

    def is_verbose(self) -> bool:
        """Check if verbose mode is enabled."""
        return self.verbose

    def is_step(self) -> bool:
        """Check if step mode is enabled and not skipped."""
        return self.step and not self._step_skipped

    def is_tty(self) -> bool:
        """Check if running in a TTY."""
        return self._is_tty

    def get_verbose_limit(self) -> int:
        """Get the verbose output truncation limit."""
        return self.verbose_limit

    def skip_steps(self) -> None:
        """Mark step mode as skipped (user pressed 's')."""
        self._step_skipped = True

    def print_request(self, method: str, url: str, body: Any) -> None:
        """Print a request in verbose format.

        Displays the HTTP method, URL, and JSON body with pretty-printing.
        Binary data (base64) is summarized rather than shown in full.
        """
        if not self.verbose:
            return

        print(f"\n[nxusKit REQUEST] {method} {url}", file=sys.stderr)

        try:
            json_str = json.dumps(body, indent=2, default=str)
            processed = self._process_json_for_display(json_str)
            print(processed, file=sys.stderr)
        except (TypeError, ValueError) as e:
            print(f"  (Could not serialize request: {e})", file=sys.stderr)

    def print_response(self, status: int, elapsed_ms: int, body: Any) -> None:
        """Print a response in verbose format.

        Displays the status code, elapsed time, and JSON body.
        """
        if not self.verbose:
            return

        status_text = _get_status_text(status)
        print(
            f"\n[nxusKit RESPONSE] {status} {status_text} ({elapsed_ms}ms)",
            file=sys.stderr,
        )

        try:
            json_str = json.dumps(body, indent=2, default=str)
            processed = self._process_json_for_display(json_str)
            print(processed, file=sys.stderr)
        except (TypeError, ValueError) as e:
            print(f"  (Could not serialize response: {e})", file=sys.stderr)

    def print_stream_chunk(self, chunk_num: int, data: str) -> None:
        """Print a streaming chunk.

        Displays chunk number and raw data for SSE debugging.
        """
        if not self.verbose:
            return

        # Truncate long chunks
        if len(data) > 200:
            display_data = f"{data[:200]}... [truncated, {len(data)} chars]"
        else:
            display_data = data

        print(f"[nxusKit STREAM] chunk {chunk_num}: {display_data}", file=sys.stderr)

    def print_stream_done(self, elapsed_ms: int, total_chunks: int) -> None:
        """Print stream completion summary."""
        if not self.verbose:
            return

        print(
            f"[nxusKit STREAM] done ({elapsed_ms}ms, {total_chunks} chunks)",
            file=sys.stderr,
        )

    def step_pause(self, title: str, explanation: list[str]) -> StepAction:
        """Pause for step-through mode with explanation.

        Displays a title and explanation bullet points, then waits for user input.
        Returns the action the user chose.

        Args:
            title: Brief description of the step (e.g., "Creating Claude provider...")
            explanation: Bullet points explaining what will happen

        Returns:
            StepAction.CONTINUE: User pressed Enter
            StepAction.QUIT: User typed 'q'
            StepAction.SKIP: User typed 's'
        """
        if not self.is_step():
            return StepAction.CONTINUE

        # Print step header
        print(f"\n[nxusKit STEP] {title}", file=sys.stderr)

        # Print explanation bullets
        for line in explanation:
            print(f"  - {line}", file=sys.stderr)

        # Print prompt
        print(
            "[Press Enter to continue, 'q' to quit, 's' to skip steps]", file=sys.stderr
        )

        # Read user input
        try:
            user_input = input().strip().lower()
        except EOFError:
            return StepAction.CONTINUE

        if user_input in ("q", "quit"):
            return StepAction.QUIT
        elif user_input in ("s", "skip"):
            self.skip_steps()
            print(
                "[nxusKit] Step mode disabled, running to completion...",
                file=sys.stderr,
            )
            return StepAction.SKIP
        else:
            return StepAction.CONTINUE

    def _process_json_for_display(self, json_str: str) -> str:
        """Process JSON for display, handling truncation and base64 summarization."""
        # Check for base64 data and summarize it
        processed = _summarize_base64(json_str)

        # Truncate if too long
        if len(processed) > self.verbose_limit:
            return f"{processed[: self.verbose_limit]}... [truncated, {len(processed)} chars total]"
        return processed


def _summarize_base64(json_str: str) -> str:
    """Summarize base64 data in JSON strings.

    Replaces long base64 strings with a summary like "[base64: 45.2KB data]".
    """
    # Pattern: strings longer than 1000 chars that are mostly alphanumeric with +/=
    pattern = r'"([A-Za-z0-9+/=]{1000,})"'

    def replacer(match: re.Match[str]) -> str:
        content = match.group(1)
        kb = len(content) / 1024.0
        return f'"[base64: {kb:.1f}KB data]"'

    return re.sub(pattern, replacer, json_str)


def _get_status_text(status: int) -> str:
    """Get a human-readable status text for common HTTP codes."""
    status_map = {
        200: "OK",
        201: "Created",
        400: "Bad Request",
        401: "Unauthorized",
        403: "Forbidden",
        404: "Not Found",
        429: "Too Many Requests",
        500: "Internal Server Error",
    }
    return status_map.get(status, "")
