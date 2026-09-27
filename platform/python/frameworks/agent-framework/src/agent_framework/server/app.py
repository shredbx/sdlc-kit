"""Builds the FastAPI app: discovers agents, wires the generic chat route, enables CORS for local
Next.js dev."""

from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware

from agent_framework.core.registry import discover
from agent_framework.core.store.base import SessionStore
from agent_framework.core.store.memory_store import MemoryStore
from agent_framework.server.routes.chat import build_router


def create_app(
    agents_package: str,
    store: SessionStore | None = None,
    cors_origins: list[str] | None = None,
) -> FastAPI:
    app = FastAPI()
    agents = {registered.name: registered for registered in discover(agents_package)}
    app.include_router(build_router(agents, store or MemoryStore()))
    app.add_middleware(
        CORSMiddleware,
        allow_origins=cors_origins or ["http://localhost:3001"],
        allow_methods=["POST"],
        allow_headers=["*"],
    )
    return app
