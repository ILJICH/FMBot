package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

const connect_cmd = "/connect"
const disconnect_cmd = "/disconnect"

func main() {

	token := flag.String("token", "", "Telegram API Token")
	flag.Parse()

	if len(*token) == 0 {
		log.Panic("Telegram token is required.")
	}

	from_telegram := make(chan Message)
	to_telegram := make(chan Message)

	telegrammer := new(Telegrammer)
	telegrammer.SetUp(*token, to_telegram, from_telegram)

	processor := new(Processor)
	processor.SetUp(to_telegram, from_telegram)

	exitWait := new(sync.WaitGroup)

	exit := make(chan bool)

	exitWait.Add(1)
	go func() {
		defer exitWait.Done()
		telegrammer.Work(exit)
	}()

	exitWait.Add(1)
	go func() {
		defer exitWait.Done()
		processor.Work(exit)
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		log.Printf("Got %s, shutting down gracefully...", sig)
		close(exit)
	}()

	log.Printf("Running.")
	exitWait.Wait()
	log.Printf("Stopped.")
}
