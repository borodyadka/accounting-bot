CREATE TABLE users
(
    "id"          BIGSERIAL NOT NULL PRIMARY KEY,
    "created_at"  TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    "telegram_id" BIGINT    NOT NULL,
    "bot_version" INTEGER   NOT NULL       DEFAULT 0,
    "enabled"     BOOLEAN   NOT NULL       DEFAULT FALSE,
    "currency"    CHAR(3)   NOT NULL       DEFAULT 'USD',
    "features"    JSONB     NOT NULL       DEFAULT '{}'
);
CREATE UNIQUE INDEX u_users_telegram_id ON users ("telegram_id");

CREATE TABLE entries
(
    "id"         BIGSERIAL                              NOT NULL PRIMARY KEY,
    "created_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    "updated_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    "deleted_at" TIMESTAMP WITH TIME ZONE DEFAULT NULL,
    "user_id"    BIGINT                                 NOT NULL REFERENCES "users" ("id") ON DELETE CASCADE,
    "message_id" BIGINT                   DEFAULT NULL,
    "reply_id"   BIGINT                   DEFAULT NULL,
    "currency"   CHAR(3)                                NOT NULL,
    "value"      DECIMAL(10, 2)                         NOT NULL,
    "comment"    VARCHAR(250)                           NOT NULL DEFAULT '',
    "tags"       VARCHAR(128)[] NOT NULL DEFAULT '{}'
);
CREATE INDEX i_entries_user_id ON entries ("user_id");
CREATE UNIQUE INDEX u_entries_message_id ON entries ("user_id", "message_id");
CREATE INDEX i_entries_tags ON entries USING GIN ("tags");
CREATE INDEX i_entries_created_at ON entries ("created_at" ASC);
