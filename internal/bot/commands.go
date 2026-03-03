package bot

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/hayratyardim/donation_tracker/internal/export"
	"github.com/hayratyardim/donation_tracker/internal/models"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (b *Bot) handleCommand(ctx context.Context, msg *tgbotapi.Message) {
	command := msg.Command()
	args := msg.CommandArguments()

	switch command {
	case "start":
		b.cmdStart(ctx, msg)
	case "help":
		b.cmdHelp(ctx, msg)
	case "ekle":
		b.cmdEkle(ctx, msg)
	case "atla":
		b.cmdAtla(ctx, msg)
	case "liste":
		b.cmdListe(ctx, msg)
	case "ara":
		b.cmdAra(ctx, msg, args)
	case "kanallar", "gruplar":
		b.cmdGruplar(ctx, msg)
	case "export":
		b.cmdExport(ctx, msg)
	case "takvim":
		b.cmdTakvim(ctx, msg)
	case "chatid":
		b.cmdChatID(ctx, msg)
	case "myid":
		b.cmdMyID(ctx, msg)
	case "kofte":
		b.cmdKofte(ctx, msg)
	default:
		b.sendMessage(msg.Chat.ID, "❌ Bilinmeyen komut. /help yazarak komutları görebilirsiniz.")
	}
}

func (b *Bot) cmdStart(ctx context.Context, msg *tgbotapi.Message) {
	text := `🎉 *Bağış Takip Botu'na Hoş Geldiniz\!*

Bu bot, grup ve kanallardan gelen bağış bildirimlerini takip etmenizi ve veritabanına kaydetmenizi sağlar\.

📋 *Komutlar için:* /help`

	b.sendMarkdownMessage(msg.Chat.ID, text)
}

func (b *Bot) cmdHelp(ctx context.Context, msg *tgbotapi.Message) {
	text := `📖 *Komut Listesi*

💾 *Kayıt İşlemleri:*
Gelen mesajların altındaki butonları kullanın:
  ✅ Ekle \- Veritabanına ekler
  ❌ Atla \- Mesajı atlar

📊 *Listeleme Komutları:*
/liste \- Son 10 kaydı listele
/ara \[kelime\] \- Kayıtlarda arama yap

📅 *Takvim Komutları:*
/takvim \- Takvim durumunu göster

📁 *Grup/Kanal Komutları:*
/gruplar \- Takip edilen grupları/kanalları listele

📤 *Dışa Aktarma:*
/export \- Tüm kayıtları Excel olarak indir

🔧 *Diğer:*
/chatid \- Chat ID'nizi öğrenin
/help \- Bu yardım mesajı`

	b.sendMarkdownMessage(msg.Chat.ID, text)
}

func (b *Bot) cmdEkle(ctx context.Context, msg *tgbotapi.Message) {
	b.sendMessage(msg.Chat.ID, "ℹ️ Artık mesajların altındaki butonları kullanabilirsiniz.\n\n✅ Ekle - Veritabanına ekler\n❌ Atla - Mesajı atlar")
}

func (b *Bot) cmdAtla(ctx context.Context, msg *tgbotapi.Message) {
	b.sendMessage(msg.Chat.ID, "ℹ️ Artık mesajların altındaki butonları kullanabilirsiniz.\n\n✅ Ekle - Veritabanına ekler\n❌ Atla - Mesajı atlar")
}

func (b *Bot) cmdListe(ctx context.Context, msg *tgbotapi.Message) {
	donations, err := b.db.GetDonations(ctx, 10)
	if err != nil {
		b.sendMessage(msg.Chat.ID, "❌ Kayıtlar alınırken hata: "+err.Error())
		return
	}

	if len(donations) == 0 {
		b.sendMessage(msg.Chat.ID, "📭 Henüz kayıt bulunmuyor.")
		return
	}

	text := "📋 *Son 10 Kayıt:*\n\n"
	for i, d := range donations {
		text += fmt.Sprintf("*%d\\.* %s\n", i+1, escapeMarkdown(truncate(d.Content, 50)))
		text += fmt.Sprintf("   📍 %s\n", escapeMarkdown(d.ChannelTitle))
		text += fmt.Sprintf("   📅 %s\n", d.MessageDate.Format("02\\.01\\.2006 15:04"))
		text += fmt.Sprintf("   🔗 [Mesaja Git](%s)\n\n", d.MessageLink)
	}

	b.sendMarkdownMessage(msg.Chat.ID, text)
}

