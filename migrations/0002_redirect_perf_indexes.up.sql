-- Composite index to speed up tenant+slug lookup
CREATE INDEX IF NOT EXISTS idx_links_tenant_slug ON links(tenant_id, slug);
