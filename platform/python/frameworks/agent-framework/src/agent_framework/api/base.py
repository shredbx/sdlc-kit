"""BaseAPIClient — shared httpx setup and auth injection for any integration. A consumer's own
`api/clients/<service>_client.py` extends or wraps this instead of hand-rolling httpx setup."""

from typing import Any

import httpx


class BaseAPIClient:
    def __init__(self, base_url: str, headers: dict[str, str] | None = None, timeout: float = 10.0, transport: httpx.AsyncBaseTransport | None = None) -> None:
        self._client = httpx.AsyncClient(base_url=base_url, headers=headers or {}, timeout=timeout, transport=transport)

    async def request(self, method: str, path: str, **kwargs: Any) -> httpx.Response:
        response = await self._client.request(method, path, **kwargs)
        response.raise_for_status()
        return response

    async def aclose(self) -> None:
        await self._client.aclose()
