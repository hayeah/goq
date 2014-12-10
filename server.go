package goq

import (
	"fmt"
	"log"
	"net"
	"net/rpc"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

type Server struct {
	db                         *DB
	moreWork                   chan Empty
	workersChan                chan Empty
	killChans                  map[int64]chan syscall.Signal
	numberOfWorkers            int
	adjustNumberOfWorkersEvent chan Empty
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
		db:                         db,
		moreWork:                   make(chan Empty, 1),
		numberOfWorkers:            server_workers,
		workersChan:                make(chan Empty, server_workers),
		killChans:                  make(map[int64]chan syscall.Signal),
		adjustNumberOfWorkersEvent: make(chan Empty, 1),
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

type RPCWorkersArgs struct {
	NumberOfWorkers      int
	CheckNumberOfWorkers bool
}

func (s *Server) Workers(args RPCWorkersArgs, n *int) error {
	if args.CheckNumberOfWorkers {
		*n = s.numberOfWorkers
		return nil
	}

	s.numberOfWorkers = args.NumberOfWorkers

	var empty Empty
	select {
	case s.adjustNumberOfWorkersEvent <- empty:
		log.Printf("Will adjust workers number to: %d\n", s.numberOfWorkers)
	default:
		log.Println("workers already sent event")
	}

	*n = args.NumberOfWorkers
	return nil
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

// Copy existing messages to new channel.
// Discard messages that exceeds the capacity of the new channel.
func (s *Server) adjustWorkersChan() {
	if cap(s.workersChan) != s.numberOfWorkers {
		log.Printf("Adjusting workers: %d -> %d\n", cap(s.workersChan), s.numberOfWorkers)
		log.Printf("Number of running tasks: %d\n", len(s.workersChan))
		oldChan := s.workersChan
		newChan := make(chan Empty, s.numberOfWorkers)
		close(oldChan)
	L:
		for empty := range s.workersChan {
			select {
			case newChan <- empty:
			default:
				break L
			}
		}

		s.workersChan = newChan
	}
}

func (s *Server) processTasks() {
	for {
		var workersLock sync.RWMutex
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

			var empty Empty

		L:
			for {
				select {
				case <-s.adjustNumberOfWorkersEvent:
					// After adjust, try the workersChan again.
					workersLock.Lock()
					s.adjustWorkersChan()
					workersLock.Unlock()
				// limit number of workers
				case s.workersChan <- empty:
					// Don't need to lock this. Is already mutex with adjustWorkersChan because it's not a separate goroutine.
					break L
				}
			}

			log.Printf("do task(%d)\n", task.Id)
			task.State = TaskRunning
			err = s.db.Save(task)
			if err != nil {
				panic(err)
			}

			go func() {
				err := s.processTask(task)
				if err != nil {
					panic(err)
				}
				workersLock.RLock()
				select {
				case <-s.workersChan:
				default:
					// If workers capacity decreased, there could be more running processes than message tokens in workersChan.
				}
				workersLock.RUnlock()
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
