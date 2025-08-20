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
		test_size INTEGER,
		accuracy DOUBLE,
		mode VARCHAR,
		allow_backspace BOOLEAN,
		target VARCHAR,
		input VARCHAR,
		wpm INTEGER,
		cps INTEGER,
		rle VARCHAR,
		raw_input VARCHAR,
		sample_rate INTEGER,
		acc_samples DOUBLE[],
		cps_samples INTEGER[],
		wpm_samples INTEGER[]
	)
`

const createPlayerInfoTableQuery =`
	CREATE TABLE IF NOT EXISTS player_info(
		id INTEGER PRIMARY KEY CHECK (id = 1),
		name VARCHAR NOT NULL,
		level INTEGER,
		max_hp INTEGER,
		cur_hp INTEGER,
		wpm INTEGER,
		bosses_defeated INTEGER
	)
`

type RacerTestInsertParams struct {
	testName string
	testDuration int
	testSize int
	accuracry float64
	mode string
	allowBackspace bool
	target string
	input string
	wpm int
	cps int
}

type GameStatsInsertParams struct {
	total int
	totalCompleted int
	totalAttempted int
	lastTestId int
}

type PlayerInfoInsertParams struct {
	name string
	level int
	maxHp int
	curHp int
	wpm int
	bossesDefeated int
}

type RacerTest struct {
	Id int
	Test string
	Time int
	TestSize int
	Accuracy float64
	Mode string
	AllowBackspace bool
	Target string
	Input string
	Wpm int
	Cps int
	Rle string
	RawInput string
	SampleRate int
	AccList []float64
	CpsList []int
	WpmList []int
}

type PlayerInfo struct {
	name string
	level int
	maxHp int
	curHp int
	wpm int
	bossesDefeated int
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

	_, err = db.Exec(createPlayerInfoTableQuery)

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

func GetPlayerInfo(db *sql.DB) (*PlayerInfo, bool, error) {
	query := "SELECT name, level, max_hp, cur_hp, wpm, bosses_defeated FROM player_info WHERE id = 1"

	row := db.QueryRow(query)
	info := PlayerInfo{}

	err := row.Scan(
		&info.name,
		&info.level,
		&info.maxHp,
		&info.curHp,
		&info.wpm,
		&info.bossesDefeated,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, nil
		} else {
			return nil, false, err
		}
	}

	return &info, true, nil
}

func InsertPlayerInfo(db *sql.DB, params *PlayerInfoInsertParams) error {
	query := "INSERT INTO player_info VALUES(1, ?, ?, ?, ?, ?, ?)"

	_, err := db.Exec(
		query,
		params.name,
		params.level,
		params.maxHp,
		params.curHp,
		params.wpm,
		params.bossesDefeated,
	)

	if err != nil {
		return err
	}

	return nil
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

func prepareStatement(s string, db *sql.DB) (*sql.Stmt, error) {
	stmt, err := db.Prepare(s)

	if err != nil {
		return nil, err
	}

	return stmt, nil
}

const insertTestStmtStr = "INSERT INTO all_tests (test_name, test_duration, test_size, accuracy, mode, allow_backspace, target, input, wpm, cps, rle, raw_input, sample_rate, acc_samples, cps_samples, wpm_samples) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"

func InsertRacerTestStmt(stmt *sql.Stmt, test *RacerTest) error {
	_, err := stmt.Exec(
		test.Test,
		test.Time,
		test.TestSize,
		test.Accuracy,
		test.Mode,
		test.AllowBackspace,
		test.Target,
		test.Input,
		test.Wpm,
		test.Cps,
		test.Rle,
		test.RawInput,
		test.SampleRate,
		test.AccList,
		test.CpsList,
		test.WpmList,
	)

	if err != nil {
		return err
	}

	return nil
}

func InsertRacerTestTx(tx *sql.Tx, test *RacerTest) error {
	query := "INSERT INTO all_tests (test_name, test_duration, test_size, accuracy, mode, allow_backspace, target, input, wpm, cps) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"

	_, err := tx.Exec(query, test.Test, test.Time, test.TestSize, test.Accuracy, test.Mode, test.AllowBackspace, test.Target, test.Input, test.Wpm, test.Cps)

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

const getAllTestsQueryStr = `
	SELECT
		id, test_name, test_duration,
		test_size, accuracy, mode,
		allow_backspace, target, input,
		wpm, cps, rle, raw_input
	FROM all_tests
	ORDER BY id DESC
	LIMIT 100
	`

func GetAllTests(stmt *sql.Stmt) ([]*RacerTest, error) {
	rows, err := stmt.Query()

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var tests []*RacerTest

	for rows.Next() {
		test := RacerTest{}

		err := rows.Scan(
			&test.Id,
			&test.Test,
			&test.Time,
			&test.TestSize,
			&test.Accuracy,
			&test.Mode,
			&test.AllowBackspace,
			&test.Target,
			&test.Input,
			&test.Wpm,
			&test.Cps,
			&test.Rle,
			&test.RawInput,
		)

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
