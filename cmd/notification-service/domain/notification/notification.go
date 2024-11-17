package notification

import "time"

type NotificationID int64

type Notification struct {
	ID     NotificationID `db:"id"`
	UserID int64          `db:"user_id"`
	Text   string         `db:"text"`
	Date   time.Time      `db:"date"`
}

func NewNotification(userID int64, text string, date time.Time) Notification {
	return Notification{
		UserID: userID,
		Text:   text,
		Date:   date,
	}
}
