package cmd

import (
	"github.com/hayeah/goq"
)

func Server(argv []string) error {
	return goq.StartServer()
}
