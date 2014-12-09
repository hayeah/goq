package goq

import (
	"bytes"
	"encoding/json"
	"log"
	"os"
	"os/exec"
	"strings"
)

type TaskState int

const (
	TaskWaiting TaskState = iota
	TaskError
	TaskSuccess
	TaskRunning
	TaskStopped
)

var TaskStateNames = [...]string{
	TaskWaiting: "waiting",
	TaskError:   "error",
	TaskSuccess: "success",
	TaskRunning: "running",
	TaskStopped: "stopped",
}

func (t TaskState) String() string {
	return TaskStateNames[t]
}

func NewTaskWithEnv(cmd string, args ...string) (*Task, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	exe, err := exec.LookPath(cmd)
	if err != nil {
		return nil, err
	}

	return &Task{
		Env:   os.Environ(),
		Cwd:   cwd,
		Cmd:   exe,
		Args:  args,
		State: TaskWaiting,
	}, nil
}

type Task struct {
	Id    int64 `json:"-"`
	Env   []string
	Cwd   string
	Cmd   string
	Args  []string
	State TaskState
}

func (t *Task) Run() error {
	log.Printf("running: %v %v\n", t.Cmd, t.Args)

	cmd := exec.Command(t.Cmd, t.Args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = t.Env
	cmd.Dir = t.Cwd
	err := cmd.Run()
	log.Printf("exits: %v %v\n", t.Cmd, t.Args)
	return err
}

func (t *Task) ToJSON() string {
	var buf []byte
	w := bytes.NewBuffer(buf)
	encoder := json.NewEncoder(w)
	err := encoder.Encode(t)
	if err != nil {
		log.Fatal(err)
	}

	return w.String()
}

func TaskFromJSON(blob string) (*Task, error) {
	r := strings.NewReader(blob)
	decoder := json.NewDecoder(r)
	var task Task
	err := decoder.Decode(&task)
	return &task, err
}
