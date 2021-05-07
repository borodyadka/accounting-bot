package accounting_bot

import (
	"database/sql/driver"
	"encoding/json"
)

type Features struct{}

func (f Features) Value() (driver.Value, error) {
	return json.Marshal(f)
}

func (f *Features) Scan(data interface{}) error {
	return json.Unmarshal(data.([]byte), f)
}

// User struct using for save user settings, version and possible multiple chat providers support
type User struct {
	ID         string
	TelegramID int64
	BotVersion int
	Enabled    bool
	Currency   string
	Features   Features
}

type Entry struct {
	ID        string
	Comment   string
	Tags      []string
	Currency  string
	Value     float32
	MessageID int64
	ReplyID   int64 // bot reply message id
}