func (b *Bot) cmdAra(ctx context.Context, msg *tgbotapi.Message, args string) {
	if args == "" {
		b.sendMessage(msg.Chat.ID, "❌ Arama kelimesi belirtmelisiniz.\nKullanım: /ara [kelime]")
		return
	}

	donations, err := b.db.SearchDonations(ctx, args)
	if err != nil {
		b.sendMessage(msg.Chat.ID, "❌ Arama hatası: "+err.Error())
		return
	}

	if len(donations) == 0 {
		b.sendMessage(msg.Chat.ID, "🔍 \""+args+"\" için sonuç bulunamadı.")
		return
	}

	text := fmt.Sprintf("🔍 *\"%s\" için %d sonuç:*\n\n", escapeMarkdown(args), len(donations))
	for i, d := range donations {
		if i >= 10 {
			text += fmt.Sprintf("\n_\\.\\.\\. ve %d sonuç daha_", len(donations)-10)
			break
		}
		text += fmt.Sprintf("*%d\\.* %s\n", i+1, escapeMarkdown(truncate(d.Content, 50)))
		text += fmt.Sprintf("   📍 %s\n", escapeMarkdown(d.ChannelTitle))
		text += fmt.Sprintf("   📅 %s\n", d.MessageDate.Format("02\\.01\\.2006 15:04"))
		text += fmt.Sprintf("   🔗 [Mesaja Git](%s)\n\n", d.MessageLink)
	}

	b.sendMarkdownMessage(msg.Chat.ID, text)
}

func (b *Bot) cmdGruplar(ctx context.Context, msg *tgbotapi.Message) {
	channels, err := b.db.GetChannels(ctx)
	if err != nil {
		b.sendMessage(msg.Chat.ID, "❌ Gruplar/kanallar alınırken hata: "+err.Error())
		return
	}

	if len(channels) == 0 {
		b.sendMessage(msg.Chat.ID, "📭 Henüz kayıtlı grup/kanal bulunmuyor.\n\nBot bir gruba/kanala eklendiğinde otomatik olarak kaydedilecektir.")
		return
	}

	text := "📺 *Kayıtlı Gruplar/Kanallar:*\n\n"
	for i, c := range channels {
		// Grup/kanal linki oluştur
		var link string
		if c.Username != "" {
			link = "https://t.me/" + c.Username
		} else {
			chatID := c.ChannelID
			if chatID < 0 {
				chatID = -chatID - 1000000000000
			}
			link = fmt.Sprintf("https://t.me/c/%d/1", chatID)
		}

		text += fmt.Sprintf("*%d\\.* [%s](%s)\n", i+1, escapeMarkdown(c.Title), link)
	}

	b.sendMarkdownMessage(msg.Chat.ID, text)
}

func (b *Bot) cmdExport(ctx context.Context, msg *tgbotapi.Message) {
	donations, err := b.db.GetAllDonations(ctx)
	if err != nil {
		b.sendMessage(msg.Chat.ID, "❌ Kayıtlar alınırken hata: "+err.Error())
		return
	}

	if len(donations) == 0 {
		b.sendMessage(msg.Chat.ID, "📭 Dışa aktarılacak kayıt bulunmuyor.")
		return
	}

	b.sendMessage(msg.Chat.ID, "📤 Excel dosyası hazırlanıyor...")

	fileData, fileName, err := export.ToExcel(donations)
	if err != nil {
		b.sendMessage(msg.Chat.ID, "❌ Excel oluşturma hatası: "+err.Error())
		return
	}

	doc := tgbotapi.FileBytes{
		Name:  fileName,
		Bytes: fileData,
	}

	docMsg := tgbotapi.NewDocument(msg.Chat.ID, doc)
	docMsg.Caption = fmt.Sprintf("📊 Toplam %d kayıt dışa aktarıldı.", len(donations))

	_, err = b.api.Send(docMsg)
	if err != nil {
		log.Printf("Excel gönderme hatası: %v", err)
		b.sendMessage(msg.Chat.ID, "❌ Dosya gönderme hatası: "+err.Error())
	}
}

