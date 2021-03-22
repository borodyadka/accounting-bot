FROM golang:1.16-alpine AS builder

ENV CGO_ENABLED 0

WORKDIR /app
COPY . /app

RUN go mod vendor && go build -mod vendor -a -ldflags "-w -s" -installsuffix cgo -tags postgres -o ./bin/bot ./cmd

FROM borodyadka/db-migrate:latest AS migrate

FROM alpine:3.12

COPY --from=builder /app/bin/bot /bin/bot
COPY --from=migrate /bin/migrate /bin/migrate
COPY entrypoint.sh /docker-entrypoint.sh
COPY migrations /opt/migrations

ENTRYPOINT ["/docker-entrypoint.sh"]
CMD ["/bin/bot"]
