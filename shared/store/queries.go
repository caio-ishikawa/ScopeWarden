package store

const createTablesQuery = `
CREATE TABLE IF NOT EXISTS target (
	uuid TEXT NOT NULL UNIQUE,
	name TEXT NOT NULL UNIQUE
);
CREATE TABLE IF NOT EXISTS scope (
	uuid TEXT NOT NULL UNIQUE,
	target_uuid TEXT NOT NULL,
	url TEXT NOT NULL UNIQUE,
	accept_subdomains BOOL NOT NULL DEFAULT false,
	first_run BOOL NOT NULL DEFAULT true,
	last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS domain (
	uuid TEXT NOT NULL UNIQUE,
	target_uuid TEXT NOT NULL,
	url TEXT NOT NULL UNIQUE,
	query_params TEXT,
	ip_address TEXT,
	status_code INTEGER NOT NULL,
	last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS port (
	uuid TEXT NOT NULL UNIQUE,
	domain_uuid TEXT NOT NULL,
	port INTEGER NOT NULL,
	port_state TEXT NOT NULL,
	last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS daemon_stats (
	total_found_urls INTEGER DEFAULT 0,
	total_new_urls INTEGER DEFAULT 0,
	total_found_ports INTEGER DEFAULT 0,
	total_new_ports INTEGER DEFAULT 0,
	scan_time TEXT NOT NULL,
	scan_begin TEXT NOT NULL,
	is_running BOOLEAN DEFAULT false
);
`
