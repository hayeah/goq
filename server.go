package goq

import (
	"fmt"
	"log"
	"net"
	"net/rpc"
	"os"
	"os/signal"
	"syscall"
)

type Server struct {
	db          *DB
	moreWork    chan Empty
	workersChan chan Empty
	killChans   map[int64]chan syscall.Signal
}

type Empty struct{}

const (
	db_path        = "./goq.db"
	socket_path    = "./goq.socket"
	server_workers = 4
)

// server loop
func StartServer() error {
	db, err := OpenDB(db_path)
	if err != nil {
		return err
	}

	server := &Server{
		db:          db,
		moreWork:    make(chan Empty),
		workersChan: make(chan Empty, server_workers),
		killChans:   make(map[int64]chan syscall.Signal),
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

	log.Printf("server pid: %d\n", os.Getpid())
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

type RPCListArgs struct {
	State TaskState
}

type RPCStopArgs struct {
	TaskId int64
	Signal syscall.Signal
}

func (s *Server) Stop(args RPCStopArgs, taskId *int64) error {
	killch, ok := s.killChans[args.TaskId]
	if ok {
		killch <- args.Signal
		*taskId = args.TaskId
		return nil
	} else {
		return fmt.Errorf("Task is not running: %d", args.TaskId)
	}
}

func (s *Server) List(args RPCListArgs, rtasks *[]Task) error {
	tasks, err := s.db.List(args.State)
	if err != nil {
		return err
	}

	*rtasks = tasks

	return nil
}

func (s *Server) Queue(args RPCQueueArgs, id *int64) error {
	err := s.db.Save(&args.Task)
	if err != nil {
		return err
	}

	var empty Empty
	select {
	case s.moreWork <- empty:
	default:
	}

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

			var empty Empty
			// limit number of workers
			s.workersChan <- empty
			go func() {
				err := s.processTask(task)
				if err != nil {
					panic(err)
				}
				<-s.workersChan
			}()
		}

		<-s.moreWork
	}
}

func (s *Server) processTask(task *Task) error {
	var err error

	cmd := task.Command()
	err = cmd.Start()

	if err != nil {
		log.Printf("task(%d): %s\n", task.Id, err)
		task.State = TaskError
	} else {
		killChan := make(chan syscall.Signal)
		exitChan := make(chan error)
		go func() {
			err := cmd.Wait()
			exitChan <- err
		}()

		s.killChans[task.Id] = killChan

		killed := false

	L:
		for {
			var err error
			select {
			case err = <-exitChan:
				// normal or error exit
				if killed {
					task.State = TaskStopped
				} else if err != nil {
					task.State = TaskError
				} else {
					task.State = TaskSuccess
				}
				break L
			case sig := <-killChan:
				// send signals to task's process
				killed = true
				cmd.Process.Signal(sig)
			}
		}
	}

	delete(s.killChans, task.Id)
	log.Printf("exit task(%d): %s\n", task.Id, task.State.String())
	err = s.db.Save(task)
	if err != nil {
		return err
	}
	return nil
}
