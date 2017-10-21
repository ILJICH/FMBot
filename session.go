package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
)

type Session struct {
	Id         int64
	exit       chan bool
	connection net.Conn
	from_muck  chan<- Message
	to_muck    <-chan Message
}

func (s *Session) SetUp(UserID int64, from_muck chan<- Message, to_muck <-chan Message) {
	s.Id = UserID
	s.from_muck = from_muck
	s.to_muck = to_muck

	s.exit = make(chan bool)

	conn, err := net.Dial("tcp", "furrymuck.com:8888")
	if err != nil {
		log.Panic(err)
	}
	log.Printf("Connection with muck established for user %s", UserID)
	s.connection = conn
}

func (s *Session) Stop() {
	log.Printf("Closing muck session of user %s", s.Id)
	close(s.exit)
}

func (s Session) Work() {
	log.Printf("Processing the connection of user %s", s.Id)
	exitWait := new(sync.WaitGroup)
	exitWait.Add(2)
	go func() {
		defer func() {
			exitWait.Done()
			log.Printf("Reading from muck for user %s was stopped.", s.Id)
		}()
		s.StartReading()
	}()
	go func() {
		defer func() {
			exitWait.Done()
			log.Printf("Writing to muck for user %s was stopped.", s.Id)
		}()
		s.StartWriting()
	}()
	exitWait.Wait()
	log.Printf("Processing for user %s was stopped.", s.Id)
}

func (s *Session) StartReading() {
	reader := bufio.NewReader(s.connection)
	for {
		text, err := reader.ReadString('\n')
		if len(text) > 0 {
			log.Printf("Got text from muck for user %s: %s", s.Id, text)
			s.from_muck <- Message{s.Id, text}
		}

		if err == io.EOF {
			log.Printf("Connection of user %s was closed by server.", s.Id)
			s.connection.Close()
			close(s.exit)
			return
		} else if err != nil {
			log.Printf("Error wile reading from muck for user %s: %s. Closing connection.", s.Id, err)
			s.connection.Close()
			close(s.exit)
			return
		}

		// XXX: Вообще, это не самое лучшее решение, так как мы ждём данных
		// из соединения перед тем как его закрыть.
		// Я не знаю как тут неблокирующе читать из сокета, поэтому пока вот так.
		select {
		case <-s.exit:
			{
				log.Printf("Closing connection for user %s", s.Id)
				s.connection.Close()
				return
			}
		default:
		}
	}
}

func (s *Session) StartWriting() {
	for {
		select {
		case message := <-s.to_muck:
			{
				log.Printf("Sending info of user %s to muck: %s", message.UserId, message.Text)
				fmt.Fprintf(s.connection, message.Text+"\n")
			}
		case <-s.exit:
			{
				log.Printf("Stopping reading for user %s", s.Id)
				return
			}
		default:
		}
	}
}
