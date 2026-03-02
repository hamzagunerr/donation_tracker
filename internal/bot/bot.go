package bot

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"

	"github.com/hayratyardim/donation_tracker/internal/config"
	"github.com/hayratyardim/donation_tracker/internal/database"
	"github.com/hayratyardim/donation_tracker/internal/models"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Bot struct {
	api             *tgbotapi.BotAPI
	db              *database.DB
	cfg             *config.Config
	pendingMessages map[string]*models.PendingMessage // Benzersiz ID -> Mesaj
	lastMessages    map[int64]*models.PendingMessage  // Her grup için son mesaj
	mu              sync.Mutex
}

func New(cfg *config.Config, db *database.DB) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(cfg.TelegramToken)
	if err != nil {
		return nil, err
	}

	log.Printf("Bot başlatıldı: @%s", api.Self.UserName)

	return &Bot{
		api:             api,
		db:              db,
		cfg:             cfg,
		pendingMessages: make(map[string]*models.PendingMessage),
		lastMessages:    make(map[int64]*models.PendingMessage),
	}, nil
}

func (b *Bot) generatePendingID(channelID int64, messageID int) string {
	return fmt.Sprintf("%d_%d", channelID, messageID)
}

func (b *Bot) Start(ctx context.Context) error {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case update := <-updates:
			b.handleUpdate(ctx, update)
		}
	}
}

func (b *Bot) handleUpdate(ctx context.Context, update tgbotapi.Update) {
	// Callback query (inline buton tıklaması)
	if update.CallbackQuery != nil {
		b.handleCallbackQuery(ctx, update.CallbackQuery)
		return
	}

	// Kanal mesajları
	if update.ChannelPost != nil {
		b.handleGroupOrChannelMessage(ctx, update.ChannelPost)
		return
	}

	// Bot gruba/kanala eklendiğinde
	if update.MyChatMember != nil {
		b.handleMyChatMember(ctx, update.MyChatMember)
		return
	}

	if update.Message != nil {
		// Grup veya supergroup mesajları
		if update.Message.Chat.IsGroup() || update.Message.Chat.IsSuperGroup() {
			b.handleGroupOrChannelMessage(ctx, update.Message)
			return
		}

		// Admin veya takvim admin'inin özel sohbetinden gelen komutlar
		chatID := update.Message.Chat.ID
		isAdmin := chatID == b.cfg.AdminChatID
		isCalendarAdmin := b.cfg.CalendarAdminID != 0 && chatID == b.cfg.CalendarAdminID

		if isAdmin || isCalendarAdmin {
			if update.Message.IsCommand() {
				b.handleCommand(ctx, update.Message)
			}
		}
		return
	}
}

