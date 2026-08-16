-- Admin functions: who can access /api/admin/*, and blocking accounts.
-- No one is an admin by default — promote the first admin manually:
--   UPDATE profiles SET is_admin = true WHERE id = '<your-user-id>';

ALTER TABLE profiles ADD COLUMN IF NOT EXISTS is_admin BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE profiles ADD COLUMN IF NOT EXISTS is_blocked BOOLEAN NOT NULL DEFAULT false;
