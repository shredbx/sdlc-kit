"""Generic structured chat output: a short text reply plus zero or more clickable/navigable
cards. Any agent can opt in by setting `output_type=str | AgentReply` — the model still answers
plain questions with plain text, and switches to this shape only when it has something (e.g.
search results) worth presenting as cards."""

from pydantic import BaseModel


class Card(BaseModel):
    id: str
    title: str
    subtitle: str | None = None
    price_display: str | None = None
    image_url: str | None = None
    link: str | None = None


class AgentReply(BaseModel):
    text: str
    cards: list[Card] = []