func (b *Bot) handleCallbackQuery(ctx context.Context, callback *tgbotapi.CallbackQuery) {
	fromID := callback.From.ID
	isAdmin := fromID == b.cfg.AdminChatID
	isCalendarAdmin := b.cfg.CalendarAdminID != 0 && fromID == b.cfg.CalendarAdminID

	if !isAdmin && !isCalendarAdmin {
		return
	}

	data := callback.Data
	parts := strings.SplitN(data, ":", 2)
	if len(parts) != 2 {
		return
	}

	action := parts[0]
	param := parts[1]

	var responseText string

	switch action {
	case "ekle":
		if !isAdmin {
			return
		}
		b.mu.Lock()
		pending, exists := b.pendingMessages[param]
		b.mu.Unlock()

		if !exists {
			responseText = "⚠️ Bu mesaj artık mevcut değil."
		} else {
			donation := &models.Donation{
				MessageID:    pending.MessageID,
				ChannelID:    pending.ChannelID,
				ChannelTitle: pending.ChannelTitle,
				SenderName:   pending.SenderName,
				SenderUser:   pending.SenderUser,
				Content:      pending.Content,
				MessageLink:  pending.MessageLink,
				MessageDate:  pending.MessageDate,
			}

			err := b.db.AddDonation(ctx, donation)
			if err != nil {
				if strings.Contains(err.Error(), "zaten veritabanında mevcut") {
					responseText = "⚠️ Bu mesaj zaten veritabanında kayıtlı."
				} else {
					responseText = "❌ Hata: " + err.Error()
				}
			} else {
				responseText = "✅ Eklendi!"
				b.mu.Lock()
				delete(b.pendingMessages, param)
				b.mu.Unlock()

				// Takvim admin'ine gönder
				if b.cfg.CalendarAdminID != 0 {
					b.sendToCalendarAdmin(ctx, donation)
				}
			}
		}

	case "atla":
		if !isAdmin {
			return
		}
		b.mu.Lock()
		_, exists := b.pendingMessages[param]
		if exists {
			delete(b.pendingMessages, param)
		}
		b.mu.Unlock()

		if !exists {
			responseText = "⚠️ Bu mesaj zaten işlendi."
		} else {
			responseText = "⏭ Atlandı"
		}

	case "takvimekle":
		if !isCalendarAdmin {
			return
		}
		donationID, err := strconv.ParseInt(param, 10, 64)
		if err != nil {
			responseText = "❌ Geçersiz ID"
		} else {
			err = b.db.AddToCalendar(ctx, donationID)
			if err != nil {
				responseText = "❌ Hata: " + err.Error()
			} else {
				responseText = "📅 Takvime eklendi!"

				// Callback'e yanıt ver
				callbackResponse := tgbotapi.NewCallback(callback.ID, responseText)
				b.api.Request(callbackResponse)

				// Mesajı güncelle - Takvimden Çıkar butonu göster
				editText := callback.Message.Text + "\n\n" + responseText
				keyboard := tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData("🗑 Takvimden Çıkar", fmt.Sprintf("takvimcikar:%d", donationID)),
					),
				)
				edit := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, editText)
				edit.ReplyMarkup = &keyboard
				b.api.Send(edit)
				return
			}
		}

	case "takvimcikar":
		if !isCalendarAdmin {
			return
		}
		donationID, err := strconv.ParseInt(param, 10, 64)
		if err != nil {
			responseText = "❌ Geçersiz ID"
		} else {
			err = b.db.RemoveFromCalendar(ctx, donationID)
			if err != nil {
				responseText = "❌ Hata: " + err.Error()
			} else {
				responseText = "🗑 Takvimden çıkarıldı"

				// Callback'e yanıt ver
				callbackResponse := tgbotapi.NewCallback(callback.ID, responseText)
				b.api.Request(callbackResponse)

				// Mesajı güncelle - Takvime Ekle butonu göster
				editText := callback.Message.Text
				// Önceki durumu temizle
				editText = strings.Replace(editText, "\n\n📅 Takvime eklendi!", "", 1)
				editText = strings.Replace(editText, "\n\n🗑 Takvimden çıkarıldı", "", 1)
				editText += "\n\n" + responseText

				keyboard := tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData("📅 Takvime Ekle", fmt.Sprintf("takvimekle:%d", donationID)),
					),
				)
				edit := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, editText)
				edit.ReplyMarkup = &keyboard
				b.api.Send(edit)
				return
			}
		}

	case "takvimatlat":
		if !isCalendarAdmin {
			return
		}
		responseText = "⏭ Atlandı"
	}

	// Callback'e yanıt ver
	callbackResponse := tgbotapi.NewCallback(callback.ID, responseText)
	b.api.Request(callbackResponse)

	// Mesajı güncelle (butonları kaldır, sonucu göster)
	editText := callback.Message.Text + "\n\n" + responseText
	edit := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, editText)
	b.api.Send(edit)
}

func (b *Bot) sendToCalendarAdmin(ctx context.Context, donation *models.Donation) {
	// Mesajı takvim admin'ine yönlendir
	forward := tgbotapi.NewForward(b.cfg.CalendarAdminID, donation.ChannelID, donation.MessageID)
	b.api.Send(forward)

	text := "📅 *Yeni Bağış Bildirimi*\n\n"
	text += "📍 *Grup/Kanal:* " + escapeMarkdown(donation.ChannelTitle) + "\n"
	text += "👤 *Gönderen:* " + escapeMarkdown(donation.SenderName)
	if donation.SenderUser != "" {
		text += " (" + escapeMarkdown(donation.SenderUser) + ")"
	}
	text += "\n"
	text += "📅 *Tarih:* " + donation.MessageDate.Format("02.01.2006 15:04") + "\n"
	text += "🔗 *Link:* " + donation.MessageLink + "\n\n"
	text += "_Paylaşım takvimine eklensin mi?_"

	// Inline butonlar
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📅 Takvime Ekle", fmt.Sprintf("takvimekle:%d", donation.ID)),
			tgbotapi.NewInlineKeyboardButtonData("⏭ Atla", fmt.Sprintf("takvimatlat:%d", donation.ID)),
		),
	)

	msg := tgbotapi.NewMessage(b.cfg.CalendarAdminID, text)
	msg.ParseMode = "Markdown"
	msg.DisableWebPagePreview = true
	msg.ReplyMarkup = keyboard

	_, err := b.api.Send(msg)
	if err != nil {
		log.Printf("Takvim admin'ine mesaj gönderme hatası: %v", err)
	}
}

