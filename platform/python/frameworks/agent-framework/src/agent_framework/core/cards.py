"""Generic structured chat output: a short text reply plus zero or more clickable/navigable
cards. Any agent can opt in by setting `output_type=str | AgentReply` — the model still answers
plain questions with plain text, and switches to this shape only when it has something (e.g.
search results) worth presenting as cards."""

from pydantic import BaseModel, Field


class Card(BaseModel):
    id: str = Field(description="Stable identifier for the thing this card represents (e.g. a listing id).")
    title: str = Field(description="The card's main heading.")
    subtitle: str | None = Field(default=None, description="A short secondary line, e.g. property type and area.")
    price_display: str | None = Field(default=None, description="Pre-formatted price string, e.g. '฿45,000' - already localized, not a raw number.")
    image_url: str | None = Field(default=None, description="A single representative image URL, if one exists.")
    link: str | None = Field(default=None, description="Where the card should navigate to - a relative site path (e.g. /p/{id}) or an absolute URL (e.g. a WhatsApp link).")


class AgentReply(BaseModel):
    text: str = Field(description="The plain-text reply, always required even when cards are also present - a channel that can't render cards still needs something to show.")
    cards: list[Card] = Field(default_factory=list, description="Zero or more cards to render alongside the text, e.g. one per search result.")


def flatten_cards_to_text(cards: list[Card]) -> str:
    """Plain-text fallback for a channel whose RegisteredAgent.output_protocol is "plain_text"
    (core/types.py) - one line per card, empty fields simply omitted rather than printed blank.
    Used by routes/chat.py so a Facebook/WhatsApp-style consumer still gets the card information,
    just folded into .text instead of a separate .cards list it wouldn't render anyway."""
    lines = []
    for card in cards:
        parts = [card.title]
        if card.subtitle:
            parts.append(f"({card.subtitle})")
        if card.price_display:
            parts.append(f"— {card.price_display}")
        if card.link:
            parts.append(f"— {card.link}")
        lines.append(" ".join(parts))
    return "\n".join(lines)
