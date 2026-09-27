CREATE TABLE settings (
    id int PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    site_title text NOT NULL DEFAULT 'bos-demo',
    tagline text NOT NULL DEFAULT ''
);

INSERT INTO settings (id, site_title, tagline) VALUES (1, 'bos-demo', 'The bos product''s own reference build.');
