# Bağış Takip Telegram Botu

Telegram grup ve kanallarından gelen bağış bildirimlerini takip eden ve veritabanına kaydeden bot.

## Özellikler

- Kanallardan gelen mesajları otomatik takip
- Admin onayı ile veritabanına kaydetme
- Duplicate mesaj kontrolü
- Kayıtlarda arama yapabilme
- Excel olarak dışa aktarma

## Kurulum

### 1. Bot Oluşturma

1. Telegram'da [@BotFather](https://t.me/BotFather) ile konuşun
2. `/newbot` komutu ile yeni bot oluşturun
3. Bot token'ını kaydedin

### 2. Yapılandırma

`.env.example` dosyasını `.env` olarak kopyalayın:

```bash
cp .env.example .env
```

`.env` dosyasını düzenleyin:

```env
TELEGRAM_BOT_TOKEN=your_bot_token_here
ADMIN_CHAT_ID=your_chat_id_here
POSTGRES_USER=postgres
POSTGRES_PASSWORD=guclu_bir_sifre
POSTGRES_DB=donation_tracker
```

**Chat ID nasıl bulunur?**
Botu başlattıktan sonra bota `/chatid` yazın.

---

## Docker ile Çalıştırma (Önerilen)

### Gereksinimler
- Docker
- Docker Compose

### Başlatma

```bash
docker compose up -d
```

### Logları İzleme

```bash
docker logs -f donation_tracker_bot
```

### Durdurma

```bash
docker compose down
```

### Yeniden Build (kod değişikliği sonrası)

```bash
docker compose up -d --build
```

---

## Lokal Geliştirme (Docker'sız)

### Gereksinimler
- Go 1.21+
- PostgreSQL

### Veritabanı Oluşturma

```sql
CREATE DATABASE donation_tracker;
```

### .env Dosyası (lokal için)

```env
TELEGRAM_BOT_TOKEN=your_bot_token_here
ADMIN_CHAT_ID=your_chat_id_here
DATABASE_URL=postgres://username:password@localhost:5432/donation_tracker?sslmode=disable
```

### Bağımlılıkları Yükleme

```bash
go mod tidy
```

### Çalıştırma

```bash
go run cmd/bot/main.go
```

## Komutlar

| Komut | Açıklama |
|-------|----------|
| `/start` | Botu başlat |
| `/help` | Yardım mesajı |
| `/ekle` | Bekleyen mesajı veritabanına ekle |
| `/atla` | Bekleyen mesajı atla |
| `/liste` | Son 10 kaydı listele |
| `/ara [kelime]` | Kayıtlarda arama yap |
| `/gruplar` | Takip edilen grupları/kanalları listele |
| `/export` | Tüm kayıtları Excel olarak indir |
| `/chatid` | Chat ID'nizi öğrenin |

## Kullanım

1. Botu oluşturup yapılandırdıktan sonra çalıştırın
2. Botu takip etmek istediğiniz gruba veya kanala **admin olarak** ekleyin
3. Grupta/kanalda yeni mesaj geldiğinde bot size özel mesaj olarak iletecek
4. `/ekle` komutu ile kaydetmek istediğiniz mesajları veritabanına ekleyin
5. `/export` komutu ile kayıtları Excel olarak dışa aktarın

## Mesaj Filtreleme

Bot, ⚡️ emojisi içeren mesajları atlar (bunlar bağış bildirim sonu göstergesi olarak kullanılır).

## Lisans

MIT
