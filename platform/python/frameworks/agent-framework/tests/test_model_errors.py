"""The error contract is a table: every exception a run can raise maps to one (status, code,
retryable) - or None when it is not a model/network failure and must stay a plain 500. Pure, no
model, no network."""

import httpx
import pytest
from agent_framework.server.model_errors import classify_model_error
from pydantic_ai.exceptions import (
    ContentFilterError,
    FallbackExceptionGroup,
    ModelAPIError,
    ModelHTTPError,
    UnexpectedModelBehavior,
)


def _http(status: int, headers: dict[str, str] | None = None) -> ModelHTTPError:
    return ModelHTTPError(status, "some-model", body="provider body with request details", headers=headers)


@pytest.mark.parametrize(
    ("exc", "status", "code", "retryable"),
    [
        (_http(429), 503, "unavailable", True),
        (_http(500), 503, "unavailable", True),
        (_http(503), 503, "unavailable", True),
        (_http(408), 504, "timeout", True),
        (_http(504), 504, "timeout", True),
        (_http(400), 502, "rejected", False),
        (_http(401), 502, "rejected", False),
        (_http(403), 502, "rejected", False),
        (_http(404), 502, "rejected", False),
        (ModelAPIError("some-model", "Connection error."), 503, "unavailable", True),
        (httpx.ConnectError("refused"), 503, "unavailable", True),
        (httpx.ReadTimeout("slow"), 504, "timeout", True),
        (TimeoutError(), 504, "timeout", True),
        (ContentFilterError("blocked"), 422, "content_filtered", False),
        (UnexpectedModelBehavior("Exceeded maximum output retries (1)"), 502, "bad_output", True),
    ],
)
def test_model_and_network_failures_map_to_a_code(exc: BaseException, status: int, code: str, retryable: bool) -> None:
    failure = classify_model_error(exc)

    assert failure is not None
    assert (failure.status, failure.code, failure.retryable) == (status, code, retryable)


def test_a_provider_sdk_timeout_wrapped_as_model_api_error_is_a_timeout() -> None:
    wrapped = ModelAPIError("some-model", "Request timed out.")
    wrapped.__cause__ = httpx.ReadTimeout("slow")

    failure = classify_model_error(wrapped)

    assert failure is not None and failure.code == "timeout"


@pytest.mark.parametrize("exc", [ValueError("bug in a tool"), KeyError("x"), RuntimeError("boom")])
def test_anything_else_is_not_classified_and_stays_a_500(exc: BaseException) -> None:
    assert classify_model_error(exc) is None


def test_an_httpx_5xx_from_a_tool_is_unavailable_but_a_4xx_is_a_bug() -> None:
    request = httpx.Request("GET", "https://example.test")
    five_hundred = httpx.HTTPStatusError("bad", request=request, response=httpx.Response(502, request=request))
    four_hundred = httpx.HTTPStatusError("bad", request=request, response=httpx.Response(404, request=request))

    assert getattr(classify_model_error(five_hundred), "code", None) == "unavailable"
    assert classify_model_error(four_hundred) is None


def test_retry_after_is_read_from_the_provider_header_and_clamped() -> None:
    assert classify_model_error(_http(503, {"Retry-After": "7"})).retry_after == 7  # type: ignore[union-attr]
    assert classify_model_error(_http(503, {"Retry-After": "9999"})).retry_after == 60  # type: ignore[union-attr]
    assert classify_model_error(_http(503)).retry_after is None  # type: ignore[union-attr]


def test_the_providers_body_is_never_in_the_client_message() -> None:
    failure = classify_model_error(_http(401))

    assert failure is not None
    assert "request details" not in failure.message


def test_a_fallback_group_prefers_a_retryable_member() -> None:
    group = FallbackExceptionGroup("all models failed", [_http(401), _http(503)])

    failure = classify_model_error(group)

    assert failure is not None and (failure.code, failure.retryable) == ("unavailable", True)


def test_a_fallback_group_of_only_rejections_is_a_rejection() -> None:
    group = FallbackExceptionGroup("all models failed", [_http(401), _http(403)])

    failure = classify_model_error(group)

    assert failure is not None and (failure.code, failure.retryable) == ("rejected", False)


def test_a_group_with_no_model_failures_is_not_classified() -> None:
    assert classify_model_error(ExceptionGroup("unrelated", [ValueError("a"), KeyError("b")])) is None
