"""Model-call retries, proven against a fake HTTP endpoint (httpx2.MockTransport): no network, no key, no real model.
Each test counts how many times the fake server was hit. The waits are shortened so the suite stays fast."""

import json

import httpx2
import pytest
from agent_framework.core.providers import google as google_provider
from agent_framework.core.providers import openrouter as openrouter_provider
from agent_framework.core.providers.retry import DEFAULT_MAX_RETRIES, MAX_RETRIES_ALLOWED, check_max_retries
from pydantic_ai import Agent
from pydantic_ai.exceptions import ModelHTTPError


@pytest.fixture(autouse=True)
def _keys(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setenv("OPENROUTER_API_KEY", "test-key")
    monkeypatch.setenv("GOOGLE_API_KEY", "test-key")


@pytest.fixture(autouse=True)
def _no_real_sleep(monkeypatch: pytest.MonkeyPatch) -> None:
    """The OpenAI SDK waits with anyio.sleep between tries (0.5 s, 1 s, ...): skip the waiting, keep the tries."""

    async def instant(_seconds: float) -> None:
        return None

    monkeypatch.setattr("anyio.sleep", instant)


class FakeServer:
    """Answers each request with the next scripted response and remembers what it was asked."""

    def __init__(self, *responses: httpx2.Response) -> None:
        self._responses = list(responses)
        self.requests: list[httpx2.Request] = []

    def __call__(self, request: httpx2.Request) -> httpx2.Response:
        self.requests.append(request)
        return self._responses.pop(0) if len(self._responses) > 1 else self._responses[0]

    @property
    def client(self) -> httpx2.AsyncClient:
        return httpx2.AsyncClient(transport=httpx2.MockTransport(self))

    @property
    def calls(self) -> int:
        return len(self.requests)


# ---- fake OpenRouter (the OpenAI chat-completions protocol) -----------------------------------------------------


def _chat_ok() -> httpx2.Response:
    body = {
        "id": "c1",
        "object": "chat.completion",
        "created": 1,
        "model": "m",
        "provider": "FakeProvider",  # OpenRouter's own addition to the OpenAI shape
        "choices": [{"index": 0, "message": {"role": "assistant", "content": "all good"}, "finish_reason": "stop"}],
        "usage": {"prompt_tokens": 3, "completion_tokens": 2, "total_tokens": 5},
    }
    return httpx2.Response(200, json=body)


def _chat_stream_ok() -> httpx2.Response:
    def chunk(delta: dict, finish: str | None = None) -> str:
        data = {
            "id": "c1",
            "object": "chat.completion.chunk",
            "created": 1,
            "model": "m",
            "provider": "FakeProvider",
            "choices": [{"index": 0, "delta": delta, "finish_reason": finish}],
        }
        return f"data: {json.dumps(data)}\n\n"

    body = chunk({"role": "assistant", "content": "all "}) + chunk({"content": "good"}) + chunk({}, "stop") + "data: [DONE]\n\n"
    return httpx2.Response(200, content=body.encode(), headers={"content-type": "text/event-stream"})


def _error(status: int, **headers: str) -> httpx2.Response:
    return httpx2.Response(status, json={"error": {"message": f"fake {status}", "code": status}}, headers=headers)


async def _run_openrouter(server: FakeServer, max_retries: int) -> str:
    agent = Agent(openrouter_provider.resolve("some/model", max_retries, http_client=server.client))
    return (await agent.run("hi")).output


async def test_openrouter_retries_a_503_until_it_works() -> None:
    server = FakeServer(_error(503), _error(503), _chat_ok())

    assert await _run_openrouter(server, max_retries=2) == "all good"
    assert server.calls == 3


async def test_openrouter_stops_after_the_configured_number_of_retries() -> None:
    server = FakeServer(_error(503))

    with pytest.raises(ModelHTTPError) as caught:
        await _run_openrouter(server, max_retries=2)

    assert caught.value.status_code == 503
    assert server.calls == 3  # the first try and two retries


async def test_openrouter_retries_can_be_switched_off() -> None:
    server = FakeServer(_error(503), _chat_ok())

    with pytest.raises(ModelHTTPError):
        await _run_openrouter(server, max_retries=0)

    assert server.calls == 1


@pytest.mark.parametrize("status", [400, 401, 403, 404, 422])
async def test_openrouter_never_retries_a_request_the_provider_rejected(status: int) -> None:
    server = FakeServer(_error(status), _chat_ok())

    with pytest.raises(ModelHTTPError) as caught:
        await _run_openrouter(server, max_retries=2)

    assert caught.value.status_code == status
    assert server.calls == 1


async def test_openrouter_retries_a_rate_limit_that_names_a_short_wait() -> None:
    server = FakeServer(_error(429, **{"retry-after": "2"}), _chat_ok())

    assert await _run_openrouter(server, max_retries=2) == "all good"
    assert server.calls == 2


async def test_openrouter_retries_a_dropped_connection() -> None:
    attempts = 0

    def handler(request: httpx2.Request) -> httpx2.Response:
        nonlocal attempts
        attempts += 1
        if attempts == 1:
            raise httpx2.ConnectError("connection reset", request=request)
        return _chat_ok()

    agent = Agent(openrouter_provider.resolve("some/model", 2, http_client=httpx2.AsyncClient(transport=httpx2.MockTransport(handler))))

    assert (await agent.run("hi")).output == "all good"
    assert attempts == 2


async def test_openrouter_streaming_retries_before_the_first_byte_so_no_text_is_repeated() -> None:
    server = FakeServer(_error(503), _chat_stream_ok())
    agent = Agent(openrouter_provider.resolve("some/model", 2, http_client=server.client))

    async with agent.run_stream("hi") as streamed:
        chunks = [text async for text in streamed.stream_text(delta=True)]

    assert "".join(chunks) == "all good"
    assert server.calls == 2


# ---- fake Google (Gemini generateContent) -----------------------------------------------------------------------


def _gemini_body() -> dict:
    return {
        "candidates": [{"content": {"role": "model", "parts": [{"text": "all good"}]}, "finishReason": "STOP", "index": 0}],
        "usageMetadata": {"promptTokenCount": 3, "candidatesTokenCount": 2, "totalTokenCount": 5},
        "modelVersion": "gemini-test",
    }


def _gemini_ok() -> httpx2.Response:
    return httpx2.Response(200, json=_gemini_body())


def _gemini_stream_ok() -> httpx2.Response:
    return httpx2.Response(200, content=f"data: {json.dumps(_gemini_body())}\n\n".encode(), headers={"content-type": "text/event-stream"})


def _gemini_error(status: int, name: str = "UNAVAILABLE") -> httpx2.Response:
    return httpx2.Response(status, json={"error": {"code": status, "message": f"fake {status}", "status": name}})


def _gemini(server: FakeServer, max_retries: int) -> Agent:
    # Same shape as production, with the waits cut to a few milliseconds.
    model = google_provider.resolve("gemini-test", max_retries, http_client=server.client, initial_delay=0.001, max_delay=0.002, jitter=0.001)
    return Agent(model)


async def test_google_retries_a_503_until_it_works() -> None:
    server = FakeServer(_gemini_error(503), _gemini_error(503), _gemini_ok())

    assert (await _gemini(server, 2).run("hi")).output == "all good"
    assert server.calls == 3


async def test_google_stops_after_the_configured_number_of_retries() -> None:
    server = FakeServer(_gemini_error(503))

    with pytest.raises(ModelHTTPError) as caught:
        await _gemini(server, 2).run("hi")

    assert caught.value.status_code == 503
    assert server.calls == 3


async def test_google_retries_can_be_switched_off() -> None:
    server = FakeServer(_gemini_error(503), _gemini_ok())

    with pytest.raises(ModelHTTPError):
        await _gemini(server, 0).run("hi")

    assert server.calls == 1


@pytest.mark.parametrize(("status", "name"), [(400, "INVALID_ARGUMENT"), (401, "UNAUTHENTICATED"), (403, "PERMISSION_DENIED"), (404, "NOT_FOUND")])
async def test_google_never_retries_a_request_the_provider_rejected(status: int, name: str) -> None:
    server = FakeServer(_gemini_error(status, name), _gemini_ok())

    with pytest.raises(ModelHTTPError) as caught:
        await _gemini(server, 2).run("hi")

    assert caught.value.status_code == status
    assert server.calls == 1


@pytest.mark.parametrize("status", [408, 429, 500, 502, 504])
async def test_google_retries_every_listed_transient_status(status: int) -> None:
    server = FakeServer(_gemini_error(status, "TRANSIENT"), _gemini_ok())

    assert (await _gemini(server, 2).run("hi")).output == "all good"
    assert server.calls == 2


async def test_google_streaming_retries_before_the_first_byte() -> None:
    server = FakeServer(_gemini_error(503), _gemini_stream_ok())

    async with _gemini(server, 2).run_stream("hi") as streamed:
        chunks = [text async for text in streamed.stream_text(delta=True)]

    assert "".join(chunks) == "all good"
    assert server.calls == 2


# ---- the setting itself -----------------------------------------------------------------------------------------


def test_the_default_is_two_retries_and_the_range_is_zero_to_five() -> None:
    assert DEFAULT_MAX_RETRIES == 2
    assert [check_max_retries(n) for n in range(MAX_RETRIES_ALLOWED + 1)] == [0, 1, 2, 3, 4, 5]


@pytest.mark.parametrize("bad", [-1, MAX_RETRIES_ALLOWED + 1, 2.5, "2", True, None])
def test_a_retry_count_outside_the_range_is_refused(bad: object) -> None:
    with pytest.raises(ValueError, match="max_retries"):
        check_max_retries(bad)  # type: ignore[arg-type]


def test_google_options_count_the_first_try_and_list_only_transient_statuses() -> None:
    options = google_provider.retry_options(2)

    assert options is not None
    assert options.attempts == 3
    assert options.http_status_codes == [408, 429, 500, 502, 503, 504]
    assert (options.initial_delay, options.max_delay) == (1.0, 8.0)
    assert google_provider.retry_options(0) is None
