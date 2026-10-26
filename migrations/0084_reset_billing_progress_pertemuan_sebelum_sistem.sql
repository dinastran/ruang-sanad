-- +goose Up
-- +goose StatementBegin

-- Billing progress used to ignore meetings held before the application
-- (kelas.pertemuan_terakhir), so santri in such classes were billed late or not
-- at all. For santri that have no tagihan yet, drop the counter and its ledger;
-- the service replays their completed meetings (Finance Sync or the next
-- finished meeting) and seeds the pre-application meetings correctly.
-- Santri that already have tagihan keep their invoice-anchored progress.
DELETE FROM santri_billing_meeting
WHERE santri_id IN (
    SELECT bm.santri_id
    FROM santri_billing_meeting bm
    JOIN pertemuan p ON p.id = bm.pertemuan_id
    JOIN kelas k ON k.id = p.kelas_id
    WHERE k.pertemuan_terakhir > 0
      AND NOT EXISTS (SELECT 1 FROM tagihan t WHERE t.santri_id = bm.santri_id)
);

DELETE FROM santri_billing_progress
WHERE NOT EXISTS (SELECT 1 FROM santri_billing_meeting bm WHERE bm.santri_id = santri_billing_progress.santri_id)
  AND NOT EXISTS (SELECT 1 FROM tagihan t WHERE t.santri_id = santri_billing_progress.santri_id);

-- +goose StatementEnd

-- +goose Down
-- Data repair only; replayed progress is the correct state.
SELECT 1;
