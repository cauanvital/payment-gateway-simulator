CREATE OR REPLACE FUNCTION prevent_transaction_events_changes()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION
        'transaction_events is append-only and cannot be updated or deleted';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_transaction_events_no_update
BEFORE UPDATE ON transaction_events
FOR EACH ROW
EXECUTE FUNCTION prevent_transaction_events_changes();

CREATE TRIGGER trg_transaction_events_no_delete
BEFORE DELETE ON transaction_events
FOR EACH ROW
EXECUTE FUNCTION prevent_transaction_events_changes();