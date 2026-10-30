-- Item 7.5: pixels de anúncio por evento (Meta, Google Analytics 4, TikTok).
-- Só os identificadores públicos. Os pixels rodam apenas nas páginas daquele
-- evento e só depois que o visitante aceita no aviso de cookies.
ALTER TABLE eventos
    ADD COLUMN pixel_meta   VARCHAR(20) NOT NULL DEFAULT '',
    ADD COLUMN pixel_ga4    VARCHAR(20) NOT NULL DEFAULT '',
    ADD COLUMN pixel_tiktok VARCHAR(30) NOT NULL DEFAULT '';
