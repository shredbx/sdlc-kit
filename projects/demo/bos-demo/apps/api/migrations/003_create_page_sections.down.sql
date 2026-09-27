ALTER TABLE pages ADD COLUMN body text NOT NULL DEFAULT '';

UPDATE pages
SET body = COALESCE(
    (SELECT (content ->> 'body') FROM page_sections
     WHERE page_sections.page_id = pages.id AND page_sections.renderer = 'prose'
     ORDER BY position LIMIT 1),
    ''
);

DROP TABLE page_sections;
