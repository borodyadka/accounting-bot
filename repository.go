package accounting_bot

import (
	"context"
	"time"
)

type Repository interface {
	SaveUser(ctx context.Context, user *User) (*User, error)
	GetUserByTelegramID(ctx context.Context, id int64) (*User, error)
	SaveEntry(ctx context.Context, user *User, command *Entry) (*Entry, error)
	SaveReplyID(ctx context.Context, user *User, message, reply int64) error
	GetAllEntries(ctx context.Context, user *User, from time.Time, tags []string) ([]*Entry, error)
	AddTag(ctx context.Context, user *User, search string, tags []string) error
	RemoveTag(ctx context.Context, user *User, tags []string) error
	ListTag(ctx context.Context, user *User, search []string) ([]string, error)
}
