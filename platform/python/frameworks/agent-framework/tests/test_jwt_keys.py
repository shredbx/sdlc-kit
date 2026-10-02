"""Checking an access token on this server (api/jwt_keys.py) and the Supabase verifier built on it (api/supabase.py): the cases that must be valid, every way
a token must be refused (RFC 8725: fixed algorithms, audience, expiry), how the key set is cached and refetched, and what is a verdict on the token
(None) versus the provider not being able to answer (AuthUnavailable). Generated keys and httpx.MockTransport - no network, no Supabase."""

import asyncio
import base64
import hashlib
import hmac
import json
import time

import httpx
import jwt
import pytest
from agent_framework.api.jwt_keys import KeySetVerifier
from agent_framework.api.supabase import build_supabase_user_verifier
from agent_framework.core.auth import AuthUnavailable
from cryptography.hazmat.primitives import serialization
from cryptography.hazmat.primitives.asymmetric import ec, rsa
from jwt.algorithms import ECAlgorithm, RSAAlgorithm

SECRET = "a-shared-secret-long-enough-for-hs256-use"
URL = "https://project.supabase.co"
KEYS_URL = f"{URL}/auth/v1/.well-known/jwks.json"
USER_URL = f"{URL}/auth/v1/user"


def ec_key() -> ec.EllipticCurvePrivateKey:
    return ec.generate_private_key(ec.SECP256R1())


def rsa_key() -> rsa.RSAPrivateKey:
    return rsa.generate_private_key(public_exponent=65537, key_size=2048)


def jwk(private_key: object, kid: str) -> dict:
    algorithm = ECAlgorithm if isinstance(private_key, ec.EllipticCurvePrivateKey) else RSAAlgorithm
    return {**json.loads(algorithm.to_jwk(private_key.public_key())), "kid": kid}  # type: ignore[attr-defined]


def token(
    private_key: object,
    kid: str = "k1",
    *,
    alg: str = "ES256",
    sub: str | None = "user-1",
    aud: str = "authenticated",
    exp_in: int | None = 3600,
    **extra: object,
) -> str:
    claims: dict = {"aud": aud, "role": "authenticated", **extra}
    if exp_in is not None:
        claims["exp"] = int(time.time()) + exp_in
    if sub is not None:
        claims["sub"] = sub
    return jwt.encode(claims, private_key, algorithm=alg, headers={"kid": kid})  # type: ignore[arg-type]


def hs256_by_hand(secret: bytes, header: dict, payload: dict) -> str:
    def b64(data: bytes) -> bytes:
        return base64.urlsafe_b64encode(data).rstrip(b"=")

    signing_input = b64(json.dumps(header).encode()) + b"." + b64(json.dumps(payload).encode())
    signature = hmac.new(secret, signing_input, hashlib.sha256).digest()
    return (signing_input + b"." + b64(signature)).decode()


class Provider:
    """A fake identity provider: serves a key set and counts the requests it gets."""

    def __init__(self, keys: list[dict] | None = None) -> None:
        self.keys = keys or []
        self.key_set_requests = 0
        self.user_requests: list[httpx.Request] = []
        self.key_set_status = 200
        self.user_status = 200
        self.user_error: Exception | None = None

    def handle(self, request: httpx.Request) -> httpx.Response:
        if request.url.path.endswith("jwks.json"):
            self.key_set_requests += 1
            if self.key_set_status != 200:
                return httpx.Response(self.key_set_status)
            return httpx.Response(200, json={"keys": self.keys})
        self.user_requests.append(request)
        if self.user_error is not None:
            raise self.user_error
        return httpx.Response(self.user_status, json={"id": "user-from-auth"} if self.user_status == 200 else {"msg": "no"})

    def transport(self) -> httpx.MockTransport:
        return httpx.MockTransport(self.handle)


def keyset(provider: Provider, clock: list[float] | None = None, **kwargs: object) -> KeySetVerifier:
    now = clock if clock is not None else [0.0]
    return KeySetVerifier(KEYS_URL, audience="authenticated", client=httpx.AsyncClient(transport=provider.transport()), clock=lambda: now[0], **kwargs)  # type: ignore[arg-type]


class TestAValidToken:
    async def test_es256_gives_the_user_id(self) -> None:
        key = ec_key()
        verifier = keyset(Provider([jwk(key, "k1")]))

        assert await verifier.user_id(token(key)) == "user-1"

    async def test_rs256_gives_the_user_id(self) -> None:
        key = rsa_key()
        verifier = keyset(Provider([jwk(key, "k1")]))

        assert await verifier.user_id(token(key, alg="RS256")) == "user-1"

    async def test_a_token_that_expired_a_moment_ago_is_still_accepted_within_the_leeway_but_not_beyond_it(self) -> None:
        key = ec_key()
        verifier = keyset(Provider([jwk(key, "k1")]))

        assert await verifier.user_id(token(key, exp_in=-5)) == "user-1"
        assert await verifier.user_id(token(key, exp_in=-60)) is None


