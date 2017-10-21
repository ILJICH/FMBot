package main

import (
	"log"
	"sync"
)

type SessionWrapper struct {
	session   *Session
	to_muck   chan<- Message
	from_muck <-chan Message
}

type Processor struct {
	sessions      map[int64]*SessionWrapper
	sessionsLock  sync.Mutex
	to_telegram   chan<- Message
	from_telegram <-chan Message
}

func (p *Processor) SetUp(to_telegram chan<- Message, from_telegram <-chan Message) {
	p.from_telegram = from_telegram
	p.to_telegram = to_telegram
	p.sessions = map[int64]*SessionWrapper{}

	log.Printf("Processor initialized")
}

func (p *Processor) Work(exit chan bool) {
	for {
		select {
		case message := <-p.from_telegram:
			{
				p.sessionsLock.Lock()
				switch message.Text {
				case connect_cmd:
					{
						if session := p.SpawnSession(message.UserId); session != nil {
							p.to_telegram <- Message{
								UserId: message.UserId,
								Text:   "SYSTEM: Connection established",
							}
						} else {
							p.to_telegram <- Message{
								UserId: message.UserId,
								Text:   "SYSTEM: Connection is active",
							}
						}
					}
				case disconnect_cmd:
					{
						if session := p.GetSession(message.UserId); session != nil {
							p.to_telegram <- Message{
								UserId: message.UserId,
								Text:   "SYSTEM: Not implemented yet, try: QUIT",
							}
						} else {
							// TODO: close connection
							p.to_telegram <- Message{
								UserId: message.UserId,
								Text:   "SYSTEM: Connection is not active",
							}
						}
					}
				default:
					{
						if session := p.GetSession(message.UserId); session != nil {
							session.to_muck <- message
						} else {
							p.to_telegram <- Message{
								UserId: message.UserId,
								Text:   "SYSTEM: Connection is not active. Try: " + connect_cmd,
							}
						}
					}
				}
				p.sessionsLock.Unlock()
			}
		case <-exit:
			{
				log.Printf("Stopping Processor...")
				return
			}
		}
	}
}

func (p *Processor) GetSession(UserId int64) *SessionWrapper {
	if session, ok := p.sessions[UserId]; ok {
		return session
	} else {
		return nil
	}
}

func (p *Processor) SpawnSession(UserId int64) *SessionWrapper {
	log.Printf("Spawning session for user %s", UserId)
	if session := p.GetSession(UserId); session != nil {
		return nil
	}

	session := new(Session)
	to_muck := make(chan Message)
	session.SetUp(UserId, p.to_telegram, to_muck)
	wrapper := SessionWrapper{
		session: session,
		to_muck: to_muck,
	}
	p.sessions[UserId] = &wrapper
	go func() {
		defer func() {
			p.CleanUpSession(UserId)
		}()
		session.Work()
	}()
	return &wrapper
}

func (p *Processor) CleanUpSession(UserId int64) {
	log.Printf("Cleaning up session of user %s", UserId)
	p.sessionsLock.Lock()
	delete(p.sessions, UserId)
	p.sessionsLock.Unlock()
	p.to_telegram <- Message{
		UserId: UserId,
		Text:   "SYSTEM: Connection closed",
	}
}
