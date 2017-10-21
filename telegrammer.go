package main

import (
	"html"
	"log"
	"sync"
	"time"

	"gopkg.in/telegram-bot-api.v4"
)

type LastMessage struct {
	lock    sync.Mutex
	text    string
	running bool
}

type Telegrammer struct {
	Bot           *tgbotapi.BotAPI
	last_message  map[int64]*LastMessage
	to_telegram   <-chan Message
	from_telegram chan<- Message
}

func (t *Telegrammer) SetUp(token string, to_telegram <-chan Message, from_telegram chan<- Message) {
	t.from_telegram = from_telegram
	t.to_telegram = to_telegram

	t.last_message = map[int64]*LastMessage{}

	t.Bot, _ = tgbotapi.NewBotAPI(token)
	t.Bot.Debug = true

	log.Printf("Authorized on account %s", t.Bot.Self.UserName)
}

// Отправка сообщения.
// Не выполняет отправку сразу, копит приходящие пользователю сообщения в единый буфер.
// Реальная отправка происходит через 0.5 секунд после прихода первого сообщения.
func (t *Telegrammer) Send(message Message) {
	if last_message, ok := t.last_message[message.UserId]; ok {
		last_message.lock.Lock()
		last_message.text += message.Text
		if !last_message.running {
			last_message.running = true
			// TODO: вынести время подавления дребезга в конфиг
			go t.DelayedSend(message.UserId, 500*time.Millisecond)
		}
		last_message.lock.Unlock()
	} else {
		t.last_message[message.UserId] = &LastMessage{
			lock:    sync.Mutex{},
			running: true,
			text:    message.Text,
		}
		go t.DelayedSend(message.UserId, 500*time.Millisecond)
	}
}

func (t *Telegrammer) DelayedSend(UserID int64, delay time.Duration) {
	// Рассматриваем случай только когда запись была создана
	if message, ok := t.last_message[UserID]; ok {
		time.Sleep(delay)
		message.lock.Lock()
		msg := tgbotapi.NewMessage(UserID, "<pre>"+html.EscapeString(message.text)+"</pre>")
		msg.ParseMode = "HTML"
		// TODO: Обработка ошибки отправки
		t.Bot.Send(msg)
		message.text = ""
		message.running = false
		message.lock.Unlock()
	}
}

func (t *Telegrammer) Work(exit chan bool) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates, _ := t.Bot.GetUpdatesChan(u)
	for {
		select {
		case update := <-updates:
			{
				if update.Message == nil {
					continue
				}
				log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)
				t.from_telegram <- Message{update.Message.Chat.ID, update.Message.Text}
			}
		case message := <-t.to_telegram:
			{
				t.Send(message)
			}
		case <-exit:
			{
				log.Printf("Stopping Telegrammer...")
				return
			}
		}
	}
}
