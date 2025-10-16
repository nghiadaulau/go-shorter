CREATE TYPE user_role AS ENUM ('admin','editor','viewer');

-- sequence for Base62 slug ids
CREATE SEQUENCE IF NOT EXISTS slug_seq;

CREATE TABLE tenants (
  id BIGSERIAL PRIMARY KEY,
  name TEXT NOT NULL,
  domain TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'active',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE users (
  id BIGSERIAL PRIMARY KEY,
  tenant_id BIGINT NOT NULL REFERENCES tenants(id),
  email TEXT NOT NULL,
  password_hash TEXT NOT NULL,
  role user_role NOT NULL DEFAULT 'admin',
  status TEXT NOT NULL DEFAULT 'active',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, email)
);

CREATE TABLE links (
  id BIGSERIAL PRIMARY KEY,
  tenant_id BIGINT NOT NULL REFERENCES tenants(id),
  slug VARCHAR(32) NOT NULL UNIQUE,
  target_url TEXT NOT NULL,
  is_active BOOLEAN NOT NULL DEFAULT true,
  expire_at TIMESTAMPTZ NULL,
  max_clicks BIGINT NULL,
  created_by BIGINT NULL REFERENCES users(id),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE link_meta (
  link_id BIGINT PRIMARY KEY REFERENCES links(id) ON DELETE CASCADE,
  title TEXT,
  description TEXT,
  og_image TEXT,
  no_preview BOOLEAN NOT NULL DEFAULT false
);

CREATE TYPE device_type AS ENUM ('ios','android','desktop');

CREATE TABLE link_rules (
  id BIGSERIAL PRIMARY KEY,
  link_id BIGINT NOT NULL REFERENCES links(id) ON DELETE CASCADE,
  country_code CHAR(2) NULL,
  device device_type NULL,
  weight INT NULL,
  target_url TEXT NOT NULL
);

CREATE TABLE clicks_agg (
  day DATE NOT NULL,
  slug TEXT NOT NULL,
  tenant_id BIGINT NOT NULL,
  total BIGINT NOT NULL DEFAULT 0,
  PRIMARY KEY (day, slug, tenant_id)
);
