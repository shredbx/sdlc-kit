"""GET /account/credits: who may ask, what comes back, and what happens when the provider cannot be reached."""

import pytest
from agent_framework.core.auth import AuthUnavailable
from agent_framework.core.providers.credits import CreditsUnavailable, KeyUsage
from agent_framework.server import app as app_module
from agent_framework.server.app import create_app
from fastapi.testclient import TestClient

READING = KeyUsage(
    provider="openrouter", used=1.23, used_today=0.05, used_week=0.4, used_month=1.1, limit=20.0, remaining=18.77, as_of="2026-10-02T11:00:00+00:00"
)
STAFF = {"authorization": "Bearer staff"}


class FakeSource:
    def __init__(self, reading: KeyUsage | Exception = READING) -> None:
        self.reading, self.asked = reading, 0

    async def current(self) -> KeyUsage:
        self.asked += 1
        if isinstance(self.reading, Exception):
            raise self.reading
        return self.reading


async def verify(token: str) -> str | None:
    return "staff-1" if token == "staff" else None


@pytest.fixture(autouse=True)
def no_agents(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setattr(app_module, "discover", lambda _package: [])


def client(source: FakeSource | None = None, *, verifier: object = verify) -> TestClient:
    return TestClient(create_app("none", secret_key="k", user_verifier=verifier, credits=source, docs_enabled=False), raise_server_exceptions=False)  # type: ignore[arg-type]


class TestWhoMayAsk:
    @pytest.mark.parametrize("headers", [{}, {"authorization": "Bearer nope"}, {"authorization": "Basic staff"}, {"authorization": "Bearer "}])
    def test_only_a_signed_in_user_and_the_provider_is_not_asked_for_anyone_else(self, headers: dict) -> None:
        source = FakeSource()

        response = client(source).get("/account/credits", headers=headers)

        assert response.status_code == 401
        assert source.asked == 0

    def test_without_a_user_verifier_the_route_does_not_exist(self) -> None:
        assert client(FakeSource(), verifier=None).get("/account/credits", headers=STAFF).status_code == 404


class TestWhatComesBack:
    def test_the_reading_for_a_signed_in_user_and_nothing_that_can_be_cached_elsewhere(self) -> None:
        response = client(FakeSource()).get("/account/credits", headers=STAFF)

        assert response.status_code == 200
        assert response.headers["cache-control"] == "no-store"
        assert response.json() == {
            "supported": True,
            "provider": "openrouter",
            "currency": "USD",
            "used": 1.23,
            "used_today": 0.05,
            "used_week": 0.4,
            "used_month": 1.1,
            "limit": 20.0,
            "remaining": 18.77,
            "as_of": "2026-10-02T11:00:00+00:00",
            "stale": False,
        }

    def test_a_deployment_with_no_such_provider_says_unsupported_instead_of_failing(self) -> None:
        response = client(None).get("/account/credits", headers=STAFF)

        assert response.status_code == 200 and response.json() == {"supported": False}
        assert response.headers["cache-control"] == "no-store"

    def test_a_stale_reading_says_so(self) -> None:
        response = client(FakeSource(READING.model_copy(update={"stale": True}))).get("/account/credits", headers=STAFF)

        assert response.json()["stale"] is True and response.json()["used"] == 1.23

    def test_when_the_provider_cannot_be_asked_the_error_is_retryable_and_says_nothing_about_why(self) -> None:
        response = client(FakeSource(CreditsUnavailable("OpenRouter answered 401 with sk-or-secret"))).get("/account/credits", headers=STAFF)

        assert response.status_code == 503
        assert response.json() == {"detail": "The usage figures are not available right now.", "code": "credits_unavailable", "retryable": True}
        assert "sk-or" not in response.text and "401" not in response.text


class TestWhenTheLoginCheckCannotAnswer:
    def test_it_is_a_retryable_503_not_a_401_and_the_provider_is_not_asked(self) -> None:
        async def provider_down(token: str) -> str | None:
            raise AuthUnavailable("Supabase Auth answered 503")

        source = FakeSource()

        response = client(source, verifier=provider_down).get("/account/credits", headers=STAFF)

        assert response.status_code == 503
        assert response.headers["cache-control"] == "no-store"
        assert (response.json()["code"], response.json()["retryable"]) == ("auth_unavailable", True)
        assert source.asked == 0
