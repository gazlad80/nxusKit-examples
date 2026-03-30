"""Unit tests for interactive module."""

from interactive import (
    InteractiveConfig,
    StepAction,
    _get_status_text,
    _summarize_base64,
)


class TestInteractiveConfig:
    """Tests for InteractiveConfig class."""

    def test_default_config(self):
        """Default config should have verbose and step disabled."""
        config = InteractiveConfig()
        assert not config.is_verbose()
        # Step mode depends on TTY, may be false
        assert config.get_verbose_limit() == 2000

    def test_verbose_enabled(self):
        """Verbose mode should be enabled when set."""
        config = InteractiveConfig(verbose=True)
        assert config.is_verbose()

    def test_verbose_limit(self):
        """Verbose limit should be configurable."""
        config = InteractiveConfig(verbose_limit=5000)
        assert config.get_verbose_limit() == 5000

    def test_skip_steps(self):
        """Skip steps should disable step mode."""
        config = InteractiveConfig(step=True, _is_tty=True)
        config._step_skipped = False  # Reset for test

        # Force step to be enabled for this test
        config.step = True
        assert config.is_step()

        config.skip_steps()
        assert not config.is_step()


class TestStepAction:
    """Tests for StepAction enum."""

    def test_action_values(self):
        """StepAction should have correct values."""
        assert StepAction.CONTINUE.value == "continue"
        assert StepAction.QUIT.value == "quit"
        assert StepAction.SKIP.value == "skip"

    def test_action_comparison(self):
        """StepAction values should be comparable."""
        assert StepAction.CONTINUE == StepAction.CONTINUE
        assert StepAction.CONTINUE != StepAction.QUIT
        assert StepAction.QUIT != StepAction.SKIP


class TestVerboseOutput:
    """Tests for verbose output methods."""

    def test_print_request_does_nothing_when_disabled(self):
        """print_request should do nothing when verbose is disabled."""
        config = InteractiveConfig(verbose=False)
        # Should not raise or print anything
        config.print_request("GET", "http://test", {"key": "value"})

    def test_print_response_does_nothing_when_disabled(self):
        """print_response should do nothing when verbose is disabled."""
        config = InteractiveConfig(verbose=False)
        # Should not raise or print anything
        config.print_response(200, 100, {"result": "ok"})

    def test_print_stream_chunk_does_nothing_when_disabled(self):
        """print_stream_chunk should do nothing when verbose is disabled."""
        config = InteractiveConfig(verbose=False)
        # Should not raise or print anything
        config.print_stream_chunk(1, "data")

    def test_print_stream_done_does_nothing_when_disabled(self):
        """print_stream_done should do nothing when verbose is disabled."""
        config = InteractiveConfig(verbose=False)
        # Should not raise or print anything
        config.print_stream_done(100, 5)


class TestHelperFunctions:
    """Tests for helper functions."""

    def test_get_status_text(self):
        """Status text should be correct for known codes."""
        assert _get_status_text(200) == "OK"
        assert _get_status_text(201) == "Created"
        assert _get_status_text(400) == "Bad Request"
        assert _get_status_text(401) == "Unauthorized"
        assert _get_status_text(403) == "Forbidden"
        assert _get_status_text(404) == "Not Found"
        assert _get_status_text(429) == "Too Many Requests"
        assert _get_status_text(500) == "Internal Server Error"
        assert _get_status_text(999) == ""

    def test_summarize_base64_short_string(self):
        """Short strings should not be summarized."""
        result = _summarize_base64('{"data": "short"}')
        assert result == '{"data": "short"}'

    def test_summarize_base64_long_string(self):
        """Long base64 strings should be summarized."""
        # Create a base64-like string > 1000 chars
        long_base64 = "A" * 1500
        input_json = f'{{"data": "{long_base64}"}}'
        result = _summarize_base64(input_json)
        assert "[base64:" in result
        assert "KB data]" in result


class TestStepPause:
    """Tests for step_pause method."""

    def test_step_pause_returns_continue_when_disabled(self):
        """step_pause should return CONTINUE when step mode is disabled."""
        config = InteractiveConfig(step=False)
        result = config.step_pause("Test step", ["Explanation"])
        assert result == StepAction.CONTINUE

    def test_step_pause_returns_continue_when_skipped(self):
        """step_pause should return CONTINUE when steps are skipped."""
        config = InteractiveConfig(step=True, _is_tty=True)
        config.step = True  # Force enable for test
        config.skip_steps()
        result = config.step_pause("Test step", ["Explanation"])
        assert result == StepAction.CONTINUE
