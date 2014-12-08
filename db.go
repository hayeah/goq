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

	SQLUpdateTask = `update tasks set blob=?,state=? where id=?`

	SQLFindTask = `select id, blob from tasks where id=?`

	SQLTasksInState = `select id, blob from tasks where state=? order by id desc`
)

// When creating new record, will set task's id.
func (d *DB) Save(t *Task) error {
	// Treat 0 as null
	id := sql.NullInt64{
		Int64: t.Id,
		Valid: t.Id != 0,
	}

	var sql string
	if !id.Valid {
		sql = SQLInsertTask
	} else {
		sql = SQLUpdateTask
	}

	if !id.Valid {
		r, err := d.Exec(sql, id, t.ToJSON(), t.State.String())
		if err != nil {
			return err
		}
		taskId, err := r.LastInsertId()
		if err != nil {
			return err
		}
		t.Id = taskId
	} else {
		_, err := d.Exec(sql, t.ToJSON(), t.State.String(), id)
		if err != nil {
			return err
		}
	}

	return nil
}

func (d *DB) NextTaskIn(state TaskState) (task *Task, err error) {
	rows, err := d.Query(SQLTasksInState, state.String())
	if err != nil {
		return
	}
	defer rows.Close()

	var blob string
	var id int64

	if rows.Next() {
		err = rows.Scan(&id, &blob)
		if err != nil {
			return
		}
		task, err = TaskFromJSON(blob)
		if err != nil {
			return
		}
		task.Id = id

		return task, nil
	}

	return nil, rows.Err()
}
