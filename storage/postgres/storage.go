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

const limit = 100_000

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
	result := &bot.Entry{
		CreatedAt: entry.CreatedAt,
		Comment:   entry.Comment,
		Tags:      entry.Tags[:],
		Currency:  entry.Currency,
		Value:     entry.Value,
		MessageID: entry.MessageID,
		ReplyID:   entry.ReplyID,
	}
	err := s.pg.QueryRow(
		ctx,
		`INSERT INTO "entries"
			("created_at", "user_id", "message_id", "reply_id", "currency", "value", "comment", "tags")
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT ("user_id", "message_id") DO UPDATE
			SET "value" = $6, "comment" = $7, "tags" = $8
		RETURNING "id"::TEXT`,
		entry.CreatedAt, user.ID, entry.MessageID, entry.ReplyID, user.Currency, entry.Value, entry.Comment, entry.Tags,
	).Scan(&result.ID)
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

func (s *Repository) GetAllEntries(
	ctx context.Context, user *bot.User, from time.Time, tags []string,
) ([]*bot.Entry, error) {
	start := from
	cond := []string{`"user_id" = $1`, "created_at > $2"}
	if len(tags) > 0 {
		cond = append(cond, "tags @> $3")
	}

	result := make([]*bot.Entry, 0, limit)
	for {
		args := []interface{}{user.ID, start}
		if len(tags) > 0 {
			args = append(args, tags)
		}

		rows, err := s.pg.Query(
			ctx,
			fmt.Sprintf(
				`SELECT "id"::TEXT, "created_at", "message_id", "reply_id", "currency", "value", "comment", "tags"
				FROM "entries" WHERE %s ORDER BY "created_at" ASC LIMIT %d`,
				strings.Join(cond, " AND "),
				limit,
			),
			args...,
		)
		if err != nil {
			return nil, err
		}

		var count int
		for rows.Next() {
			count++
			entry := &bot.Entry{}
			if err := rows.Scan(
				&entry.ID,
				&entry.CreatedAt,
				&entry.MessageID,
				&entry.ReplyID,
				&entry.Currency,
				&entry.Value,
				&entry.Comment,
				&entry.Tags,
			); err != nil {
				rows.Close()
				return nil, err
			}
			start = entry.CreatedAt
			result = append(result, entry)
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

func New(url string) (bot.Repository, error) {
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
