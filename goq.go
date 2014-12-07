package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"os"
	"os/exec"
	"strings"
)

type TaskState int

const (
	TaskWaiting TaskState = iota
	TasKError
	TaskSuccess
	TaskRunning
	TaskStopped
)

var TaskStateNames = [...]string{
	TaskWaiting: "waiting",
	TasKError:   "error",
	TaskSuccess: "success",
	TaskRunning: "running",
	TaskStopped: "stopped",
}

func (t TaskState) String() string {
	return TaskStateNames[t]
}

type Task struct {
	Env   []string
	Cwd   string
	Cmd   []string
	state TaskState
}

func (t *Task) Run() error {
	log.Printf("running: %v\n", t.Cmd)
	cmd := exec.Command(t.Cmd[0], t.Cmd[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = t.Env
	cmd.Dir = t.Cwd
	err := cmd.Run()
	return err
}

func main() {
	setupDb()
	if len(os.Args) > 1 {
		mode := os.Args[1]
		switch mode {
		case "q", "queue":
			queueCommand()
		default:
			log.Fatal("goq [queue | list | stop | retry] ...")
		}
	} else {
		queueServer()
	}
}

func queueCommand() {
	db, err := sql.Open("sqlite3", "./goq.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	cwd, err := os.Getwd()
	task := &Task{
		Env:   os.Environ(),
		Cwd:   cwd,
		Cmd:   os.Args[2:],
		state: TaskWaiting,
	}

	var buf []byte
	w := bytes.NewBuffer(buf)
	encoder := json.NewEncoder(w)
	err = encoder.Encode(task)

	if err != nil {
		log.Fatal(err)
	}

	taskJSON := w.String()
	log.Printf("json: %s\n %s", taskJSON, task.state)
	_, err = db.Exec("insert into tasks(task,state) values(?,?) ", taskJSON, task.state.String())
	if err != nil {
		log.Fatal(err)
	}
}

func queueServer() {
	var err error
	db, err := sql.Open("sqlite3", "./goq.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	rows, err := db.Query("select id, task, state from tasks order by id asc;")
	if err != nil {
		log.Fatal(err)
	}
	for rows.Next() {
		var id int
		var state string
		var taskJSON string
		rows.Scan(&id, &taskJSON, &state)
		r := strings.NewReader(taskJSON)
		decoder := json.NewDecoder(r)
		var task Task
		err = decoder.Decode(&task)
		if err != nil {
			log.Println(err)
			continue
		}
		rows.Close()

		err = task.Run()
		if err != nil {
			log.Println(err)
		} else {
			_, err := db.Exec("update tasks set state=? where id=?", TaskSuccess.String(), id)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}

func setupDb() {
	db, err := sql.Open("sqlite3", "./goq.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	sqlStmt := `
  create table if not exists tasks (id integer not null primary key, task text, state text);
  `
	_, err = db.Exec(sqlStmt)
	if err != nil {
		log.Fatal(err)
	}
}
