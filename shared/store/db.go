package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/caio-ishikawa/target-tracker/shared/models"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

type Database struct {
	connection *sql.DB
}

func Init() (Database, error) {
	// dbPath := os.Getenv("SQLITE_PATH")
	// if dbPath == "" {
	// 	return Database{}, fmt.Errorf("Failed to get DB path from environment variable")
	// }

	db, err := sql.Open("sqlite3", "./test.db")
	if err != nil {
		return Database{}, fmt.Errorf("Failed to start DB connection: %w", err)
	}

	// Create tables on startup
	if _, err := db.Exec(createTablesQuery); err != nil {
		return Database{}, fmt.Errorf("Failed to create tables: %w", err)
	}

	return Database{
		connection: db,
	}, nil
}

func (db Database) InsertTarget(target models.Target) error {
	if _, err := db.connection.Exec(`INSERT INTO target (uuid, name) VALUES (?, ?)`, target.UUID, target.Name); err != nil {
		return fmt.Errorf("Failed to insert target: %w", err)
	}

	return nil
}

func (db Database) GetTarget(uuid string) (*models.Target, error) {
	var target models.Target
	err := db.connection.QueryRow(`SELECT uuid, name FROM target WHERE uuid = ?`, uuid).Scan(&target.UUID, &target.Name)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, fmt.Errorf("Failed to get target: %w", err)
	}

	return &target, nil
}

func (db Database) RemoveTarget(targetUUID uuid.UUID) error {
	// TODO: delete from other tables too
	if _, err := db.connection.Exec(`DELETE FROM target WHERE uuid = ?`, targetUUID); err != nil {
		return fmt.Errorf("Failed to delete target: %w", err)
	}

	return nil
}

func (db Database) InsertScope(scope models.Scope) error {
	if _, err := db.connection.Exec(
		`INSERT INTO scope (uuid, target_uuid, url, accept_subdomains) VALUES (?, ?, ?, ?)`,
		scope.UUID,
		scope.TargetUUID,
		scope.URL,
		scope.AcceptSubdomains,
	); err != nil {
		return fmt.Errorf("Failed to insert scope: %w", err)
	}

	return nil
}

func (db Database) GetScope(targetUUID string) (*models.Scope, error) {
	var scope models.Scope
	err := db.connection.QueryRow(`SELECT uuid, target_uuid, url, accept_subdomains, first_run FROM scope WHERE target_uuid = ?`, targetUUID).Scan(
		&scope.UUID,
		&scope.TargetUUID,
		&scope.URL,
		&scope.AcceptSubdomains,
		&scope.FirstRun,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, fmt.Errorf("Failed to get scope: %w", err)
	}

	return &scope, nil
}

