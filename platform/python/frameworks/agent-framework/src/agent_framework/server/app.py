"""Builds the FastAPI app: discovers agents, wires the generic chat route, enables CORS for local
Next.js dev."""

from fastapi import APIRouter, FastAPI
from fastapi.middleware.cors import CORSMiddleware

from agent_framework.core.auth import UserVerifier
from agent_framework.core.limits import Limits
from agent_framework.core.providers.credits import CreditsSource
from agent_framework.core.registry import discover
from agent_framework.core.store.base import SessionStore
from agent_framework.core.store.memory_store import MemoryStore
from agent_framework.core.tool_registry import ToolEntry
from agent_framework.core.types import RegisteredAgent
from agent_framework.server.routes.chat import build_router
from agent_framework.server.routes.credits import build_credits_router
from agent_framework.server.routes.introspect import build_introspection_router
from agent_framework.server.routes.knowledge import build_knowledge_router
from agent_framework.server.usage import TokenLimit


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
    # Soft per-session token allowance, reported with every finished turn so a client can warn before the
    # session gets expensive. Nothing is refused or cut off. None = no limit.
    token_limit: TokenLimit | None = None,
    # Longest message accepted, in characters; None = no cap. A longer one is refused (413) before it costs a model call.
    max_message_chars: int | None = None,
    # The interactive API docs (/docs, /redoc, /openapi.json): useful in development, an unneeded public
    # description of every route in production.
    docs_enabled: bool = True,
    # Which of the discovered agents are served; None = all of them. An agent that is not listed is not registered
    # at all - nothing to call, the same 404 as a name that never existed - so a deployment serves exactly the
    # agents it names, whatever else lives in the package (an anonymous website agent, an older one). A name that
    # matches no agent stops startup: a typo must not quietly serve fewer (or different) agents than intended.
    enabled_agents: list[str] | None = None,
    # What the provider key has spent (core/providers/credits.py), served at GET /account/credits to signed-in users.
    # None -> that route answers {"supported": false}. Needs a user_verifier; without one the route is not mounted.
    credits: CreditsSource | None = None,
    # The most model requests one turn may make (each tool call is another request); a model that keeps calling tools is stopped there with a clear
    # error instead of at pydantic-ai's own limit of 50. None = that default.
    max_requests_per_turn: int | None = None,
) -> FastAPI:
    app = FastAPI() if docs_enabled else FastAPI(docs_url=None, redoc_url=None, openapi_url=None)

    # No dependency checks (store/model reachability) - a health check is polled frequently by the
    # deploy platform to decide whether to keep routing traffic to this container/restart it, and a
    # transient blip in a dependency shouldn't flap that decision. This only confirms the process
    # itself is up and serving.
    @app.get("/health")
    def health() -> dict[str, str]:
        return {"status": "ok"}

    agents = _served(discover(agents_package), enabled_agents)
    app.include_router(
        build_router(
            agents,
            store or MemoryStore(),
            tool_registry,
            debug_mode,
            limits,
            user_verifier,
            secret_key=secret_key,
            token_limit=token_limit,
            max_message_chars=max_message_chars,
            max_requests_per_turn=max_requests_per_turn,
        )
    )
    if auth_router is not None:
        app.include_router(auth_router)
    if user_verifier is not None:
        app.include_router(build_credits_router(credits, user_verifier))
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


def _served(discovered: list[RegisteredAgent], enabled: list[str] | None) -> dict[str, RegisteredAgent]:
    agents = {registered.name: registered for registered in discovered}
    if enabled is None:
        return agents
    unknown = sorted(set(enabled) - set(agents))
    if unknown:
        raise ValueError(f"enabled_agents names agents that do not exist: {unknown} (found: {sorted(agents)})")
    return {name: registered for name, registered in agents.items() if name in enabled}
