package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/caio-ishikawa/scopewarden/shared/models"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

type Database struct {
	connection *sql.DB
}

type TargetTable interface {
	GetAll() []TargetTable
}

func Init() (Database, error) {
	db, err := sql.Open("sqlite3", "./scopewarden.db?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return Database{}, fmt.Errorf("Failed to start DB connection: %w", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

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

func (db Database) UpdateTargetEnabled(targetName string, enabled bool) error {
	if _, err := db.connection.Exec(`UPDATE target SET enabled = ? WHERE name = ?`, enabled, targetName); err != nil {
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
	if _, err := db.connection.Exec(`DELETE FROM target WHERE uuid = ?`, targetUUID); err != nil {
		return fmt.Errorf("Failed to delete target: %w", err)
	}

	if _, err := db.connection.Exec(`DELETE FROM domain WHERE target_uuid = ?`, targetUUID); err != nil {
		return fmt.Errorf("Failed to delete target: %w", err)
	}

	// TODO: delete from port, scope, bruteforced tables too

	return nil
}

func (db Database) InsertScope(scope models.Scope) error {
	if _, err := db.connection.Exec(
		`INSERT INTO scope (uuid, target_uuid, url) VALUES (?, ?, ?)`,
		scope.UUID,
		scope.TargetUUID,
		scope.URL,
	); err != nil {
		return fmt.Errorf("Failed to insert scope: %w", err)
	}

	return nil
}

func (db Database) GetScope(targetUUID string) (*models.Scope, error) {
	var scope models.Scope
	err := db.connection.QueryRow(`SELECT uuid, target_uuid, url, first_run FROM scope WHERE target_uuid = ?`, targetUUID).Scan(
		&scope.UUID,
		&scope.TargetUUID,
		&scope.URL,
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
	rows, err := db.connection.Query("SELECT uuid, target_uuid, url, first_run FROM scope")
	if err != nil {
		return nil, fmt.Errorf("Failed to get all scopes: %w", err)
	}
	defer rows.Close()

	var results []models.Scope
	for rows.Next() {
		var item models.Scope
		if err := rows.Scan(&item.UUID, &item.TargetUUID, &item.URL, &item.FirstRun); err != nil {
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

func (db Database) GetDomainsByTarget(limit, offset int, sortBy models.DomainSortBy, targetUUID string) (models.DomainListResponse, error) {
	query := fmt.Sprintf(`
	SELECT 
		d.uuid,
		d.scan_uuid,
		d.url,
		d.status_code,
		COALESCE(p.cnt_ports, 0) AS count_ports,
		COALESCE(b.cnt_bruteforced, 0) AS count_bruteforced
	FROM domain d
	LEFT JOIN (
		SELECT domain_uuid, COUNT(*) AS cnt_ports
		FROM port
		GROUP BY domain_uuid
	) p ON p.domain_uuid = d.uuid
	LEFT JOIN (
		SELECT domain_uuid, COUNT(*) AS cnt_bruteforced
		FROM bruteforced
		GROUP BY domain_uuid
	) b ON b.domain_uuid = d.uuid
	WHERE d.target_uuid = ? AND d.status_code != 0`)

	if sortBy != models.SortNone {
		query = fmt.Sprintf("%s ORDER BY %s DESC", query, sortBy)
	}

	query = fmt.Sprintf("%s LIMIT %v OFFSET %v", query, limit, offset)

	rows, err := db.connection.Query(query, targetUUID)
	if err != nil {
		return models.DomainListResponse{}, fmt.Errorf("Failed to get all domain: %w", err)
	}
	defer rows.Close()

	var results models.DomainListResponse
	results.Domains = make([]models.DomainWithCount, 0)
	for rows.Next() {
		var item models.DomainWithCount
		if err := rows.Scan(&item.UUID, &item.ScanUUID, &item.URL, &item.StatusCode, &item.PortCount, &item.BruteForcedCount); err != nil {
			return models.DomainListResponse{}, fmt.Errorf("Failed to scan domain row: %w", err)
		}

		results.Domains = append(results.Domains, item)
	}

	if err := rows.Err(); err != nil {
		return models.DomainListResponse{}, fmt.Errorf("Rows error when getting domain: %w", err)
	}

	for _, domain := range results.Domains {
		ports, err := db.GetPortByDomain(domain.UUID)
		if err != nil {
			return models.DomainListResponse{}, err
		}

		domain.Ports = ports

		bruteForced, err := db.GetBruteForcedByDomain(domain.UUID, limit, 0)
		if err != nil {
			return models.DomainListResponse{}, err
		}

		domain.BruteForced = bruteForced

		results.Domains = append(results.Domains, domain)

	}

	return results, nil
}

func (db Database) InsertDomainRecord(domain models.Domain) error {
	if _, err := db.connection.Exec(
		`INSERT INTO domain (uuid, target_uuid, scan_uuid, url, status_code) VALUES (?, ?, ?, ?, ?)`,
		domain.UUID,
		domain.TargetUUID,
		domain.ScanUUID,
		domain.URL,
		domain.StatusCode,
	); err != nil {
		return fmt.Errorf("Failed to insert domain: %w", err)
	}

	return nil
}

// Delete all records from domains where status code represents an unsuccessful request
func (db Database) DeleteUnsuccessfulDomains() error {
	if _, err := db.connection.Exec(
		`DELETE FROM domain WHERE status_code = 0`,
	); err != nil {
		return fmt.Errorf("Failed to insert domain: %w", err)
	}

	return nil
}

func (db Database) UpdateDomainRecord(domain models.Domain) error {
	if _, err := db.connection.Exec(
		`UPDATE domain SET scan_uuid = ?, url = ?, status_code = ?, last_updated = ? WHERE uuid = ?`,
		domain.ScanUUID,
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
		`SELECT uuid, target_uuid, scan_uuid, url, status_code, last_updated FROM domain WHERE url = ?`,
		url).Scan(&domain.UUID, &domain.TargetUUID, &domain.ScanUUID, &domain.URL, &domain.StatusCode, &domain.LastUpdated)
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
		`INSERT INTO port (uuid, domain_uuid, port, protocol, port_state) VALUES (?, ?, ?, ?, ?)`,
		port.UUID,
		port.DomainUUID,
		port.Port,
		port.Protocol,
		port.State,
	); err != nil {
		return fmt.Errorf("Failed to insert target: %w", err)
	}

	return nil
}

func (db Database) UpdatePort(port models.Port) error {
	if _, err := db.connection.Exec(
		`UPDATE port SET port_state = ?, last_updated = ? where uuid = ?`,
		port.State,
		port.LastUpdated,
		port.UUID,
	); err != nil {
		return fmt.Errorf("Failed to insert target: %w", err)
	}

	return nil
}

func (db Database) GetPortByNumberAndDomain(portNum int, domainUUID string) (*models.Port, error) {
	var port models.Port
	err := db.connection.QueryRow(
		`SELECT uuid, domain_uuid, port, protocol, port_state, last_updated FROM port WHERE port = ? AND domain_uuid = ?`,
		portNum, domainUUID).Scan(&port.UUID, &port.DomainUUID, &port.Port, &port.Protocol, &port.State, &port.LastUpdated)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, fmt.Errorf("Failed to get port: %w", err)
	}

	return &port, nil
}

func (db Database) GetPortByDomain(domainUUID string) ([]models.Port, error) {
	rows, err := db.connection.Query("SELECT uuid, port, protocol, port_state, last_updated FROM port WHERE domain_uuid = ?", domainUUID)
	if err != nil {
		return nil, fmt.Errorf("Failed to get all ports: %w", err)
	}
	defer rows.Close()

	var results []models.Port
	for rows.Next() {
		var item models.Port
		if err := rows.Scan(&item.UUID, &item.Port, &item.Protocol, &item.State, &item.LastUpdated); err != nil {
			return nil, fmt.Errorf("Failed to scan port row: %w", err)
		}

		results = append(results, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("Rows error when getting port: %w", err)
	}

	return results, nil
}

func (db Database) InsertDaemonStats(stats models.DaemonStats) error {
	if _, err := db.connection.Exec(
		`INSERT INTO daemon_stats(uuid, total_found_urls, total_new_urls, total_found_ports, total_new_ports, scan_time, scan_begin, last_scan_ended, is_running) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		stats.UUID,
		stats.TotalFoundURLs,
		stats.TotalNewURLs,
		stats.TotalFoundPorts,
		stats.TotalNewPorts,
		stats.ScanTime,
		stats.ScanBegin,
		stats.LastScanEnded,
		stats.IsRunning,
	); err != nil {
		return fmt.Errorf("Failed to insert domain: %w", err)
	}

	return nil
}

func (db Database) UpdateDaemonStats(stats models.DaemonStats) error {
	if _, err := db.connection.Exec(
		`UPDATE daemon_stats SET uuid = ?, total_found_urls = ?, total_new_urls = ?, total_found_ports = ?, total_new_ports = ?, scan_time = ?, scan_begin = ?, last_scan_ended = ?, is_running = ?`,
		stats.UUID,
		stats.TotalFoundURLs,
		stats.TotalNewURLs,
		stats.TotalFoundPorts,
		stats.TotalNewPorts,
		stats.ScanTime,
		stats.ScanBegin,
		stats.LastScanEnded,
		stats.IsRunning,
	); err != nil {
		return fmt.Errorf("Failed to update stats: %w", err)
	}

	return nil
}

func (db Database) GetStats() ([]models.DaemonStats, error) {
	rows, err := db.connection.Query(
		`SELECT total_found_urls, total_new_urls, total_found_ports, total_new_ports, scan_time, scan_begin, last_scan_ended, is_running FROM daemon_stats`)
	if err != nil {
		return nil, fmt.Errorf("Failed to get daemon stats: %w", err)
	}
	defer rows.Close()

	var results []models.DaemonStats
	for rows.Next() {
		var statsStr models.StatsResponse
		if err := rows.Scan(
			&statsStr.TotalFoundURLs,
			&statsStr.TotalNewURLs,
			&statsStr.TotalFoundPorts,
			&statsStr.TotalNewPorts,
			&statsStr.ScanTime,
			&statsStr.ScanBegin,
			&statsStr.LastScanEnded,
			&statsStr.IsRunning,
		); err != nil {
			return nil, fmt.Errorf("Failed to get stats: %w", err)
		}

		parsedScanTime, err := time.ParseDuration(statsStr.ScanTime)
		if err != nil {
			return nil, fmt.Errorf("Failed to convert scan_time to duration: %s", statsStr.ScanTime)
		}

		layout := "2006-01-02 15:04:05.000000000-07:00"

		var parsedLastScanEnded *time.Time
		if statsStr.LastScanEnded != nil {
			lastScanEnded, err := time.Parse(layout, *statsStr.LastScanEnded)
			if err != nil {
				return nil, fmt.Errorf("Faield to convert last_scan_eded to time: %s", *statsStr.LastScanEnded)
			}
			parsedLastScanEnded = &lastScanEnded
		}

		parsedScanBegin, err := time.Parse(layout, statsStr.ScanBegin)
		if err != nil {
			return nil, fmt.Errorf("Failed to convert scan_begin to time: %s", statsStr.ScanBegin)
		}

		stats := models.DaemonStats{
			TotalFoundURLs:  statsStr.TotalFoundURLs,
			TotalNewURLs:    statsStr.TotalNewURLs,
			TotalFoundPorts: statsStr.TotalFoundPorts,
			TotalNewPorts:   statsStr.TotalNewPorts,
			ScanTime:        parsedScanTime,
			ScanBegin:       parsedScanBegin,
			LastScanEnded:   parsedLastScanEnded,
			IsRunning:       statsStr.IsRunning,
		}

		results = append(results, stats)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("Rows error when getting stats: %w", err)
	}

	return results, nil
}

func (db Database) GetBruteForcedByPath(path string, domainUUID string) (*models.BruteForced, error) {
	var bruteForced models.BruteForced
	err := db.connection.QueryRow(
		`SELECT uuid, domain_uuid, path, first_run, last_updated FROM bruteforced WHERE path = ? AND domain_uuid = ?`,
		path, domainUUID).Scan(&bruteForced.UUID, &bruteForced.DomainUUID, &bruteForced.Path, &bruteForced.FirstRun, &bruteForced.LastUpdated)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, fmt.Errorf("Failed to get bruteforced path: %w", err)
	}

	return &bruteForced, nil
}

func (db Database) GetBruteForcedByDomain(domainUUID string, limit int, offset int) ([]models.BruteForced, error) {
	query := fmt.Sprintf("SELECT uuid, domain_uuid, path, first_run, last_updated FROM bruteforced WHERE domain_uuid = ? LIMIT %v OFFSET %v", limit, offset)
	rows, err := db.connection.Query(query, domainUUID)
	if err != nil {
		return nil, fmt.Errorf("Failed to get all ports: %w", err)
	}
	defer rows.Close()

	var results []models.BruteForced
	for rows.Next() {
		var item models.BruteForced
		if err := rows.Scan(&item.UUID, &item.DomainUUID, &item.Path, &item.FirstRun, &item.LastUpdated); err != nil {
			return nil, fmt.Errorf("Failed to scan bruteforced path row: %w", err)
		}

		results = append(results, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("Rows error when getting bruteforced paths: %w", err)
	}

	return results, nil
}

func (db Database) GetBruteForcedCountByDomain(domainUUID string) (int, error) {

	var count int
	err := db.connection.QueryRow("SELECT COUNT(path) FROM bruteforced WHERE domain_uuid = ?", domainUUID).Scan(&count)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, fmt.Errorf("Failed to get bruteforced count by domain: %w", err)
	}

	return count, nil
}

func (db Database) InsertBruteForced(bruteForced models.BruteForced) error {
	if _, err := db.connection.Exec(
		`INSERT INTO bruteforced(uuid, domain_uuid, path, first_run, last_updated) VALUES ( ?, ?, ?, ?, ?)`,
		bruteForced.UUID,
		bruteForced.DomainUUID,
		bruteForced.Path,
		bruteForced.FirstRun,
		bruteForced.LastUpdated,
	); err != nil {
		return fmt.Errorf("Failed to insert bruteforced paths: %w", err)
	}

	return nil
}

func (db Database) UpdateBruteForced(bruteForced models.BruteForced) error {
	if _, err := db.connection.Exec(
		`UPDATE bruteforced SET domain_uuid = ?, path = ?, first_run = ?, last_updated = ? WHERE uuid = ?`,
		bruteForced.DomainUUID,
		bruteForced.Path,
		bruteForced.FirstRun,
		bruteForced.LastUpdated,
		bruteForced.UUID,
	); err != nil {
		return fmt.Errorf("Failed to update bruteforced paths: %w", err)
	}

	return nil
}
