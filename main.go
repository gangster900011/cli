package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"time"

	"code.rocketnine.space/tslocum/cview"
)

var (
	serverAddress string
	gameLogged    bool
	statusLogged  bool
	debug         int
)

func l(s string) {
	m := time.Now().Format("15:04") + string(cview.BoxDrawingsLightVertical) + " " + s
	if statusWriter != nil {
		if statusLogged {
			statusWriter.Write([]byte("\n" + m))
			return
		}
		statusWriter.Write([]byte(m))
		statusLogged = true
		return
	}
	log.Print(m)
}

func lf(format string, a ...interface{}) {
	l(fmt.Sprintf(format, a...))
}

func lg(s string) {
	m := time.Now().Format("15:04") + string(cview.BoxDrawingsLightVertical) + " " + s
	if gameWriter != nil {
		if gameLogged {
			gameWriter.Write([]byte("\n" + m))
			return
		}
		gameWriter.Write([]byte(m))
		gameLogged = true
		return
	}
	log.Print(m)
}

func handleAutoRefresh() {
	t := time.NewTicker(10 * time.Second) // TODO configurable
	for range t.C {
		if !autoRefresh || gameInProgress {
			continue
		}

		board.client.Out <- []byte("ls")
	}
}

func main() {
	var (
		username string
		password string
	)
	flag.StringVar(&username, "username", "", "username")
	flag.StringVar(&password, "password", "", "password")
	flag.StringVar(&serverAddress, "server", "bgammon.org:1337", "server address")
	flag.IntVar(&debug, "debug", 0, "print debug information and serve pprof on specified port")
	flag.Parse()

	if debug > 0 {
		go func() {
			log.Fatal(http.ListenAndServe(fmt.Sprintf("localhost:%d", debug), nil))
		}()
	}

	app = cview.NewApplication()

	c := NewClient(serverAddress, username, password)

	board = NewGameBoard(c)

	go handleAutoRefresh()

	err := RunApp(c, board)
	if err != nil {
		log.Fatalf("%+v", err)
	}
}
