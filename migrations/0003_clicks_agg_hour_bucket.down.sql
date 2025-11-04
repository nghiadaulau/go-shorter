-- Revert timestamp bucket back to day DATE
ALTER TABLE clicks_agg ALTER COLUMN bucket TYPE date USING (bucket::date);
ALTER TABLE clicks_agg RENAME COLUMN bucket TO day;


