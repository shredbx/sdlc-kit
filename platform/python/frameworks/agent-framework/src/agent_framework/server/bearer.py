"""The token in an `Authorization: Bearer <token>` header - shared by every route that needs a signed-in user."""


def bearer_token(authorization: str | None) -> str | None:
    if not authorization or not authorization.lower().startswith("bearer "):
        return None
    return authorization[7:].strip() or None
