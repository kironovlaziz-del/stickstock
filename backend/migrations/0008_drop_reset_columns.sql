-- Удаляем колонки reset_token и reset_expires_at, если они есть
ALTER TABLE profiles DROP COLUMN IF EXISTS reset_token;
ALTER TABLE profiles DROP COLUMN IF EXISTS reset_expires_at;
