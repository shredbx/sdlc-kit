"""Generic structured chat output: a short text reply plus zero or more clickable/navigable
cards. Any agent can opt in by setting `output_type=str | AgentReply` — the model still answers
plain questions with plain text, and switches to this shape only when it has something (e.g.
search results) worth presenting as cards."""

from pydantic import BaseModel, Field


class CardAction(BaseModel):
    """A quick action on a card (e.g. "I'm interested") - deliberately NOT a tool call. Clicking
    one only inserts `text` into the human's own outgoing message box, never sends it and never
    triggers anything server-side by itself; the human still reviews/edits/sends it like any other
    message, which then goes through the normal agent.run() flow same as if they'd typed it. This
    is what keeps it from ever being confused with a real tool: a tool is something the MODEL
    decides to call with validated args; a CardAction is something a HUMAN clicks to save typing.
    Content is authored by the same deterministic `to_reply()` mapping that builds the card itself
    (see tools/properties/to_reply.py) - never composed by the model, so it's exactly as
    predictable as the rest of a card's fields."""

    label: str = Field(description="Button text, e.g. \"I'm interested\".")
    text: str = Field(description="Plain text to insert into the outgoing message box when clicked - the human can still edit it before sending, and nothing is sent automatically.")


class Card(BaseModel):
    id: str = Field(description="Stable identifier for the thing this card represents (e.g. a listing id).")
    title: str = Field(description="The card's main heading.")
    subtitle: str | None = Field(default=None, description="A short secondary line, e.g. property type and area.")
    price_display: str | None = Field(default=None, description="Pre-formatted price string, e.g. '฿45,000' - already localized, not a raw number.")
    image_url: str | None = Field(default=None, description="A single representative image URL, if one exists.")
    link: str | None = Field(default=None, description="Where the card should navigate to - a relative site path (e.g. /p/{id}) or an absolute URL (e.g. a WhatsApp link).")
    actions: list[CardAction] = Field(default_factory=list, description="Zero or more quick actions for this card (e.g. \"I'm interested\") - a plain_text channel has no way to render these as buttons, so they're dropped (not rendered as text) by flatten_cards_to_text below.")


class AgentReply(BaseModel):
    text: str = Field(description="The plain-text reply, always required even when cards are also present - a channel that can't render cards still needs something to show.")
    cards: list[Card] = Field(default_factory=list, description="Zero or more cards to render alongside the text, e.g. one per search result.")


def flatten_cards_to_text(cards: list[Card]) -> str:
    """Plain-text fallback for a channel whose RegisteredAgent.output_protocol is "plain_text"
    (core/types.py) - one line per card, empty fields simply omitted rather than printed blank.
    Used by routes/chat.py so a Facebook/WhatsApp-style consumer still gets the card information,
    just folded into .text instead of a separate .cards list it wouldn't render anyway.
    `card.actions` is deliberately never rendered here - there's no button to click in a paste-back
    Messenger flow, and printing its `text` as a plain line would just be a confusing duplicate of
    the card's own title/link."""
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
