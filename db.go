package goq

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	*sql.DB
	Path string
}

func OpenDB(path string) (*DB, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(SQLCreateTasksTable)

	return &DB{
		DB:   db,
		Path: path,
	}, nil
}

const (
	SQLCreateTasksTable = `
create table if not exists tasks (
  id integer not null primary key,
  blob text,
  state text
);`
	SQLInsertTask = `insert into tasks (id,blob,state) values (?,?,?)`

	SQLFindTask = `select blob from tasks where id=?`
)

// When creating new record, will set task's id.
func (d *DB) Save(t *Task) error {
	// Treat 0 as null
	id := sql.NullInt64{
		Int64: t.Id,
		Valid: t.Id != 0,
	}

	r, err := d.Exec(SQLInsertTask, id, t.ToJSON(), t.State.String())
	if !id.Valid {
		// Find the last insert id.
		taskId, err := r.LastInsertId()
		if err != nil {
			return err
		}
		t.Id = taskId
	}

	return err
}

func (d *DB) Find(id uint64) (*Task, error) {
	row := d.QueryRow(SQLFindTask, id)
	var blob string
	err := row.Scan(&blob)
	if err != nil {
		return nil, err
	}

	return TaskFromJSON(blob)
}

// save task

// Store task as JSON blob.
// Map selected fields to column for querying and indexing.
