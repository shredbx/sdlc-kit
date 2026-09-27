ALTER TABLE settings
    ADD COLUMN header jsonb NOT NULL DEFAULT '{"enabled": true, "preset": "default", "nav": []}',
    ADD COLUMN footer jsonb NOT NULL DEFAULT '{"enabled": true, "preset": "default", "sections": []}';
