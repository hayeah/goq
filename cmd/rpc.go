package cmd

import (
	"net/rpc"
)

var client *rpc.Client

func GetClient() (*rpc.Client, error) {
	client, err := rpc.Dial("unix", "./goq.socket")
	if err != nil {
		return nil, err
	}
	return client, nil
}
