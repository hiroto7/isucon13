-- Apply once after provisioning; persists across zone reloads and reboot.
CREATE INDEX nametype_index ON isudns.records(name, type);
-- Rollback: DROP INDEX nametype_index ON isudns.records;
