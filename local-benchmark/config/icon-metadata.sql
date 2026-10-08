USE isupipe;
ALTER TABLE icons ADD COLUMN image_hash CHAR(64) CHARACTER SET ascii GENERATED ALWAYS AS (SHA2(image, 256)) STORED,
  DROP INDEX idx_icons_user, ADD INDEX idx_icons_user(user_id, image_hash);
