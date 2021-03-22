#!/bin/sh

set -e

# wait for database is ready
while true; do
  /bin/migrate test -d "$DATABASE_URL" && break
  sleep 1
done

PROVIDER=$(echo -n "$DATABASE_URL" | grep --color=never -oiE '^(postgres)' )
/bin/migrate up -s "/opt/migrations/$PROVIDER" -v info -t migrations -d "$DATABASE_URL"

exec "$@"
