"""Supabase-backed SessionStore — session CRUD via PostgREST (SUPABASE_URL + a service/secret
key), never a direct Postgres connection. The always-running app process using this store never
touches the database password at all.

Schema DDL can't go through PostgREST at all — it's a data API, not a DDL executor — so this store
assumes its table already exists, created separately by migrate.py's runner over a direct Postgres
connection (see chat-api-python/scripts/migrate_db.py; that's the one place the password is
needed, run manually/once, not by this process).

Targets a non-`public` schema via PostgREST's Accept-Profile (reads) / Content-Profile (writes)
headers — see https://docs.postgrest.org/en/v13/references/api/schemas.html. That schema must be
added to Supabase's own exposed-schemas config (`supabase/config.toml`'s `api.schemas` for local
dev, the hosted dashboard's API settings for production) or PostgREST 404s on it; it must NOT be
granted to the `anon`/`authenticated` Postgres roles, which is what actually keeps session data
invisible to anyone holding only the public/publishable key — the schema being "exposed" to
PostgREST's routing is a separate concern from who has read/write grants on what's inside it."""

import json
from datetime import UTC, datetime

import httpx
from pydantic_ai import ModelMessagesTypeAdapter

from agent_framework.api.supabase import build_supabase_client
from agent_framework.core.store.base import Session, SessionStore


class SupabaseSessionStore(SessionStore):
    def __init__(self, url: str, key: str, schema: str, table: str = "sessions", transport: httpx.AsyncBaseTransport | None = None) -> None:
        self._client = build_supabase_client(url, key, transport=transport)
        self._schema = schema
        self._path = f"/{table}"

    async def get_or_create(self, session_id: str) -> Session:
        response = await self._client.request(
            "GET",
            self._path,
            params={"id": f"eq.{session_id}", "select": "*"},
            headers={"Accept-Profile": self._schema},
        )
        rows = response.json()
        if not rows:
            return Session(id=session_id)  # a miss writes nothing: the first save creates the row
        row = rows[0]
        messages = ModelMessagesTypeAdapter.validate_python(row["messages"] or [])
        return Session(id=row["id"], user_id=row["user_id"], messages=messages, data=row["data"] or {})

    async def save(self, session: Session) -> None:
        payload = {
            "id": session.id,
            "user_id": session.user_id,
            "messages": json.loads(ModelMessagesTypeAdapter.dump_json(session.messages)),
            "data": session.data,
            "updated_at": datetime.now(UTC).isoformat(),  # the upsert would otherwise keep the first save's time
        }
        await self._client.request(
            "POST",
            self._path,
            json=payload,
            headers={"Content-Profile": self._schema, "Prefer": "resolution=merge-duplicates,return=minimal"},
        )
        if session.replaces:
            await self._client.request("DELETE", self._path, params={"id": f"eq.{session.replaces}"}, headers={"Content-Profile": self._schema})
            session.replaces = None

    async def close(self) -> None:
        await self._client.aclose()
