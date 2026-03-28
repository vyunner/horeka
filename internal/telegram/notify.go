package telegram

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"
)

// OrderInfo contains everything needed to format a notification.
type OrderInfo struct {
	OrderID    int64
	LocationID int64
	UserPhone  string
	Comment    string
	Items      []string // raw_product lines
	CreatedAt  time.Time
}

type orderAction int

const (
	actionNew       orderAction = iota
	actionUpdated
	actionCancelled
)

// NotifyNewOrder sends a "new order" notification to all registered chats.
func (b *Bot) NotifyNewOrder(info OrderInfo) {
	b.notifyOrder(info, actionNew)
}

// NotifyUpdatedOrder sends an "order updated" notification to all registered chats.
func (b *Bot) NotifyUpdatedOrder(info OrderInfo) {
	b.notifyOrder(info, actionUpdated)
}

// NotifyCancelledOrder sends an "order cancelled" notification to all registered chats.
func (b *Bot) NotifyCancelledOrder(info OrderInfo) {
	b.notifyOrder(info, actionCancelled)
}

func (b *Bot) notifyOrder(info OrderInfo, action orderAction) {
	if b == nil {
		return
	}

	// Get location name
	var locName string
	err := b.db.QueryRow(`SELECT name FROM locations WHERE id=$1`, info.LocationID).Scan(&locName)
	if err != nil {
		locName = fmt.Sprintf("ID %d", info.LocationID)
	}

	// Build message
	var sb strings.Builder
	switch action {
	case actionUpdated:
		sb.WriteString(fmt.Sprintf("✏️ <b>Заказ #%d изменён</b>\n", info.OrderID))
	case actionCancelled:
		sb.WriteString(fmt.Sprintf("❌ <b>Заказ #%d отменён</b>\n", info.OrderID))
	default:
		sb.WriteString(fmt.Sprintf("🆕 <b>Новый заказ #%d</b>\n", info.OrderID))
	}
	sb.WriteString(fmt.Sprintf("📍 %s\n", escapeHTML(locName)))
	if info.UserPhone != "" {
		sb.WriteString(fmt.Sprintf("👤 +%s\n", escapeHTML(info.UserPhone)))
	}
	if info.Comment != "" {
		sb.WriteString(fmt.Sprintf("💬 %s\n", escapeHTML(info.Comment)))
	}

	sb.WriteString("\n<b>Позиции:</b>\n")
	for _, item := range info.Items {
		sb.WriteString(fmt.Sprintf("• %s\n", escapeHTML(item)))
	}

	loc, _ := time.LoadLocation("Asia/Almaty")
	sb.WriteString(fmt.Sprintf("\n🕐 %s", info.CreatedAt.In(loc).Format("02.01.2006 15:04")))

	text := sb.String()

	chatIDs := b.getSubscribers()
	for _, chatID := range chatIDs {
		b.sendMessage(chatID, text)
	}
}

func (b *Bot) getSubscribers() []int64 {
	rows, err := b.db.Query(`SELECT DISTINCT chat_id FROM tg_chats`)
	if err != nil {
		log.Printf("[tg-bot] getSubscribers: %v", err)
		return nil
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	return ids
}

// NotifyNewOrderFromDB builds OrderInfo from DB and sends notifications.
func (b *Bot) NotifyNewOrderFromDB(db *sql.DB, orderID int64) {
	b.notifyOrderFromDB(db, orderID, actionNew)
}

// NotifyUpdatedOrderFromDB builds OrderInfo from DB and sends update notification.
func (b *Bot) NotifyUpdatedOrderFromDB(db *sql.DB, orderID int64) {
	b.notifyOrderFromDB(db, orderID, actionUpdated)
}

// NotifyCancelledOrderFromDB builds OrderInfo from DB and sends cancellation notification.
func (b *Bot) NotifyCancelledOrderFromDB(db *sql.DB, orderID int64) {
	b.notifyOrderFromDB(db, orderID, actionCancelled)
}

func (b *Bot) notifyOrderFromDB(db *sql.DB, orderID int64, action orderAction) {
	if b == nil {
		return
	}

	var info OrderInfo
	info.OrderID = orderID

	var commentNull sql.NullString
	err := db.QueryRow(
		`SELECT o.location_id, u.phone, o.comment, o.created_at
		 FROM orders o
		 JOIN users u ON u.id = o.user_id
		 WHERE o.id = $1`, orderID,
	).Scan(&info.LocationID, &info.UserPhone, &commentNull, &info.CreatedAt)

	if err != nil {
		log.Printf("[tg-bot] notifyOrderFromDB query order: %v", err)
		return
	}

	if commentNull.Valid {
		info.Comment = commentNull.String
	}

	rows, err := db.Query(
		`SELECT raw_product FROM order_requests WHERE order_id=$1 ORDER BY id`, orderID,
	)
	if err != nil {
		log.Printf("[tg-bot] notifyOrderFromDB query requests: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var item string
		if err := rows.Scan(&item); err == nil {
			info.Items = append(info.Items, item)
		}
	}

	b.notifyOrder(info, action)
}

func escapeHTML(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
	return r.Replace(s)
}
