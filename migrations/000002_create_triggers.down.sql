DROP TRIGGER IF EXISTS trg_transaction_events_no_delete ON transaction_events;
DROP TRIGGER IF EXISTS trg_transaction_events_no_update ON transaction_events;
DROP FUNCTION IF EXISTS prevent_transaction_events_changes();