func (b *Bot) handleMyChatMember(ctx context.Context, member *tgbotapi.ChatMemberUpdated) {
	// Bot gruba/kanala eklendiyse kaydet
	newStatus := member.NewChatMember.Status
	if newStatus == "member" || newStatus == "administrator" {
		channel := &models.Channel{
			ChannelID: member.Chat.ID,
			Title:     member.Chat.Title,
			Username:  member.Chat.UserName,
		}

		if err := b.db.AddChannel(ctx, channel); err != nil {
			log.Printf("Grup/kanal kayıt hatası: %v", err)
		} else {
			log.Printf("Yeni grup/kanal kaydedildi: %s (%d)", member.Chat.Title, member.Chat.ID)

			// Grup/kanal linki oluştur
			var link string
			if member.Chat.UserName != "" {
				link = "https://t.me/" + member.Chat.UserName
			} else {
				// Private grup/kanal için link
				chatID := member.Chat.ID
				if chatID < 0 {
					chatID = -chatID - 1000000000000
				}
				link = fmt.Sprintf("https://t.me/c/%d/1", chatID)
			}

			text := fmt.Sprintf("✅ Bot yeni bir gruba/kanala eklendi!\n\n📍 [%s](%s)\nID: `%d`",
				escapeMarkdown(member.Chat.Title), link, member.Chat.ID)
			msg := tgbotapi.NewMessage(b.cfg.AdminChatID, text)
			msg.ParseMode = "Markdown"
			b.api.Send(msg)
		}
	}
}

func (b *Bot) handleGroupOrChannelMessage(ctx context.Context, msg *tgbotapi.Message) {
	// Önce grubu/kanalı kaydet (yoksa)
	b.saveGroupOrChannel(ctx, msg.Chat)

	if msg.Text == "" && msg.Caption == "" {
		return
	}

	content := msg.Text
	if content == "" {
		content = msg.Caption
	}

	chatID := msg.Chat.ID

	// ⚡️ içeren mesaj = Bağış tamamlandı sinyali
	if b.containsLightningEmoji(content) {
		b.mu.Lock()
		lastMsg := b.lastMessages[chatID]
		delete(b.lastMessages, chatID) // Temizle
		b.mu.Unlock()

		// Önceki mesaj varsa admin'e yönlendir
		if lastMsg != nil {
			b.forwardToAdminByPending(ctx, lastMsg)
		}
		return
	}

	// Şimşek yoksa, bu mesajı sakla (bir sonraki şimşeği bekle)
	b.mu.Lock()
	b.lastMessages[chatID] = &models.PendingMessage{
		MessageID:    msg.MessageID,
		ChannelID:    msg.Chat.ID,
		ChannelTitle: msg.Chat.Title,
		SenderName:   b.getSenderName(msg),
		SenderUser:   b.getSenderUsername(msg),
		Content:      content,
		MessageLink:  b.generateMessageLink(msg),
		MessageDate:  msg.Time(),
	}
	b.mu.Unlock()
}

func (b *Bot) saveGroupOrChannel(ctx context.Context, chat *tgbotapi.Chat) {
	channel := &models.Channel{
		ChannelID: chat.ID,
		Title:     chat.Title,
		Username:  chat.UserName,
	}
	b.db.AddChannel(ctx, channel)
}

func (b *Bot) containsLightningEmoji(text string) bool {
	return strings.Contains(text, "⚡")
}

