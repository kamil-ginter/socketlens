package store

import (
	"database/sql"
	"os"
	"path/filepath"
	"time"

	"github.com/kamil-ginter/socketlens/internal/models"
	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

type Snapshot struct {
	OpenPorts      []int
	RequestedPorts []int
}

func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	store := &Store{db: db}
	if err := store.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}

	return store, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS scans (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			target TEXT NOT NULL,
			resolved_ip TEXT NOT NULL,
			started_at TEXT NOT NULL,
			duration_ms INTEGER NOT NULL,
			requested_ports INTEGER NOT NULL
		);

		CREATE TABLE IF NOT EXISTS scan_ports (
			scan_id INTEGER NOT NULL,
			port INTEGER NOT NULL,
			service TEXT NOT NULL,
			PRIMARY KEY(scan_id, port),
			FOREIGN KEY(scan_id) REFERENCES scans(id) ON DELETE CASCADE
		);

		CREATE TABLE IF NOT EXISTS scan_requested_ports (
			scan_id INTEGER NOT NULL,
			port INTEGER NOT NULL,
			PRIMARY KEY(scan_id, port),
			FOREIGN KEY(scan_id) REFERENCES scans(id) ON DELETE CASCADE
		);

		CREATE INDEX IF NOT EXISTS idx_scans_target
		ON scans(target, id DESC);
	`)
	return err
}

func (s *Store) LatestOpenPorts(target string) ([]int, bool, error) {
	var scanID int64

	err := s.db.QueryRow(`
		SELECT id
		FROM scans
		WHERE target = ?
		ORDER BY id DESC
		LIMIT 1
	`, target).Scan(&scanID)

	if err == sql.ErrNoRows {
		return []int{}, false, nil
	}
	if err != nil {
		return nil, false, err
	}

	rows, err := s.db.Query(`
		SELECT port
		FROM scan_ports
		WHERE scan_id = ?
		ORDER BY port
	`, scanID)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	ports := make([]int, 0)

	for rows.Next() {
		var port int
		if err := rows.Scan(&port); err != nil {
			return nil, false, err
		}
		ports = append(ports, port)
	}

	return ports, true, rows.Err()
}

func (s *Store) LatestSnapshot(target string) (Snapshot, bool, error) {
	var scanID int64

	err := s.db.QueryRow(`
		SELECT id
		FROM scans
		WHERE target = ?
		ORDER BY id DESC
		LIMIT 1
	`, target).Scan(&scanID)

	if err == sql.ErrNoRows {
		return Snapshot{
			OpenPorts:      []int{},
			RequestedPorts: []int{},
		}, false, nil
	}
	if err != nil {
		return Snapshot{}, false, err
	}

	openPorts, err := s.readPortList(`
		SELECT port
		FROM scan_ports
		WHERE scan_id = ?
		ORDER BY port
	`, scanID)
	if err != nil {
		return Snapshot{}, false, err
	}

	requestedPorts, err := s.readPortList(`
		SELECT port
		FROM scan_requested_ports
		WHERE scan_id = ?
		ORDER BY port
	`, scanID)
	if err != nil {
		return Snapshot{}, false, err
	}

	return Snapshot{
		OpenPorts:      openPorts,
		RequestedPorts: requestedPorts,
	}, true, nil
}

func (s *Store) readPortList(query string, scanID int64) ([]int, error) {
	rows, err := s.db.Query(query, scanID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ports := make([]int, 0)

	for rows.Next() {
		var port int
		if err := rows.Scan(&port); err != nil {
			return nil, err
		}
		ports = append(ports, port)
	}

	return ports, rows.Err()
}

func (s *Store) SaveRequestedPorts(scanID int64, ports []int) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM scan_requested_ports WHERE scan_id = ?`, scanID); err != nil {
		return err
	}

	for _, port := range ports {
		if _, err := tx.Exec(`
			INSERT INTO scan_requested_ports(scan_id, port)
			VALUES(?, ?)
		`, scanID, port); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *Store) SaveScan(result *models.ScanResult) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	insert, err := tx.Exec(`
		INSERT INTO scans(
			target,
			resolved_ip,
			started_at,
			duration_ms,
			requested_ports
		)
		VALUES(?, ?, ?, ?, ?)
	`,
		result.Target,
		result.ResolvedIP,
		result.StartedAt.Format(time.RFC3339Nano),
		result.DurationMs,
		result.RequestedPorts,
	)
	if err != nil {
		return err
	}

	id, err := insert.LastInsertId()
	if err != nil {
		return err
	}
	result.ID = id

	for _, item := range result.OpenPorts {
		if _, err := tx.Exec(`
			INSERT INTO scan_ports(scan_id, port, service)
			VALUES(?, ?, ?)
		`, id, item.Port, item.Service); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *Store) History(limit int) ([]models.HistoryItem, error) {
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	rows, err := s.db.Query(`
		SELECT
			s.id,
			s.target,
			s.resolved_ip,
			s.started_at,
			s.duration_ms,
			COUNT(p.port) AS open_count
		FROM scans s
		LEFT JOIN scan_ports p ON p.scan_id = s.id
		GROUP BY s.id
		ORDER BY s.id DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]models.HistoryItem, 0)

	for rows.Next() {
		var item models.HistoryItem
		var startedAt string

		if err := rows.Scan(
			&item.ID,
			&item.Target,
			&item.ResolvedIP,
			&startedAt,
			&item.DurationMs,
			&item.OpenCount,
		); err != nil {
			return nil, err
		}

		item.StartedAt, err = time.Parse(time.RFC3339Nano, startedAt)
		if err != nil {
			return nil, err
		}

		items = append(items, item)
	}

	return items, rows.Err()
}
