-- Adds support for file-based data sources: CSV uploads land as tables in
-- a dedicated "uploads" schema, kept separate from the app's own metadata
-- tables in "public" so an ad-hoc query against an uploaded file can be
-- scoped away from user/auth data at the application layer (see
-- queryengine.ValidateFileScope and handlers.FileHandler).
--
-- Note: individual upload tables are created dynamically by FileHandler.Upload
-- at request time (they don't exist yet when this migration runs), so
-- there's no per-table RLS policy here — the primary defense for file
-- sources is the application-layer check in ValidateFileScope, same as
-- the rest of this schema is protected by Go-side ownership checks with
-- RLS as a second layer (see the note at the end of 0001_init.sql).

CREATE SCHEMA IF NOT EXISTS uploads;

ALTER TABLE data_sources ADD COLUMN IF NOT EXISTS file_table TEXT;