func (b *Bot) getSenderName(msg *tgbotapi.Message) string {
	if msg.ForwardFrom != nil {
		name := msg.ForwardFrom.FirstName
		if msg.ForwardFrom.LastName != "" {
			name += " " + msg.ForwardFrom.LastName
		}
		return name
	}

	if msg.From != nil {
		name := msg.From.FirstName
		if msg.From.LastName != "" {
			name += " " + msg.From.LastName
		}
		return name
	}

	if msg.SenderChat != nil {
		return msg.SenderChat.Title
	}

	return "Bilinmiyor"
}

func (b *Bot) getSenderUsername(msg *tgbotapi.Message) string {
	if msg.ForwardFrom != nil && msg.ForwardFrom.UserName != "" {
		return "@" + msg.ForwardFrom.UserName
	}

	if msg.From != nil && msg.From.UserName != "" {
		return "@" + msg.From.UserName
	}

	if msg.SenderChat != nil && msg.SenderChat.UserName != "" {
		return "@" + msg.SenderChat.UserName
	}

	return ""
}

func (b *Bot) generateMessageLink(msg *tgbotapi.Message) string {
	if msg.Chat.UserName != "" {
		return "https://t.me/" + msg.Chat.UserName + "/" + itoa(msg.MessageID)
	}
	channelID := msg.Chat.ID
	if channelID < 0 {
		channelID = -channelID - 1000000000000
	}
	return "https://t.me/c/" + itoa64(channelID) + "/" + itoa(msg.MessageID)
}

func (b *Bot) forwardToAdminByPending(ctx context.Context, pending *models.PendingMessage) {
	isDup, err := b.db.IsDuplicate(ctx, pending.MessageID, pending.ChannelID)
	if err != nil {
		log.Printf("Duplicate kontrolü hatası: %v", err)
	}

	if isDup {
		return
	}

	// Benzersiz ID oluştur ve pending'e ekle
	pendingID := b.generatePendingID(pending.ChannelID, pending.MessageID)
	b.mu.Lock()
	b.pendingMessages[pendingID] = pending
	b.mu.Unlock()

	// Mesajı yönlendir
	forward := tgbotapi.NewForward(b.cfg.AdminChatID, pending.ChannelID, pending.MessageID)
	_, err = b.api.Send(forward)
	if err != nil {
		log.Printf("Mesaj yönlendirme hatası: %v", err)
		return
	}

	text := "📥 *Yeni Bağış Bildirimi*\n\n"
	text += "📍 *Grup/Kanal:* " + escapeMarkdown(pending.ChannelTitle) + "\n"
	text += "👤 *Gönderen:* " + escapeMarkdown(pending.SenderName)
	if pending.SenderUser != "" {
		text += " (" + escapeMarkdown(pending.SenderUser) + ")"
	}
	text += "\n"
	text += "📅 *Tarih:* " + pending.MessageDate.Format("02.01.2006 15:04") + "\n"
	text += "🔗 *Link:* " + pending.MessageLink

	// Inline butonlar oluştur
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Ekle", "ekle:"+pendingID),
			tgbotapi.NewInlineKeyboardButtonData("❌ Atla", "atla:"+pendingID),
		),
	)

	infoMsg := tgbotapi.NewMessage(b.cfg.AdminChatID, text)
	infoMsg.ParseMode = "Markdown"
	infoMsg.DisableWebPagePreview = true
	infoMsg.ReplyMarkup = keyboard

	_, err = b.api.Send(infoMsg)
	if err != nil {
		log.Printf("Bilgi mesajı gönderme hatası: %v", err)
	}
}

func escapeMarkdown(text string) string {
	replacer := strings.NewReplacer(
		"_", "\\_",
		"*", "\\*",
		"[", "\\[",
		"]", "\\]",
		"(", "\\(",
		")", "\\)",
		"~", "\\~",
		"`", "\\`",
		">", "\\>",
		"#", "\\#",
		"+", "\\+",
		"-", "\\-",
		"=", "\\=",
		"|", "\\|",
		"{", "\\{",
		"}", "\\}",
		".", "\\.",
		"!", "\\!",
	)
	return replacer.Replace(text)
}

func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}

func itoa64(n int64) string {
	return fmt.Sprintf("%d", n)
}
