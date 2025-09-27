package store

const createTablesQuery = `
CREATE TABLE IF NOT EXISTS target (
	uuid TEXT NOT NULL UNIQUE,
	name TEXT NOT NULL UNIQUE,
	enabled BOOL NOT NULL DEFAULT true
);
CREATE TABLE IF NOT EXISTS scope (
	uuid TEXT NOT NULL UNIQUE,
	target_uuid TEXT NOT NULL,
	url TEXT NOT NULL UNIQUE,
	first_run BOOL NOT NULL DEFAULT true,
	last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS domain (
	uuid TEXT NOT NULL UNIQUE,
	target_uuid TEXT NOT NULL,
	scan_uuid TEXT NOT NULL,
	url TEXT NOT NULL UNIQUE,
	ip_address TEXT,
	status_code INTEGER NOT NULL,
	first_run BOOL NOT NULL DEFAULT true,
	last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS bruteforced (
	uuid TEXT NOT NULL UNIQUE,
	domain_uuid TEXT NOT NULL,
	path TEXT NOT NULL,
	first_run BOOL NOT NULL DEFAULT true,
	last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(domain_uuid, path)
);
CREATE TABLE IF NOT EXISTS port (
	uuid TEXT NOT NULL UNIQUE,
	domain_uuid TEXT NOT NULL,
	port INTEGER NOT NULL,
	protocol TEXT NOT NULL,
	port_state TEXT NOT NULL,
	last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(uuid, port)
);
CREATE TABLE IF NOT EXISTS daemon_stats (
	uuid TEXT NOT NULL UNIQUE,
	total_found_urls INTEGER DEFAULT 0,
	total_new_urls INTEGER DEFAULT 0,
	total_found_ports INTEGER DEFAULT 0,
	total_new_ports INTEGER DEFAULT 0,
	scan_time TEXT NOT NULL,
	scan_begin TEXT NOT NULL,
	last_scan_ended TEXT,
	is_running BOOLEAN DEFAULT false
);
`
