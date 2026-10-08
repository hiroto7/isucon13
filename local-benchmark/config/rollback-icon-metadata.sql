USE isupipe;
ALTER TABLE icons DROP INDEX idx_icons_user, DROP COLUMN image_hash, ADD INDEX idx_icons_user(user_id);
