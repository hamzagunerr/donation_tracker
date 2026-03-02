package models

import "time"

type Donation struct {
	ID              int64      `json:"id"`
	MessageID       int        `json:"message_id"`
	ChannelID       int64      `json:"channel_id"`
	ChannelTitle    string     `json:"channel_title"`
	SenderName      string     `json:"sender_name"`
	SenderUser      string     `json:"sender_username"`
	Content         string     `json:"content"`
	MessageLink     string     `json:"message_link"`
	MessageDate     time.Time  `json:"message_date"`
	CreatedAt       time.Time  `json:"created_at"`
	AddedToCalendar bool       `json:"added_to_calendar"`
	CalendarAddedAt *time.Time `json:"calendar_added_at,omitempty"`
}

type Channel struct {
	ID        int64     `json:"id"`
	ChannelID int64     `json:"channel_id"`
	Title     string    `json:"title"`
	Username  string    `json:"username"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
}

type PendingMessage struct {
	MessageID    int
	ChannelID    int64
	ChannelTitle string
	SenderName   string
	SenderUser   string
	Content      string
	MessageLink  string
	MessageDate  time.Time
}
