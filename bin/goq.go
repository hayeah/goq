package main

import (
	"log"
	"os"

	"github.com/hayeah/goq/cmd"
)

type subcommand func(argv []string) error

var subcommands = map[string]subcommand{
	"add":     cmd.Add,
	"queue":   cmd.Add,
	"start":   cmd.Server,
	"list":    cmd.List,
	"stop":    cmd.Stop,
	"workers": cmd.Workers,
}

const Usage = `
goq - A probably hazardous queue server.
  start - start the queue server
  queue - same as "start"
  add - queue a task

goq <cmd> -h for more details.
`

func main() {
	var err error
	var argv []string
	if len(os.Args) < 2 {
		err = cmd.Server(argv)
		log.Fatalln(err)
	}
	mode := os.Args[1]

	subcmd, ok := subcommands[mode]

	if !ok {
		log.Fatalf("Command not recognized: %s\n", mode)
	}

	argv = os.Args[2:]
	err = subcmd(argv)

	if err != nil {
		log.Fatal(err)
	}

}
