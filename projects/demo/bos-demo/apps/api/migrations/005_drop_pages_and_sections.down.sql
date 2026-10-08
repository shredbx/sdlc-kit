-- Recreates the dropped tables' shape (001 + 003 combined, post-backfill) for
-- reversibility. Data is NOT restored — see the up migration's own note.
CREATE TABLE pages (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    slug text NOT NULL UNIQUE,
    title text NOT NULL,
    published boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE page_sections (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    page_id uuid NOT NULL REFERENCES pages(id) ON DELETE CASCADE,
    slot text NOT NULL DEFAULT 'main',
    position int NOT NULL DEFAULT 0,
    renderer text NOT NULL,
    content jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX page_sections_page_id_slot_position_idx ON page_sections (page_id, slot, position);
