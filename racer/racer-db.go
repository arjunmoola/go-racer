package racer

import (
	"database/sql"
	_ "github.com/marcboeker/go-duckdb/v2"
	"path/filepath"
	"errors"
)

const driverName = "duckdb"

var defaultDbPath = filepath.Join(defaultConfigDir, "racer.db")

const createRacerTestIdSeq = `
	CREATE SEQUENCE IF NOT EXISTS seq_test_id START 1;
`

const createGameStatsTableQuery = `
	CREATE TABLE IF NOT EXISTS game_stats(
		id INTEGER PRIMARY KEY CHECK (id = 1),
		total INTEGER,
		total_completed INTEGER,
		total_attempted INTEGER,
		last_test_id INTEGER
	)
`

const createTestsTableQuery = `
	CREATE TABLE IF NOT EXISTS all_tests(
		id INTEGER PRIMARY KEY DEFAULT nextval('seq_test_id'),
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		test_name VARCHAR,
		test_duration INTEGER,
		target VARCHAR,
		input VARCHAR
	)
`

type RacerTestInsertParams struct {
	testName string
	testDuration int
	target string
	input string
}

type GameStatsInsertParams struct {
	total int
	totalCompleted int
	totalAttempted int
	lastTestId int
}

func SetupDB(path string) (*sql.DB, error) {
	db, err := sql.Open(driverName, defaultDbPath)

	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	_, err = db.Exec(createGameStatsTableQuery)

	if err != nil {
		return nil, err
	}

	_, err = db.Exec(createRacerTestIdSeq)

	if err != nil {
		return nil, err
	}

	_, err = db.Exec(createTestsTableQuery)

	if err != nil {
		return nil, err
	}

	return db, nil
}

func GetGameStats(db *sql.DB) (*GameStats, error) {
	query := "SELECT total, total_completed, total_attempted, last_test_id FROM game_stats where id = 1"
	row := db.QueryRow(query)
	stats := GameStats{}

	var rowNotFound bool

	err := row.Scan(&stats.Total, &stats.TotalCompleted, &stats.TotalAttempted, &stats.LastTestId)

	if err != nil {
		if errors.Is(err,sql.ErrNoRows) {
			rowNotFound = true
		} else {
			return nil, err
		}
	}

	if rowNotFound {
		insertQuery := "INSERT INTO game_stats VALUES(1, ?, ?, ?, ?)"

		if _, err := db.Exec(insertQuery, stats.Total, stats.TotalCompleted, stats.TotalAttempted, stats.LastTestId); err != nil {
			return nil, err
		}
	}

	return &stats, nil
}

func UpdateGameStats(db *sql.DB, stats *GameStats) error {
	query := "UPDATE game_stats SET total = ?, total_completed = ?, total_attempted = ?, last_test_id = ? where id = 1"

	_, err := db.Exec(query, stats.Total, stats.TotalCompleted, stats.TotalAttempted, stats.LastTestId)

	if err != nil {
		return err
	}

	return nil
}

func UpdateGameStatsTx(tx *sql.Tx, stats *GameStats) error {
	query := "UPDATE game_stats SET total = ?, total_completed = ?, total_attempted = ?, last_test_id = ? where id = 1"

	_, err := tx.Exec(query, stats.Total, stats.TotalCompleted, stats.TotalAttempted, stats.LastTestId)

	if err != nil {
		return err
	}

	return nil
}

func InsertRacerTest(db *sql.DB, test *RacerTest) error {
	query := "INSERT INTO all_tests (test_name, test_duration, target, input) VALUES(?, ?, ?, ?)"

	_, err := db.Exec(query, test.Test, test.Time, test.Target, test.Input)

	if err != nil {
		return err
	}

	return nil
}

func InsertRacerTestTx(tx *sql.Tx, test *RacerTest) error {
	query := "INSERT INTO all_tests (test_name, test_duration, target, input) VALUES(?, ?, ?, ?)"

	_, err := tx.Exec(query, test.Test, test.Time, test.Target, test.Input)

	if err != nil {
		return err
	}

	return nil
}

func GetTotalNumberOfTests(db *sql.DB) (int, error) {
	query := "SELECT count(*) FROM all_tests"

	row := db.QueryRow(query)

	var count int

	if err := row.Scan(&count); err != nil {
		return 0, err
	}

	return count, nil
}

func GetAllTests(db *sql.DB) ([]*RacerTest, error) {
	query := "SELECT id, test_name, test_duration, target, input FROM all_tests ORDER BY id DESC LIMIT 100"

	rows, err := db.Query(query)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var tests []*RacerTest

	for rows.Next() {
		test := RacerTest{}

		err := rows.Scan(&test.Id, &test.Test, &test.Time, &test.Target, &test.Input)

		if err != nil {
			return nil, err
		}

		tests = append(tests, &test)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tests, nil
}
