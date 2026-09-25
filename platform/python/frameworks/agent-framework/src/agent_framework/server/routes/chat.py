"""Generic chat endpoint: POST /agents/{name}/chat — works for any RegisteredAgent the registry
discovered. No per-agent route code needed; a new agent folder is automatically served here."""

from fastapi import APIRouter, Depends, HTTPException
from pydantic import BaseModel

from agent_framework.core.store.base import Session, SessionStore
from agent_framework.core.types import RegisteredAgent
from agent_framework.server.middleware.session import session_dependency


class ChatRequest(BaseModel):
    message: str


class ChatResponse(BaseModel):
    reply: str


def build_router(agents: dict[str, RegisteredAgent], store: SessionStore) -> APIRouter:
    router = APIRouter()
    resolve_session = session_dependency(store)

    @router.post("/agents/{name}/chat", response_model=ChatResponse)
    async def chat(name: str, request: ChatRequest, session: Session = Depends(resolve_session)) -> ChatResponse:
        registered = agents.get(name)
        if registered is None:
            raise HTTPException(status_code=404, detail=f"no agent named {name!r}")
        deps = registered.build_deps(session)
        result = await registered.agent.run(request.message, deps=deps)
        return ChatResponse(reply=str(result.output))

    return router
