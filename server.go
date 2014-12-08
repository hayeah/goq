package goq

import (
	"log"
	"net"
	"net/rpc"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

type Server struct {
	db       *DB
	moreWork *sync.Cond
}

const (
	db_path     = "./goq.db"
	socket_path = "./goq.socket"
)

// server loop
func StartServer() error {
	db, err := OpenDB(db_path)
	if err != nil {
		return err
	}
	var lock sync.Mutex

	server := &Server{
		db:       db,
		moreWork: sync.NewCond(&lock),
	}

	so, err := net.Listen("unix", socket_path)
	if err != nil {
		return err
	}

	sigc := make(chan os.Signal)
	signal.Notify(sigc, syscall.SIGINT)

	go func() {
		<-sigc
		so.Close()
		os.Exit(1)
	}()

	go server.processTasks()

	log.Printf("Queue server listening on: %s\n", socket_path)
	rpc := &rpc.Server{}
	err = rpc.Register(server)
	if err != nil {
		return err
	}
	rpc.Accept(so)
	return nil
}

type RPCQueueArgs struct {
	Task Task
}

func (s *Server) Queue(args RPCQueueArgs, id *int64) error {
	err := s.db.Save(&args.Task)
	if err != nil {
		return err
	}
	s.moreWork.Signal()
	*id = args.Task.Id
	return nil
}

func (s *Server) processTasks() {
	for {
		for {
			var err error
			task, err := s.db.NextTaskIn(TaskWaiting)
			if err != nil {
				panic(err)
			}

			if task == nil {
				log.Println("No tasks waiting... sleep.")
				break
			}

			log.Printf("do task(%d)\n", task.Id)
			task.State = TaskRunning
			err = s.db.Save(task)
			if err != nil {
				panic(err)
			}

			err = task.Run()

			if err != nil {
				task.State = TaskError
			} else {
				task.State = TaskSuccess
			}

			err = s.db.Save(task)
			if err != nil {
				panic(err)
			}
		}

		s.moreWork.L.Lock()
		s.moreWork.Wait()
		s.moreWork.L.Unlock()
	}
}
