package goq

import (
	"log"
	"net"
	"net/rpc"
	"os"
	"os/signal"
	"syscall"
)

type Server struct {
	db *DB
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
	server := &Server{db: db}

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
	*id = args.Task.Id
	return nil
}

// func (s *Server) Start() error {
// 	so, err := net.Listen("unix", socket_path)

// 	sigc = make(chan os.Signal)
// 	signal.Notify(sigc, syscall.SIGINT)

// 	go func() {
// 		<-sigc
// 		so.Close()
// 		os.Exit(1)
// 	}()

// 	// go runCommands
// 	// go serverRequests

// 	// var err error
// 	// db, err := sql.Open("sqlite3", "./goq.db")
// 	// if err != nil {
// 	//  log.Fatal(err)
// 	// }
// 	// defer db.Close()

// 	// rows, err := db.Query("select id, task, state from tasks order by id asc;")
// 	// if err != nil {
// 	//  log.Fatal(err)
// 	// }
// 	// for rows.Next() {
// 	//  var id int
// 	//  var state string
// 	//  var taskJSON string
// 	//  rows.Scan(&id, &taskJSON, &state)
// 	//  r := strings.NewReader(taskJSON)
// 	//  decoder := json.NewDecoder(r)
// 	//  var task Task
// 	//  err = decoder.Decode(&task)
// 	//  if err != nil {
// 	//    log.Println(err)
// 	//    continue
// 	//  }
// 	//  rows.Close()

// 	//  err = task.Run()
// 	//  if err != nil {
// 	//    log.Println(err)
// 	//  } else {
// 	//    _, err := db.Exec("update tasks set state=? where id=?", TaskSuccess.String(), id)
// 	//    if err != nil {
// 	//      log.Fatal(err)
// 	//    }
// 	//  }
// 	// }
// }

// func (s *Server) serveRequests(so net.Listener) {
// 	for {
// 		c, err := so.Accept()
// 		if err != nil {
// 			log.Println(err)
// 			continue
// 		}
// 	}
// }

// func (s *Server) serveRequest(c net.Conn) {

// }
