USE isupipe;
CREATE INDEX idx_reservation_range ON reservation_slots(start_at,end_at);
