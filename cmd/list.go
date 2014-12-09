package cmd

import (
	"fmt"
	"log"

	"github.com/hayeah/goq"
)

func List(argv []string) error {
	var err error
	client, err := GetClient()
	if err != nil {
		return err
	}

	var state goq.TaskState
	if len(argv) == 0 {
		state = goq.TaskWaiting
	} else {
		switch argv[0] {
		case "success":
			state = goq.TaskSuccess
		case "error":
			state = goq.TaskError
		case "running":
			state = goq.TaskRunning
		default:
			log.Fatalf("Unknown task state: %s\n", argv[0])
		}
	}

	args := &goq.RPCListArgs{
		State: state,
	}

	var tasks []goq.Task
	err = client.Call("Server.List", args, &tasks)
	if err != nil {
		return err
	}

	for _, task := range tasks {
		fmt.Printf("%d %s %v\n", task.Id, task.Cmd, task.Args)
	}

	// log.Printf("tasks(state=%s): %#v\n", args.State.String(), tasks)
	return nil
}
