package cmd

import (
	"log"
	"strconv"
	"syscall"

	"github.com/hayeah/goq"
)

func Stop(argv []string) error {
	var err error
	taskId, err := strconv.ParseInt(argv[0], 10, 64)
	if err != nil {
		return err
	}
	args := &goq.RPCStopArgs{
		TaskId: taskId,
		Signal: syscall.SIGINT,
	}

	client, err := GetClient()
	if err != nil {
		return err
	}

	err = client.Call("Server.Stop", args, &taskId)
	if err != nil {
		return err
	}

	log.Printf("Sent %v to task: %d", args.Signal, taskId)
	return nil
}
