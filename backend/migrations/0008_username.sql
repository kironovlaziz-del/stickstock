
ALTER TABLE profiles ADD COLUMN IF NOT EXISTS username TEXT UNIQUE;
ALTER TABLE profiles ADD COLUMN IF NOT EXISTS password_hash TEXT;
ALTER TABLE profiles ADD COLUMN IF NOT EXISTS reset_token TEXT;
ALTER TABLE profiles ADD COLUMN IF NOT EXISTS reset_expires_at TIMESTAMPTZ;
CREATE INDEX IF NOT EXISTS profiles_username_idx ON profiles(username);
