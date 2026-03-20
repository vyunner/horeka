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
	UpdateID      int64            `json:"update_id"`
	Message       *tgMessage       `json:"message"`
	CallbackQuery *tgCallbackQuery `json:"callback_query"`
}

type tgMessage struct {
	Chat tgChat `json:"chat"`
	Text string `json:"text"`
}

type tgChat struct {
	ID int64 `json:"id"`
}

type tgCallbackQuery struct {
	ID      string    `json:"id"`
	From    tgUser    `json:"from"`
	Message *tgMessage `json:"message"`
	Data    string    `json:"data"`
}

type tgUser struct {
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
	if u.CallbackQuery != nil {
		b.handleCallback(u.CallbackQuery)
		return
	}

	if u.Message == nil {
		return
	}

	text := strings.TrimSpace(u.Message.Text)
	if text == "/start" {
		b.handleStart(u.Message.Chat.ID)
	}
}

func (b *Bot) handleStart(chatID int64) {
	// Fetch locations from DB
	rows, err := b.db.Query(`SELECT id, name FROM locations ORDER BY name`)
	if err != nil {
		log.Printf("[tg-bot] locations query: %v", err)
		b.sendMessage(chatID, "Ошибка загрузки локаций. Попробуйте позже.")
		return
	}
	defer rows.Close()

	type loc struct {
		ID   int64
		Name string
	}

	var locs []loc
	for rows.Next() {
		var l loc
		if err := rows.Scan(&l.ID, &l.Name); err != nil {
			continue
		}
		locs = append(locs, l)
	}

	if len(locs) == 0 {
		b.sendMessage(chatID, "Нет доступных локаций.")
		return
	}

	// Build inline keyboard
	var keyboard [][]map[string]string
	for _, l := range locs {
		keyboard = append(keyboard, []map[string]string{
			{"text": l.Name, "callback_data": fmt.Sprintf("sub:%d", l.ID)},
		})
	}

	markup, _ := json.Marshal(map[string]interface{}{
		"inline_keyboard": keyboard,
	})

	b.sendMessageWithMarkup(chatID, "Выберите заведение для получения уведомлений о заказах:", string(markup))
}

func (b *Bot) handleCallback(cb *tgCallbackQuery) {
	// Answer callback to remove loading spinner
	b.answerCallback(cb.ID)

	if !strings.HasPrefix(cb.Data, "sub:") {
		return
	}

	locIDStr := strings.TrimPrefix(cb.Data, "sub:")
	locID, err := strconv.ParseInt(locIDStr, 10, 64)
	if err != nil {
		return
	}

	chatID := cb.Message.Chat.ID

	// Get location name
	var locName string
	err = b.db.QueryRow(`SELECT name FROM locations WHERE id=$1`, locID).Scan(&locName)
	if err != nil {
		b.sendMessage(chatID, "Локация не найдена.")
		return
	}

	// Upsert subscription
	_, err = b.db.Exec(
		`INSERT INTO tg_chats(chat_id, location_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		chatID, locID,
	)
	if err != nil {
		log.Printf("[tg-bot] insert tg_chats: %v", err)
		b.sendMessage(chatID, "Ошибка. Попробуйте позже.")
		return
	}

	b.sendMessage(chatID, fmt.Sprintf("✅ Вы подписаны на уведомления: %s\n\nОтправьте /start чтобы подписаться на другие заведения.", locName))
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

func (b *Bot) sendMessageWithMarkup(chatID int64, text, replyMarkup string) {
	vals := url.Values{}
	vals.Set("chat_id", strconv.FormatInt(chatID, 10))
	vals.Set("text", text)
	vals.Set("parse_mode", "HTML")
	vals.Set("reply_markup", replyMarkup)

	resp, err := http.PostForm(b.apiURL+"/sendMessage", vals)
	if err != nil {
		log.Printf("[tg-bot] sendMessage error: %v", err)
		return
	}
	resp.Body.Close()
}

func (b *Bot) answerCallback(callbackID string) {
	vals := url.Values{}
	vals.Set("callback_query_id", callbackID)

	resp, err := http.PostForm(b.apiURL+"/answerCallbackQuery", vals)
	if err != nil {
		return
	}
	resp.Body.Close()
}
