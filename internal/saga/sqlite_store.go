package saga

import (
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/danielalmeidafarias/saga-pattern/pkg"
	"github.com/danielalmeidafarias/saga-pattern/pkg/db"
	"github.com/danielalmeidafarias/saga-pattern/pkg/msg"
)

type SQLiteStore struct{ database *sql.DB }

func NewSQLiteStore(database *sql.DB) (*SQLiteStore, error) {
	if err := db.Migrate(database,
		`CREATE TABLE IF NOT EXISTS sagas (id TEXT PRIMARY KEY, order_id TEXT NOT NULL, trigger TEXT NOT NULL, status TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS saga_steps (id TEXT PRIMARY KEY, saga_id TEXT NOT NULL, phase INTEGER NOT NULL, status TEXT NOT NULL, command BLOB NOT NULL, compensation BLOB, result BLOB, FOREIGN KEY (saga_id) REFERENCES sagas(id))`,
		`CREATE TABLE IF NOT EXISTS saga_outbox (id TEXT PRIMARY KEY, message BLOB NOT NULL, published_at TEXT)`,
	); err != nil {
		return nil, err
	}
	return &SQLiteStore{database: database}, nil
}

func (s *SQLiteStore) Save(saga *Saga) *pkg.Error {
	tx, err := s.database.Begin()
	if err != nil {
		return storeError("begin saving saga", err)
	}
	result, err := tx.Exec(`INSERT OR IGNORE INTO sagas (id, order_id, trigger, status) VALUES (?, ?, ?, ?)`, saga.ID, saga.OrderID, saga.Trigger, saga.Status)
	if err != nil {
		_ = tx.Rollback()
		return storeError("save saga", err)
	}
	inserted, err := result.RowsAffected()
	if err != nil {
		_ = tx.Rollback()
		return storeError("check saved saga", err)
	}
	if inserted == 0 {
		_ = tx.Rollback()
		return nil
	}
	for i := range saga.StepList {
		step := &saga.StepList[i]
		command, marshalErr := json.Marshal(step.Command)
		if marshalErr != nil {
			_ = tx.Rollback()
			return storeError("serialize saga command", marshalErr)
		}
		var compensation []byte
		if step.Compensation != nil {
			compensation, marshalErr = json.Marshal(step.Compensation)
			if marshalErr != nil {
				_ = tx.Rollback()
				return storeError("serialize saga compensation", marshalErr)
			}
		}
		if _, err := tx.Exec(`INSERT INTO saga_steps (id, saga_id, phase, status, command, compensation, result) VALUES (?, ?, ?, ?, ?, ?, ?)`, step.ID, saga.ID, step.Phase, step.Status, command, compensation, step.Result); err != nil {
			_ = tx.Rollback()
			return storeError("save saga step", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return storeError("commit saga", err)
	}
	return nil
}

func (s *SQLiteStore) FindByID(id string) (*Saga, *pkg.Error) {
	var saga Saga
	if err := s.database.QueryRow(`SELECT id, order_id, trigger, status FROM sagas WHERE id = ?`, id).Scan(&saga.ID, &saga.OrderID, &saga.Trigger, &saga.Status); err != nil {
		return nil, storeError("find saga", err)
	}
	rows, err := s.database.Query(`SELECT id, saga_id, phase, status, command, compensation, result FROM saga_steps WHERE saga_id = ? ORDER BY phase, id`, id)
	if err != nil {
		return nil, storeError("find saga steps", err)
	}
	defer rows.Close()
	for rows.Next() {
		step, scanErr := scanStep(rows)
		if scanErr != nil {
			return nil, storeError("scan saga step", scanErr)
		}
		saga.StepList = append(saga.StepList, step)
	}
	if err := rows.Err(); err != nil {
		return nil, storeError("read saga steps", err)
	}
	return &saga, nil
}

func (s *SQLiteStore) Update(saga *Saga) *pkg.Error {
	result, err := s.database.Exec(`UPDATE sagas SET order_id = ?, trigger = ?, status = ? WHERE id = ?`, saga.OrderID, saga.Trigger, saga.Status, saga.ID)
	return changed(result, err, "update saga")
}

func (s *SQLiteStore) UpdateResult(saga *Saga, step *SagaStep) *pkg.Error {
	tx, err := s.database.Begin()
	if err != nil {
		return storeError("begin result update", err)
	}
	if _, err := tx.Exec(`UPDATE saga_steps SET status = ?, result = ? WHERE id = ?`, step.Status, step.Result, step.ID); err != nil {
		_ = tx.Rollback()
		return storeError("update saga step result", err)
	}
	if _, err := tx.Exec(`UPDATE sagas SET status = ? WHERE id = ?`, saga.Status, saga.ID); err != nil {
		_ = tx.Rollback()
		return storeError("update saga result status", err)
	}
	if err := tx.Commit(); err != nil {
		return storeError("commit result update", err)
	}
	return nil
}

func (s *SQLiteStore) GetAll(filter GetAllSagaFilter) ([]Saga, *pkg.Error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}
	query := `SELECT id FROM sagas`
	args := []any{}
	if filter.Status != nil {
		query += ` WHERE status = ?`
		args = append(args, *filter.Status)
	}
	query += ` ORDER BY id LIMIT ?`
	args = append(args, limit)
	rows, err := s.database.Query(query, args...)
	if err != nil {
		return nil, storeError("list sagas", err)
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			_ = rows.Close()
			return nil, storeError("scan saga id", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Close(); err != nil {
		return nil, storeError("close saga list", err)
	}
	var sagas []Saga
	for _, id := range ids {
		saga, findErr := s.FindByID(id)
		if findErr != nil {
			return nil, findErr
		}
		sagas = append(sagas, *saga)
	}
	return sagas, nil
}

func (s *SQLiteStore) FindStepByID(id string) (*SagaStep, *pkg.Error) {
	row := s.database.QueryRow(`SELECT id, saga_id, phase, status, command, compensation, result FROM saga_steps WHERE id = ?`, id)
	step, err := scanStep(row)
	if err != nil {
		return nil, storeError("find saga step", err)
	}
	return &step, nil
}

func (s *SQLiteStore) UpdateStep(step *SagaStep) *pkg.Error {
	result, err := s.database.Exec(`UPDATE saga_steps SET status = ?, result = ? WHERE id = ?`, step.Status, step.Result, step.ID)
	return changed(result, err, "update saga step")
}

func (s *SQLiteStore) Enqueue(message msg.Message) *pkg.Error {
	encoded, err := json.Marshal(message)
	if err != nil {
		return storeError("serialize outbox message", err)
	}
	_, err = s.database.Exec(`INSERT OR IGNORE INTO saga_outbox (id, message) VALUES (?, ?)`, message.ID, encoded)
	if err != nil {
		return storeError("enqueue outbox message", err)
	}
	return nil
}

func (s *SQLiteStore) Pending(limit int) ([]msg.Message, *pkg.Error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.database.Query(`SELECT message FROM saga_outbox WHERE published_at IS NULL ORDER BY rowid LIMIT ?`, limit)
	if err != nil {
		return nil, storeError("list outbox messages", err)
	}
	defer rows.Close()
	var messages []msg.Message
	for rows.Next() {
		var encoded []byte
		var message msg.Message
		if err := rows.Scan(&encoded); err != nil {
			return nil, storeError("scan outbox message", err)
		}
		if err := json.Unmarshal(encoded, &message); err != nil {
			return nil, storeError("decode outbox message", err)
		}
		messages = append(messages, message)
	}
	return messages, nil
}

func (s *SQLiteStore) MarkPublished(id string) *pkg.Error {
	result, err := s.database.Exec(`UPDATE saga_outbox SET published_at = CURRENT_TIMESTAMP WHERE id = ?`, id)
	return changed(result, err, "mark outbox message published")
}

type rowScanner interface{ Scan(dest ...any) error }

func scanStep(row rowScanner) (SagaStep, error) {
	var step SagaStep
	var command, compensation, result []byte
	if err := row.Scan(&step.ID, &step.SagaID, &step.Phase, &step.Status, &command, &compensation, &result); err != nil {
		return step, err
	}
	if err := json.Unmarshal(command, &step.Command); err != nil {
		return step, err
	}
	if len(compensation) > 0 {
		step.Compensation = &msg.Message{}
		if err := json.Unmarshal(compensation, step.Compensation); err != nil {
			return step, err
		}
	}
	step.Result = append(step.Result, result...)
	return step, nil
}

func changed(result sql.Result, err error, operation string) *pkg.Error {
	if err != nil {
		return storeError(operation, err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return storeError(operation, err)
	}
	if count == 0 {
		return storeError(operation, sql.ErrNoRows)
	}
	return nil
}

func storeError(operation string, err error) *pkg.Error {
	code := pkg.ErrorCode("STORE_ERROR")
	if errors.Is(err, sql.ErrNoRows) {
		code = "NOT_FOUND"
	}
	return pkg.NewError(code, operation, err)
}
