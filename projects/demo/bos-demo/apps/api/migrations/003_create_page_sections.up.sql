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

-- Backfill: every existing page's flat body becomes one "prose" section in its main slot,
-- so nothing already created (the MVP's seeded "home" page) is lost.
INSERT INTO page_sections (page_id, slot, position, renderer, content)
SELECT id, 'main', 0, 'prose', jsonb_build_object('body', body)
FROM pages
WHERE body IS NOT NULL AND body <> '';

ALTER TABLE pages DROP COLUMN body;
