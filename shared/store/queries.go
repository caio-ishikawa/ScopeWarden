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
	scan_time TEXT,
	scan_begin TEXT,
	last_scan_ended TEXT,
	is_running BOOLEAN DEFAULT false
);
INSERT INTO daemon_stats(total_found_urls, total_new_urls, total_found_ports, total_new_ports, scan_time, last_scan_ended, is_running) VALUES
	(0, 0, 0, 0, "", "", false);
`
