package cmd

import (
	"fmt"
	"github.com/hayeah/goq"
	"log"
	"os"
)

const AddUsage = `
goq add cmd arg [arg2] [arg] ...
`

func Add(argv []string) error {
	if len(argv) < 1 {
		fmt.Fprintln(os.Stderr, AddUsage)
		os.Exit(1)
	}

	cmd := argv[0]
	args := argv[1:]

	task, err := goq.NewTaskWithEnv(cmd, args...)
	if err != nil {
		log.Fatal(err)
	}

	qArgs := &goq.RPCQueueArgs{Task: *task}
	var id int64

	client, err := GetClient()
	if err != nil {
		return err
	}

	err = client.Call("Server.Queue", qArgs, &id)
	if err != nil {
		return err
	}

	fmt.Println(id)
	return nil
}
