package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	bot "github.com/borodyadka/accounting-bot"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

const limit = 1024

type Repository struct {
	pg *pgxpool.Pool
}

func (s *Repository) SaveUser(ctx context.Context, user *bot.User) (*bot.User, error) {
	var id string
	err := s.pg.QueryRow(
		ctx,
		`INSERT INTO "users" ("telegram_id", "bot_version", "enabled", "currency", "features")
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT ("telegram_id") DO UPDATE SET "bot_version" = $2, "enabled" = $3, "currency" = $4, "features" = $5
		RETURNING "id"::TEXT`,
		user.TelegramID, bot.VERSION, user.Enabled, user.Currency, user.Features,
	).Scan(&id)
	if err != nil {
		return nil, err
	}
	return &bot.User{
		ID:         id,
		TelegramID: user.TelegramID,
		BotVersion: bot.VERSION,
		Enabled:    user.Enabled,
		Currency:   user.Currency,
		Features:   user.Features,
	}, nil
}

func (s *Repository) GetUserByTelegramID(ctx context.Context, id int64) (*bot.User, error) {
	user := new(bot.User)
	err := s.pg.QueryRow(
		ctx,
		`SELECT "id"::TEXT, "telegram_id", "bot_version", "enabled", "currency", "features"
		FROM "users"
		WHERE "telegram_id" = $1`,
		id,
	).Scan(&user.ID, &user.TelegramID, &user.BotVersion, &user.Enabled, &user.Currency, &user.Features)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return user, nil
}

func (s *Repository) SaveEntry(ctx context.Context, user *bot.User, entry *bot.Entry) (*bot.Entry, error) {
	result := &bot.Entry{}
	err := s.pg.QueryRow(
		ctx,
		`INSERT INTO "entries" ("user_id", "message_id", "reply_id", "currency", "value", "comment", "tags")
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT ("user_id", "message_id") DO UPDATE
			SET "value" = $5, "comment" = $6, "tags" = $7
		RETURNING "id"::TEXT, "message_id", "reply_id", "currency", "value", "comment", "tags"`,
		user.ID, entry.MessageID, entry.ReplyID, user.Currency, entry.Value, entry.Comment, entry.Tags,
	).Scan(
		&result.ID, &result.MessageID, &result.ReplyID, &result.Currency, &result.Value, &result.Comment, &result.Tags,
	)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *Repository) SaveReplyID(ctx context.Context, user *bot.User, message, reply int64) error {
	_, err := s.pg.Exec(
		ctx,
		`UPDATE "entries" SET "reply_id" = $1 WHERE "user_id" = $2 AND "message_id" = $3`,
		reply, user.ID, message,
	)
	return err
}

func (s *Repository) IterEntries(ctx context.Context, user *bot.User, from time.Time, tags []string) (<-chan *bot.Entry, error) {
	result := make(chan *bot.Entry)
	defer close(result)

	cond := []string{`"user_id" = $1`, "created_at > $2"}
	if len(tags) > 0 {
		cond = append(cond, "tags @> $3")
	}

	start := from
	for {
		rows, err := s.pg.Query(
			ctx,
			fmt.Sprintf(
				`SELECT "id"::TEXT, "created_at", "message_id", "reply_id", "currency", "value", "comment", "tags"
				FROM "entries" WHERE %s ORDER BY "created_at" ASC LIMIT %d`,
				strings.Join(cond, " AND "),
				limit,
			),
			user.ID, start, tags,
		)
		if err != nil {
			return nil, err
		}

		var count int
		for rows.Next() {
			count++
			entry := new(bot.Entry)
			if err := rows.Scan(
				entry.ID,
				&start,
				entry.MessageID,
				entry.ReplyID,
				entry.Currency,
				entry.Value,
				entry.Comment,
				entry.Tags,
			); err != nil {
				rows.Close()
				return nil, err
			}
			result <- entry
		}
		rows.Close()

		if count < limit {
			break
		}
	}

	return result, nil
}

func (s *Repository) AddTag(ctx context.Context, user *bot.User, search string, tags []string) error {
	_, err := s.pg.Exec(
		ctx,
		`UPDATE "entries" SET "tags" = array_cat("tags", $1) WHERE "user_id" = $2 AND $3 = any("tags")`,
		tags, user.ID, search,
	)
	return err
}

func (s *Repository) RemoveTag(ctx context.Context, user *bot.User, tags []string) error {
	tx, err := s.pg.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// TODO: remove and rewrite sql query
	if len(tags) > 10 {
		return errors.New("too many tags")
	}
	// okay, this is a shitty solution to make queries in a loop
	for _, tag := range tags {
		if _, err := tx.Exec(
			ctx,
			`UPDATE "entries" SET "tags" = array_remove("tags", $1::varchar) WHERE "user_id" = $2 AND $1 = any("tags")`,
			tag, user.ID,
		); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (s *Repository) ListTag(ctx context.Context, user *bot.User, search []string) ([]string, error) {
	rows, err := s.pg.Query(
		ctx,
		`SELECT DISTINCT UNNEST("tags") AS "tag"
		FROM "entries" WHERE "user_id" = $1 AND $2 <@ "tags"
		ORDER BY "tag" ASC`,
		user.ID, search,
	)
	if err != nil {
		return nil, err
	}
	tags := make([]string, 0, 32)
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	return tags, nil
}

func New(url string) (bot.Storage, error) {
	config, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, err
	}

	config.MaxConns = 2
	config.MaxConnLifetime = 5 * time.Minute
	config.LazyConnect = true

	pool, err := pgxpool.ConnectConfig(context.Background(), config)
	if err != nil {
		return nil, err
	}
	ping, err := pool.Acquire(context.Background())
	if err != nil {
		return nil, err
	}
	defer ping.Release()

	return &Repository{pg: pool}, nil
}
