package main

import (
	"github.com/hayeah/goq"
	"log"
	"net/rpc"
	"os"
)

func main() {
	mode := os.Args[1]

	var err error
	switch mode {
	case "server":
		err = server()
	case "q", "queue":
		if len(os.Args) < 2 {
			log.Fatal("go queue <cmd> arg ...")
		} else {
			err = queue(os.Args[2], os.Args[2:]...)
		}
	}
	if err != nil {
		log.Fatal(err)
	}

}

func server() error {
	return goq.StartServer()
}

func queue(cmd string, args ...string) error {
	client, err := rpc.Dial("unix", "./goq.socket")
	if err != nil {
		return err
	}

	task := goq.NewTaskWithEnv(cmd, args...)

	qArgs := &goq.RPCQueueArgs{Task: *task}
	var id int64
	err = client.Call("Server.Queue", qArgs, &id)
	if err != nil {
		return err
	}

	log.Printf("new task id: %d\n", id)
	return nil
}
