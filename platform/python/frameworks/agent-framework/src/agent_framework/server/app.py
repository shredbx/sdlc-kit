"""Builds the FastAPI app: discovers agents, wires the generic chat route, enables CORS for local
Next.js dev."""

from fastapi import APIRouter, FastAPI
from fastapi.middleware.cors import CORSMiddleware

from agent_framework.core.auth import UserVerifier
from agent_framework.core.limits import Limits
from agent_framework.core.registry import discover
from agent_framework.core.store.base import SessionStore
from agent_framework.core.store.memory_store import MemoryStore
from agent_framework.core.tool_registry import ToolEntry
from agent_framework.server.routes.chat import build_router
from agent_framework.server.routes.introspect import build_introspection_router
from agent_framework.server.routes.knowledge import build_knowledge_router
from agent_framework.server.usage import Pricing


def create_app(
    agents_package: str,
    store: SessionStore | None = None,
    cors_origins: list[str] | None = None,
    tool_registry: dict[str, ToolEntry] | None = None,
    debug_mode: bool = False,
    limits: Limits | None = None,
    user_verifier: UserVerifier | None = None,
    # Already-built by the consumer (e.g. routes.auth.build_auth_router(supabase_url, anon_key)) -
    # create_app stays provider-agnostic, the same way it never hardcodes Supabase for user_verifier
    # either. None -> no /auth/* routes at all, not a route that 404s or misconfigures.
    auth_router: APIRouter | None = None,
    *,
    secret_key: str,
    # USD per million tokens, optional - lets the streaming route report what a session has cost so
    # far. None (or a missing price) -> the usage summary carries tokens only, never a guessed cost.
    pricing: Pricing | None = None,
    # Longest message accepted, in characters; None = no cap. A longer one is refused (413) before it costs a model call.
    max_message_chars: int | None = None,
    # The interactive API docs (/docs, /redoc, /openapi.json): useful in development, an unneeded public
    # description of every route in production.
    docs_enabled: bool = True,
) -> FastAPI:
    app = FastAPI() if docs_enabled else FastAPI(docs_url=None, redoc_url=None, openapi_url=None)

    # No dependency checks (store/model reachability) - a health check is polled frequently by the
    # deploy platform to decide whether to keep routing traffic to this container/restart it, and a
    # transient blip in a dependency shouldn't flap that decision. This only confirms the process
    # itself is up and serving.
    @app.get("/health")
    def health() -> dict[str, str]:
        return {"status": "ok"}

    agents = {registered.name: registered for registered in discover(agents_package)}
    app.include_router(
        build_router(
            agents,
            store or MemoryStore(),
            tool_registry,
            debug_mode,
            limits,
            user_verifier,
            secret_key=secret_key,
            pricing=pricing,
            max_message_chars=max_message_chars,
        )
    )
    if auth_router is not None:
        app.include_router(auth_router)
    if debug_mode:
        app.include_router(build_introspection_router(agents, tool_registry))
        app.include_router(build_knowledge_router(tool_registry))
    app.add_middleware(
        CORSMiddleware,
        allow_origins=cors_origins or ["http://localhost:3100"],
        allow_methods=["GET", "POST"],
        allow_headers=["*"],
    )
    return app
