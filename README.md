# Accounting Telegram bot

## Usage

Copy `docker-compose.dist.yml` to `docker-compose.yml`, edit as You want and say `docker-compose up -d`.

## Configuration

* `LOG_LEVEL` one of `debug`, `info` (default), `warning`, `error`
* `DATABASE_URL` in format `postgres://user:pass@host:5432/database?sslmode=disable`
* `TELEGRAM_BOT_TOKEN` telegram bot token
* `AUTH_CODE` (optional) some password to keep bot private

## License

[MIT](LICENSE)
