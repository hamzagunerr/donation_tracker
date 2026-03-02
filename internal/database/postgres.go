package database

import (
	"context"
	"fmt"
	"time"

	"github.com/hayratyardim/donation_tracker/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DB struct {
	pool *pgxpool.Pool
}

func New(ctx context.Context, databaseURL string) (*DB, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("veritabanı bağlantısı kurulamadı: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("veritabanı ping başarısız: %w", err)
	}

	db := &DB{pool: pool}
	if err := db.migrate(ctx); err != nil {
		return nil, fmt.Errorf("migration başarısız: %w", err)
	}

	return db, nil
}

func (db *DB) Close() {
	db.pool.Close()
}

func (db *DB) migrate(ctx context.Context) error {
	// Tablo oluşturma
	createTables := `
	CREATE TABLE IF NOT EXISTS channels (
		id SERIAL PRIMARY KEY,
		channel_id BIGINT UNIQUE NOT NULL,
		title VARCHAR(255),
		username VARCHAR(255),
		active BOOLEAN DEFAULT true,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS donations (
		id SERIAL PRIMARY KEY,
		message_id INTEGER NOT NULL,
		channel_id BIGINT NOT NULL,
		channel_title VARCHAR(255),
		sender_name VARCHAR(255),
		sender_username VARCHAR(255),
		content TEXT,
		message_link VARCHAR(500),
		message_date TIMESTAMP,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(message_id, channel_id)
	);
	`

	if _, err := db.pool.Exec(ctx, createTables); err != nil {
		return err
	}

	// Mevcut tabloya yeni sütunları ekle (varsa hata vermez)
	alterTable := `
	ALTER TABLE donations ADD COLUMN IF NOT EXISTS added_to_calendar BOOLEAN DEFAULT false;
	ALTER TABLE donations ADD COLUMN IF NOT EXISTS calendar_added_at TIMESTAMP;
	`

	if _, err := db.pool.Exec(ctx, alterTable); err != nil {
		return err
	}

	// İndexler
	createIndexes := `
	CREATE INDEX IF NOT EXISTS idx_donations_channel_id ON donations(channel_id);
	CREATE INDEX IF NOT EXISTS idx_donations_message_date ON donations(message_date);
	CREATE INDEX IF NOT EXISTS idx_donations_calendar ON donations(added_to_calendar);
	`

	_, err := db.pool.Exec(ctx, createIndexes)
	return err
}

func (db *DB) AddDonation(ctx context.Context, d *models.Donation) error {
	query := `
		INSERT INTO donations (message_id, channel_id, channel_title, sender_name, sender_username, content, message_link, message_date)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (message_id, channel_id) DO NOTHING
		RETURNING id
	`

	err := db.pool.QueryRow(ctx, query,
		d.MessageID,
		d.ChannelID,
		d.ChannelTitle,
		d.SenderName,
		d.SenderUser,
		d.Content,
		d.MessageLink,
		d.MessageDate,
	).Scan(&d.ID)

	if err != nil && err.Error() == "no rows in result set" {
		return fmt.Errorf("bu mesaj zaten veritabanında mevcut")
	}

	return err
}

