package telegram

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)


// Bot handles Telegram long-polling and notifications.
type Bot struct {
	token  string
	db     *sql.DB
	apiURL string
}

// New creates a Bot. Returns nil if token is empty.
func New(token string, db *sql.DB) *Bot {
	if token == "" {
		return nil
	}
	return &Bot{
		token:  token,
		db:     db,
		apiURL: "https://api.telegram.org/bot" + token,
	}
}

// ---------- Telegram API types ----------

type tgUpdate struct {
	UpdateID int64      `json:"update_id"`
	Message  *tgMessage `json:"message"`
}

type tgMessage struct {
	Chat tgChat `json:"chat"`
	Text string `json:"text"`
}

type tgChat struct {
	ID int64 `json:"id"`
}

type tgResponse struct {
	OK     bool            `json:"ok"`
	Result json.RawMessage `json:"result"`
}

// ---------- Long-polling loop ----------

// StartPolling runs the update loop in a goroutine.
func (b *Bot) StartPolling() {
	go b.poll()
}

func (b *Bot) poll() {
	offset := int64(0)
	client := &http.Client{Timeout: 35 * time.Second}

	for {
		updates, err := b.getUpdates(client, offset)
		if err != nil {
			log.Printf("[tg-bot] getUpdates error: %v", err)
			time.Sleep(3 * time.Second)
			continue
		}

		for _, u := range updates {
			b.handleUpdate(u)
			offset = u.UpdateID + 1
		}
	}
}

func (b *Bot) getUpdates(client *http.Client, offset int64) ([]tgUpdate, error) {
	resp, err := client.Get(fmt.Sprintf("%s/getUpdates?offset=%d&timeout=30", b.apiURL, offset))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var r tgResponse
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, err
	}

	var updates []tgUpdate
	if err := json.Unmarshal(r.Result, &updates); err != nil {
		return nil, err
	}
	return updates, nil
}

// ---------- Handlers ----------

func (b *Bot) handleUpdate(u tgUpdate) {
	if u.Message == nil {
		return
	}

	text := strings.TrimSpace(u.Message.Text)
	if text == "/start" {
		b.handleStart(u.Message.Chat.ID)
	}
}

func (b *Bot) handleStart(chatID int64) {
	_, err := b.db.Exec(
		`INSERT INTO tg_chats(chat_id) VALUES ($1) ON CONFLICT DO NOTHING`,
		chatID,
	)
	if err != nil {
		log.Printf("[tg-bot] insert tg_chats: %v", err)
		b.sendMessage(chatID, "Ошибка. Попробуйте позже.")
		return
	}

	b.sendMessage(chatID, "✅ Вы подписаны на уведомления о заказах.")
}

// ---------- Send helpers ----------

func (b *Bot) sendMessage(chatID int64, text string) {
	vals := url.Values{}
	vals.Set("chat_id", strconv.FormatInt(chatID, 10))
	vals.Set("text", text)
	vals.Set("parse_mode", "HTML")

	resp, err := http.PostForm(b.apiURL+"/sendMessage", vals)
	if err != nil {
		log.Printf("[tg-bot] sendMessage error: %v", err)
		return
	}
	resp.Body.Close()
}

