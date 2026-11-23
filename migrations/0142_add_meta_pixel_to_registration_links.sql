-- +goose Up
-- +goose StatementBegin
-- Meta Pixel + CAPI per link pendaftaran. meta_tracking = 1: halaman
-- /daftar/:slug memuat Pixel utama (META_PIXEL_ID / META_CAPI_DATASET_ID)
-- dan event form dikirim juga lewat CAPI (dedup via event_id).
-- meta_pixel_extra: Pixel ID tambahan (pisahkan koma), hanya sisi browser.
ALTER TABLE registration_links ADD COLUMN meta_tracking INTEGER NOT NULL DEFAULT 1 CHECK (meta_tracking IN (0, 1));
ALTER TABLE registration_links ADD COLUMN meta_pixel_extra TEXT NOT NULL DEFAULT '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE registration_links DROP COLUMN meta_pixel_extra;
ALTER TABLE registration_links DROP COLUMN meta_tracking;
-- +goose StatementEnd
