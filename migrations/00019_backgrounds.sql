-- +goose Up
-- Background images uploaded by users (max. 5 MB; raster images only). They
-- form a shared library from which users, customers and projects pick.
CREATE TABLE backgrounds (
    id          INTEGER PRIMARY KEY,
    mime        TEXT     NOT NULL,
    size        INTEGER  NOT NULL,
    uploaded_by INTEGER  REFERENCES users (id) ON DELETE SET NULL,
    created_at  DATETIME NOT NULL,
    data        BLOB     NOT NULL
);

-- Background of a customer / project: '' = none (inherit), 'g<n>' = gradient
-- preset, 'img:<id>' = uploaded image. The user's own background is a pref.
ALTER TABLE customers ADD COLUMN background TEXT NOT NULL DEFAULT '';
ALTER TABLE projects ADD COLUMN background TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE projects DROP COLUMN background;
ALTER TABLE customers DROP COLUMN background;
DROP TABLE backgrounds;