func (db Database) GetAllScopes() ([]models.Scope, error) {
	rows, err := db.connection.Query("SELECT uuid, target_uuid, url, accept_subdomains, first_run FROM scope")
	if err != nil {
		return nil, fmt.Errorf("Failed to get all scopes: %w", err)
	}
	defer rows.Close()

	var results []models.Scope
	for rows.Next() {
		var item models.Scope
		if err := rows.Scan(&item.UUID, &item.TargetUUID, &item.URL, &item.AcceptSubdomains, &item.FirstRun); err != nil {
			return nil, fmt.Errorf("Failed to scan scope row: %w", err)
		}

		results = append(results, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("Rows error when getting scopes: %w", err)
	}

	return results, nil
}

func (db Database) UpdateScope(scope models.Scope) error {
	if _, err := db.connection.Exec(
		`UPDATE scope SET url = ?, first_run = ? WHERE uuid = ?`,
		scope.URL,
		scope.FirstRun,
	); err != nil {
		return fmt.Errorf("Failed to update scope: %w", err)
	}

	return nil
}

// Returns a 3d list of strings for use in the CLI
func (db Database) GetDomainsPerTarget(limit, offset int, targetUUID string) ([]models.Domain, error) {
	query := fmt.Sprintf("SELECT url, query_params, status_code FROM domain WHERE target_uuid = ? LIMIT %v OFFSET %v", limit, offset)
	rows, err := db.connection.Query(query, targetUUID)
	if err != nil {
		return nil, fmt.Errorf("Failed to get all domain: %w", err)
	}
	defer rows.Close()

	var results []models.Domain
	for rows.Next() {
		var item models.Domain
		if err := rows.Scan(&item.URL, &item.QueryParams, &item.StatusCode); err != nil {
			return nil, fmt.Errorf("Failed to scan domain row: %w", err)
		}

		results = append(results, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("Rows error when getting domain: %w", err)
	}

	return results, nil
}

func (db Database) InsertDomainRecord(domain models.Domain) error {
	if _, err := db.connection.Exec(
		`INSERT INTO domain (uuid, target_uuid, url, status_code, query_params) VALUES (?, ?, ?, ?, ?)`,
		domain.UUID,
		domain.TargetUUID,
		domain.URL,
		domain.StatusCode,
		domain.QueryParams,
	); err != nil {
		return fmt.Errorf("Failed to insert domain: %w", err)
	}

	return nil
}

func (db Database) UpdateDomainRecord(domain models.Domain) error {
	if _, err := db.connection.Exec(
		`UPDATE domain SET url = ?, status_code = ?, last_updated = ? WHERE uuid = ?`,
		domain.URL,
		domain.StatusCode,
		time.Now(),
		domain.UUID,
	); err != nil {
		return fmt.Errorf("Failed to update domain: %w", err)
	}

	return nil
}

func (db Database) GetTargetByName(name string) (*models.Target, error) {
	var target models.Target
	err := db.connection.QueryRow(
		`SELECT uuid, name FROM target WHERE name= ?`, name).Scan(&target.UUID, &target.Name)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, fmt.Errorf("Failed to get domain: %w", err)
	}

	return &target, nil
}

func (db Database) GetDomainByURL(url string) (*models.Domain, error) {
	var domain models.Domain
	err := db.connection.QueryRow(
		`SELECT uuid, target_uuid, url, status_code, last_updated FROM domain WHERE url = ?`,
		url).Scan(&domain.UUID, &domain.TargetUUID, &domain.URL, &domain.StatusCode, &domain.LastUpdated)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, fmt.Errorf("Failed to get domain: %w", err)
	}

	return &domain, nil
}

func (db Database) InsertPort(port models.Port) error {
	if _, err := db.connection.Exec(
		`INSERT INTO port (uuid, domain_uuid, port, port_state) VALUES (?, ?, ?, ?)`,
		port.UUID,
		port.DomainUUID,
		port.State,
	); err != nil {
		return fmt.Errorf("Failed to insert target: %w", err)
	}

	return nil
}

func (db Database) UpdatePort(port models.Port) error {
	if _, err := db.connection.Exec(
		`UPDATE port SET port = ?, port_state = ?, last_updated = ?`,
		port.Port,
		port.State,
		time.Now(),
	); err != nil {
		return fmt.Errorf("Failed to insert target: %w", err)
	}

	return nil
}

func (db Database) GetPort(domainUUID string) (*models.Port, error) {
	var port models.Port
	err := db.connection.QueryRow(
		`SELECT uuid, domain_uuid, port, state, last_updated FROM domain WHERE domain_uuid = ?`,
		domainUUID).Scan(&port.UUID, &port.DomainUUID, &port.Port, &port.State, &port.LastUpdated)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, fmt.Errorf("Failed to get port: %w", err)
	}

	return &port, nil
}

func (db Database) UpdateDaemonStats(stats models.DaemonStats) error {
	if _, err := db.connection.Exec(
		`UPDATE daemon_stats SET total_found_urls = ?, total_new_urls = ?, total_found_ports = ?, total_new_ports = ?, scan_time = ?, scan_begin = ?, last_scan_ended =?, is_running = ?`,
		stats.TotalFoundURLs,
		stats.TotalNewURLs,
		stats.TotalFoundPorts,
		stats.TotalNewPorts,
		stats.ScanTime.String(),
		stats.ScanBegin.String(),
		stats.LastScanEnded.String(),
		stats.IsRunning,
	); err != nil {
		return fmt.Errorf("Failed to update stats: %w", err)
	}

	return nil
}

func (db Database) GetStats() (*models.DaemonStats, error) {
	var stats models.DaemonStats
	err := db.connection.QueryRow(
		`SELECT total_found_urls, total_new_urls, total_found_ports, total_new_ports, scan_time, scan_begin, last_scan_ended, is_running FROM daemon_stats`).
		Scan(
			&stats.TotalFoundURLs,
			&stats.TotalNewURLs,
			&stats.TotalFoundPorts,
			&stats.TotalNewPorts,
			&stats.ScanTime,
			&stats.ScanBegin,
			&stats.LastScanEnded,
			&stats.IsRunning,
		)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, fmt.Errorf("Failed to get stats: %w", err)
	}

	return &stats, nil
}
