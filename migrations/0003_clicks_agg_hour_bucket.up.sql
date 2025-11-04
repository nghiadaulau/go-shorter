-- Switch clicks_agg from day-level DATE to timestamp bucket (hour-level)
ALTER TABLE clicks_agg RENAME COLUMN day TO bucket;
ALTER TABLE clicks_agg ALTER COLUMN bucket TYPE timestamptz USING (bucket::timestamptz);
-- Optional: ensure existing rows are at start of day already; we keep as-is