class TestAnyOtherTokenIsRefused:
    @pytest.fixture
    def setup(self) -> tuple[ec.EllipticCurvePrivateKey, Provider, KeySetVerifier]:
        key = ec_key()
        provider = Provider([jwk(key, "k1")])
        return key, provider, keyset(provider)

    async def test_expired(self, setup: tuple) -> None:
        key, _, verifier = setup
        assert await verifier.user_id(token(key, exp_in=-3600)) is None

    async def test_wrong_audience(self, setup: tuple) -> None:
        key, _, verifier = setup
        assert await verifier.user_id(token(key, aud="anon")) is None

    async def test_no_subject(self, setup: tuple) -> None:
        key, _, verifier = setup
        assert await verifier.user_id(token(key, sub=None)) is None

    async def test_no_expiry(self, setup: tuple) -> None:
        key, _, verifier = setup
        assert await verifier.user_id(token(key, exp_in=None)) is None  # type: ignore[arg-type]

    async def test_signed_by_another_key_that_claims_the_same_kid(self, setup: tuple) -> None:
        _, _, verifier = setup
        assert await verifier.user_id(token(ec_key(), "k1")) is None

    async def test_an_rs256_token_naming_an_ec_key(self, setup: tuple) -> None:
        _, _, verifier = setup
        assert await verifier.user_id(token(rsa_key(), "k1", alg="RS256")) is None

    async def test_something_that_is_not_a_token(self, setup: tuple) -> None:
        _, provider, verifier = setup
        for junk in ("", "abc", "a.b.c", "Bearer x"):
            assert await verifier.user_id(junk) is None
        assert provider.key_set_requests == 0  # refused without a fetch

    async def test_alg_none_and_hs256_are_never_accepted_here_and_cost_no_fetch(self, setup: tuple) -> None:
        key, provider, verifier = setup
        unsigned = jwt.encode({"sub": "user-1", "aud": "authenticated", "exp": int(time.time()) + 60}, key=None, algorithm="none", headers={"kid": "k1"})
        public_key_as_secret = key.public_key().public_bytes(serialization.Encoding.DER, serialization.PublicFormat.SubjectPublicKeyInfo)
        # the classic algorithm-confusion attempt: HS256 keyed with the public key. PyJWT refuses to build it, so it is made by hand.
        claims = {"sub": "user-1", "aud": "authenticated", "exp": int(time.time()) + 60}
        confused = hs256_by_hand(public_key_as_secret, {"alg": "HS256", "typ": "JWT", "kid": "k1"}, claims)

        assert await verifier.user_id(unsigned) is None
        assert await verifier.user_id(confused) is None
        assert provider.key_set_requests == 0

    async def test_a_token_without_a_kid(self, setup: tuple) -> None:
        key, provider, verifier = setup
        no_kid = jwt.encode({"sub": "u", "aud": "authenticated", "exp": int(time.time()) + 60}, key, algorithm="ES256")

        assert await verifier.user_id(no_kid) is None
        assert provider.key_set_requests == 0


class TestTheKeySet:
    async def test_is_fetched_once_for_many_tokens_and_for_many_at_the_same_moment(self) -> None:
        key = ec_key()
        provider = Provider([jwk(key, "k1")])
        verifier = keyset(provider)

        results = await asyncio.gather(*(verifier.user_id(token(key)) for _ in range(50)))
        await asyncio.gather(*(verifier.user_id(token(key)) for _ in range(50)))

        assert results == ["user-1"] * 50
        assert provider.key_set_requests == 1

    async def test_is_fetched_again_once_the_refresh_time_has_passed(self) -> None:
        key = ec_key()
        provider = Provider([jwk(key, "k1")])
        clock = [0.0]
        verifier = keyset(provider, clock, refresh_seconds=600)
        await verifier.user_id(token(key))

        clock[0] = 599
        await verifier.user_id(token(key))
        assert provider.key_set_requests == 1
        clock[0] = 600
        await verifier.user_id(token(key))
        assert provider.key_set_requests == 2

    async def test_a_key_rotated_in_is_picked_up_by_one_refetch(self) -> None:
        old, new = ec_key(), ec_key()
        provider = Provider([jwk(old, "k1")])
        clock = [0.0]
        verifier = keyset(provider, clock, unknown_kid_wait_seconds=30)
        await verifier.user_id(token(old))

        provider.keys = [jwk(old, "k1"), jwk(new, "k2")]  # the provider rotated
        clock[0] = 31

        assert await verifier.user_id(token(new, "k2")) == "user-1"
        assert provider.key_set_requests == 2

    async def test_tokens_with_made_up_kids_cannot_make_this_server_hammer_the_provider(self) -> None:
        key = ec_key()
        provider = Provider([jwk(key, "k1")])
        clock = [0.0]
        verifier = keyset(provider, clock, unknown_kid_wait_seconds=30)
        await verifier.user_id(token(key))

        clock[0] = 31
        for i in range(100):
            assert await verifier.user_id(token(ec_key(), f"made-up-{i}")) is None

        assert provider.key_set_requests == 2  # the first fetch plus ONE refetch for all the unknown kids

    async def test_with_no_key_set_and_none_reachable_the_answer_is_unavailable_and_it_waits_before_trying_again(self) -> None:
        key = ec_key()
        provider = Provider([jwk(key, "k1")])
        provider.key_set_status = 503
        clock = [0.0]
        verifier = keyset(provider, clock, unknown_kid_wait_seconds=30)

        with pytest.raises(AuthUnavailable):
            await verifier.user_id(token(key))
        with pytest.raises(AuthUnavailable):
            await verifier.user_id(token(key))
        assert provider.key_set_requests == 1  # the second request did not hit the provider

        provider.key_set_status = 200
        clock[0] = 31
        assert await verifier.user_id(token(key)) == "user-1"

    async def test_a_key_set_that_cannot_be_read_is_unavailable_not_a_bad_token(self) -> None:
        key = ec_key()
        provider = Provider()
        provider.handle = lambda request: httpx.Response(200, text="<html>not json</html>")  # type: ignore[method-assign]
        verifier = keyset(provider)

        with pytest.raises(AuthUnavailable):
            await verifier.user_id(token(key))

    async def test_the_keys_already_held_keep_working_when_a_refetch_fails(self) -> None:
        key = ec_key()
        provider = Provider([jwk(key, "k1")])
        clock = [0.0]
        verifier = keyset(provider, clock, refresh_seconds=600)
        await verifier.user_id(token(key))

        provider.key_set_status = 500
        clock[0] = 700

        assert await verifier.user_id(token(key)) == "user-1"


