package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-playground/log"
	"github.com/go-playground/log/handlers/console"
)

func handleRoot(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "Hello Universe!")
}

func mainWithStatus() int {
	cLog := console.New()
	log.RegisterHandler(cLog, log.AllLevels...)
	defer log.Trace("Https server finished").End()

	signalChannel := make(chan os.Signal, 10)
	signal.Notify(signalChannel, syscall.SIGINT, syscall.SIGABRT, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	listenAddrFlag := flag.String("listen", "0.0.0.0:5000", "Listen address for http server")
	flag.Parse()
	listenAddr := *listenAddrFlag

	mux := http.NewServeMux()
	mux.HandleFunc("/", handleRoot)

	server := &http.Server{
		Addr:    listenAddr,
		Handler: mux,
	}

	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Error("Failed to listen at '%s': %s", listenAddr, err)
		return 1
	}

	// if you don't specify addr (e.g. port) we need to find to which it was bound so e.g. tests can use it
	listenAddr = listener.Addr().String()
	log.Infof("Http server listening on http://%s/", listenAddr)

	// TODO: rewrite this to use Shutdown method once we have Go 1.8
	go func() {
		<-ctx.Done()
		log.Infof("Stopping http server listening on http://%s/", listenAddr)
		listener.Close()
	}()

	var errorChannel chan error
	go func() {
		errorChannel <- server.Serve(listener)
	}()

	for {
		select {
		case err := <-errorChannel:
			log.Error(err)
			return 1
		case s := <-signalChannel:
			log.Infof("Cancelling due to signal '%s'", s)
			cancel()
			return 0
		}
	}
}

func main() {
	os.Exit(mainWithStatus())
}