func (b *Bot) cmdTakvim(ctx context.Context, msg *tgbotapi.Message) {
	added, notAdded, err := b.db.GetCalendarStats(ctx)
	if err != nil {
		b.sendMessage(msg.Chat.ID, "❌ İstatistikler alınırken hata: "+err.Error())
		return
	}

	text := "📅 Takvim Durumu\n\n"
	text += fmt.Sprintf("✅ Takvime eklenen: %d\n", added)
	text += fmt.Sprintf("⏳ Bekleyen: %d\n\n", notAdded)

	// Son eklenenler
	addedDonations, _ := b.db.GetCalendarDonations(ctx, true, 5)
	if len(addedDonations) > 0 {
		text += "📋 Son Eklenenler:\n"
		for _, d := range addedDonations {
			content := sanitizeUTF8(truncate(d.Content, 30))
			text += fmt.Sprintf("  • %s (%s)\n",
				content,
				d.MessageDate.Format("02.01"))
		}
		text += "\n"
	}

	// Bekleyenler
	pendingDonations, _ := b.db.GetCalendarDonations(ctx, false, 5)
	if len(pendingDonations) > 0 {
		text += "⏳ Bekleyenler:\n"
		for _, d := range pendingDonations {
			content := sanitizeUTF8(truncate(d.Content, 30))
			text += fmt.Sprintf("  • %s (%s)\n",
				content,
				d.MessageDate.Format("02.01"))
		}
	}

	b.sendMessage(msg.Chat.ID, text)
}

func (b *Bot) cmdChatID(ctx context.Context, msg *tgbotapi.Message) {
	text := fmt.Sprintf("🆔 Chat ID'niz: `%d`", msg.Chat.ID)
	b.sendMarkdownMessage(msg.Chat.ID, text)
}

func (b *Bot) cmdMyID(ctx context.Context, msg *tgbotapi.Message) {
	text := fmt.Sprintf("🆔 Kullanıcı ID'niz: `%d`", msg.From.ID)
	b.sendMarkdownMessage(msg.Chat.ID, text)
}

func (b *Bot) cmdKofte(ctx context.Context, msg *tgbotapi.Message) {
	// Sadece ana admin kullanabilir
	if msg.Chat.ID != b.cfg.AdminChatID {
		return
	}

	err := b.db.ResetAll(ctx)
	if err != nil {
		b.sendMessage(msg.Chat.ID, "❌ Sıfırlama hatası: "+err.Error())
		return
	}

	// Bellekteki pending mesajları da temizle
	b.mu.Lock()
	b.pendingMessages = make(map[string]*models.PendingMessage)
	b.lastMessages = make(map[int64]*models.PendingMessage)
	b.mu.Unlock()

	b.sendMessage(msg.Chat.ID, "🔄 Veritabanı sıfırlandı! Tüm kayıtlar silindi.")
}

func (b *Bot) sendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	_, err := b.api.Send(msg)
	if err != nil {
		log.Printf("Mesaj gönderme hatası: %v", err)
	}
}

func (b *Bot) sendMarkdownMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "MarkdownV2"
	msg.DisableWebPagePreview = true
	_, err := b.api.Send(msg)
	if err != nil {
		log.Printf("Mesaj gönderme hatası: %v", err)
		msg.ParseMode = ""
		msg.Text = strings.ReplaceAll(text, "\\", "")
		b.api.Send(msg)
	}
}

func truncate(s string, maxLen int) string {
	// UTF-8 safe truncate
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}

func sanitizeUTF8(s string) string {
	// Geçersiz UTF-8 karakterlerini temizle
	result := make([]rune, 0, len(s))
	for _, r := range s {
		if r != '\uFFFD' && r >= 0x20 {
			result = append(result, r)
		}
	}
	return string(result)
}

func init() {
	_ = strconv.Itoa(0)
}