class TestTheSupabaseVerifier:
    async def test_an_asymmetric_token_is_checked_here_and_asks_supabase_nothing(self) -> None:
        key = ec_key()
        provider = Provider([jwk(key, "k1")])
        verify = build_supabase_user_verifier(URL, "anon-key", transport=provider.transport())

        assert [await verify(token(key)) for _ in range(20)] == ["user-1"] * 20

        assert provider.user_requests == []
        assert provider.key_set_requests == 1

    async def test_a_legacy_hs256_token_is_checked_by_supabase_auth_with_the_callers_own_token(self) -> None:
        provider = Provider()
        verify = build_supabase_user_verifier(URL, "anon-key", transport=provider.transport())
        legacy = jwt.encode({"sub": "user-from-auth", "aud": "authenticated", "exp": int(time.time()) + 60}, SECRET, algorithm="HS256")

        assert await verify(legacy) == "user-from-auth"

        (request,) = provider.user_requests
        assert request.url == USER_URL
        assert request.headers["authorization"] == f"Bearer {legacy}" and request.headers["apikey"] == "anon-key"
        assert provider.key_set_requests == 0

    @pytest.mark.parametrize("status", [401, 403, 400, 404])
    async def test_auth_saying_no_is_an_invalid_token(self, status: int) -> None:
        provider = Provider()
        provider.user_status = status
        verify = build_supabase_user_verifier(URL, "anon-key", transport=provider.transport())

        assert await verify(jwt.encode({"sub": "u"}, SECRET, algorithm="HS256")) is None

    @pytest.mark.parametrize("status", [429, 500, 502, 503, 504])
    async def test_auth_throttling_or_failing_is_unavailable_never_an_invalid_token(self, status: int) -> None:
        provider = Provider()
        provider.user_status = status
        verify = build_supabase_user_verifier(URL, "anon-key", transport=provider.transport())

        with pytest.raises(AuthUnavailable):
            await verify(jwt.encode({"sub": "u"}, SECRET, algorithm="HS256"))

    @pytest.mark.parametrize("error", [httpx.ConnectError("refused"), httpx.ReadTimeout("slow")])
    async def test_auth_unreachable_or_timing_out_is_unavailable(self, error: Exception) -> None:
        provider = Provider()
        provider.user_error = error
        verify = build_supabase_user_verifier(URL, "anon-key", transport=provider.transport())

        with pytest.raises(AuthUnavailable):
            await verify(jwt.encode({"sub": "u"}, SECRET, algorithm="HS256"))

    async def test_a_token_that_cannot_be_valid_costs_no_call_at_all(self) -> None:
        provider = Provider()
        verify = build_supabase_user_verifier(URL, "anon-key", transport=provider.transport())
        unsigned = jwt.encode({"sub": "u"}, key=None, algorithm="none")

        assert [await verify(t) for t in ("", "garbage", unsigned)] == [None, None, None]
        assert provider.user_requests == [] and provider.key_set_requests == 0

    async def test_one_client_serves_every_check(self, monkeypatch: pytest.MonkeyPatch) -> None:
        built: list[int] = []
        original = httpx.AsyncClient.__init__

        def counting(self: httpx.AsyncClient, *args: object, **kwargs: object) -> None:
            built.append(1)
            original(self, *args, **kwargs)  # type: ignore[arg-type]

        monkeypatch.setattr(httpx.AsyncClient, "__init__", counting)
        provider = Provider()
        verify = build_supabase_user_verifier(URL, "anon-key", transport=provider.transport())
        legacy = jwt.encode({"sub": "u"}, SECRET, algorithm="HS256")

        for _ in range(30):
            await verify(legacy)

        assert len(built) == 1 and len(provider.user_requests) == 30  # a new client per check was the old way