func (db *DB) IsDuplicate(ctx context.Context, messageID int, channelID int64) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM donations WHERE message_id = $1 AND channel_id = $2)`
	err := db.pool.QueryRow(ctx, query, messageID, channelID).Scan(&exists)
	return exists, err
}

func (db *DB) GetDonations(ctx context.Context, limit int) ([]models.Donation, error) {
	query := `
		SELECT id, message_id, channel_id, channel_title, sender_name, sender_username, content, message_link, message_date, created_at, added_to_calendar, calendar_added_at
		FROM donations
		ORDER BY created_at DESC
		LIMIT $1
	`

	rows, err := db.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var donations []models.Donation
	for rows.Next() {
		var d models.Donation
		err := rows.Scan(
			&d.ID,
			&d.MessageID,
			&d.ChannelID,
			&d.ChannelTitle,
			&d.SenderName,
			&d.SenderUser,
			&d.Content,
			&d.MessageLink,
			&d.MessageDate,
			&d.CreatedAt,
			&d.AddedToCalendar,
			&d.CalendarAddedAt,
		)
		if err != nil {
			return nil, err
		}
		donations = append(donations, d)
	}

	return donations, nil
}

func (db *DB) GetAllDonations(ctx context.Context) ([]models.Donation, error) {
	query := `
		SELECT id, message_id, channel_id, channel_title, sender_name, sender_username, content, message_link, message_date, created_at, added_to_calendar, calendar_added_at
		FROM donations
		ORDER BY message_date DESC
	`

	rows, err := db.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var donations []models.Donation
	for rows.Next() {
		var d models.Donation
		err := rows.Scan(
			&d.ID,
			&d.MessageID,
			&d.ChannelID,
			&d.ChannelTitle,
			&d.SenderName,
			&d.SenderUser,
			&d.Content,
			&d.MessageLink,
			&d.MessageDate,
			&d.CreatedAt,
			&d.AddedToCalendar,
			&d.CalendarAddedAt,
		)
		if err != nil {
			return nil, err
		}
		donations = append(donations, d)
	}

	return donations, nil
}

func (db *DB) SearchDonations(ctx context.Context, keyword string) ([]models.Donation, error) {
	query := `
		SELECT id, message_id, channel_id, channel_title, sender_name, sender_username, content, message_link, message_date, created_at, added_to_calendar, calendar_added_at
		FROM donations
		WHERE content ILIKE $1 OR sender_name ILIKE $1 OR channel_title ILIKE $1
		ORDER BY message_date DESC
		LIMIT 20
	`

	searchTerm := "%" + keyword + "%"
	rows, err := db.pool.Query(ctx, query, searchTerm)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var donations []models.Donation
	for rows.Next() {
		var d models.Donation
		err := rows.Scan(
			&d.ID,
			&d.MessageID,
			&d.ChannelID,
			&d.ChannelTitle,
			&d.SenderName,
			&d.SenderUser,
			&d.Content,
			&d.MessageLink,
			&d.MessageDate,
			&d.CreatedAt,
			&d.AddedToCalendar,
			&d.CalendarAddedAt,
		)
		if err != nil {
			return nil, err
		}
		donations = append(donations, d)
	}

	return donations, nil
}

func (db *DB) AddChannel(ctx context.Context, c *models.Channel) error {
	query := `
		INSERT INTO channels (channel_id, title, username)
		VALUES ($1, $2, $3)
		ON CONFLICT (channel_id) DO UPDATE SET title = $2, username = $3, active = true
	`

	_, err := db.pool.Exec(ctx, query, c.ChannelID, c.Title, c.Username)
	return err
}

func (db *DB) GetChannels(ctx context.Context) ([]models.Channel, error) {
	query := `
		SELECT id, channel_id, title, username, active, created_at
		FROM channels
		WHERE active = true
		ORDER BY created_at DESC
	`

	rows, err := db.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var channels []models.Channel
	for rows.Next() {
		var c models.Channel
		err := rows.Scan(
			&c.ID,
			&c.ChannelID,
			&c.Title,
			&c.Username,
			&c.Active,
			&c.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		channels = append(channels, c)
	}

	return channels, nil
}

func (db *DB) GetDonationsByDateRange(ctx context.Context, start, end time.Time) ([]models.Donation, error) {
	query := `
		SELECT id, message_id, channel_id, channel_title, sender_name, sender_username, content, message_link, message_date, created_at, added_to_calendar, calendar_added_at
		FROM donations
		WHERE message_date BETWEEN $1 AND $2
		ORDER BY message_date DESC
	`

	rows, err := db.pool.Query(ctx, query, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var donations []models.Donation
	for rows.Next() {
		var d models.Donation
		err := rows.Scan(
			&d.ID,
			&d.MessageID,
			&d.ChannelID,
			&d.ChannelTitle,
			&d.SenderName,
			&d.SenderUser,
			&d.Content,
			&d.MessageLink,
			&d.MessageDate,
			&d.CreatedAt,
			&d.AddedToCalendar,
			&d.CalendarAddedAt,
		)
		if err != nil {
			return nil, err
		}
		donations = append(donations, d)
	}

	return donations, nil
}

func (db *DB) GetDonationByID(ctx context.Context, id int64) (*models.Donation, error) {
	query := `
		SELECT id, message_id, channel_id, channel_title, sender_name, sender_username, content, message_link, message_date, created_at, added_to_calendar, calendar_added_at
		FROM donations
		WHERE id = $1
	`

	var d models.Donation
	err := db.pool.QueryRow(ctx, query, id).Scan(
		&d.ID,
		&d.MessageID,
		&d.ChannelID,
		&d.ChannelTitle,
		&d.SenderName,
		&d.SenderUser,
		&d.Content,
		&d.MessageLink,
		&d.MessageDate,
		&d.CreatedAt,
		&d.AddedToCalendar,
		&d.CalendarAddedAt,
	)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (db *DB) AddToCalendar(ctx context.Context, donationID int64) error {
	query := `
		UPDATE donations 
		SET added_to_calendar = true, calendar_added_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`
	_, err := db.pool.Exec(ctx, query, donationID)
	return err
}

func (db *DB) RemoveFromCalendar(ctx context.Context, donationID int64) error {
	query := `
		UPDATE donations 
		SET added_to_calendar = false, calendar_added_at = NULL
		WHERE id = $1
	`
	_, err := db.pool.Exec(ctx, query, donationID)
	return err
}

func (db *DB) GetCalendarDonations(ctx context.Context, addedToCalendar bool, limit int) ([]models.Donation, error) {
	query := `
		SELECT id, message_id, channel_id, channel_title, sender_name, sender_username, content, message_link, message_date, created_at, added_to_calendar, calendar_added_at
		FROM donations
		WHERE added_to_calendar = $1
		ORDER BY message_date DESC
		LIMIT $2
	`

	rows, err := db.pool.Query(ctx, query, addedToCalendar, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var donations []models.Donation
	for rows.Next() {
		var d models.Donation
		err := rows.Scan(
			&d.ID,
			&d.MessageID,
			&d.ChannelID,
			&d.ChannelTitle,
			&d.SenderName,
			&d.SenderUser,
			&d.Content,
			&d.MessageLink,
			&d.MessageDate,
			&d.CreatedAt,
			&d.AddedToCalendar,
			&d.CalendarAddedAt,
		)
		if err != nil {
			return nil, err
		}
		donations = append(donations, d)
	}

	return donations, nil
}

func (db *DB) GetCalendarStats(ctx context.Context) (added int, notAdded int, err error) {
	query := `
		SELECT 
			COUNT(*) FILTER (WHERE added_to_calendar = true) as added,
			COUNT(*) FILTER (WHERE added_to_calendar = false) as not_added
		FROM donations
	`
	err = db.pool.QueryRow(ctx, query).Scan(&added, &notAdded)
	return
}
