"""Builds the FastAPI app: discovers agents, wires the generic chat route, enables CORS for local
Next.js dev."""

from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware

from agent_framework.core.limits import Limits
from agent_framework.core.registry import discover
from agent_framework.core.store.base import SessionStore
from agent_framework.core.store.memory_store import MemoryStore
from agent_framework.core.tool_registry import ToolEntry
from agent_framework.server.routes.chat import build_router
from agent_framework.server.routes.introspect import build_introspection_router
from agent_framework.server.routes.knowledge import build_knowledge_router


def create_app(
    agents_package: str,
    store: SessionStore | None = None,
    cors_origins: list[str] | None = None,
    tool_registry: dict[str, ToolEntry] | None = None,
    debug_mode: bool = False,
    limits: Limits | None = None,
    *,
    secret_key: str,
) -> FastAPI:
    app = FastAPI()
    agents = {registered.name: registered for registered in discover(agents_package)}
    app.include_router(build_router(agents, store or MemoryStore(), tool_registry, debug_mode, limits, secret_key=secret_key))
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
