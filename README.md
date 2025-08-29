```markdown
# whatsmeow-light

A minimal whatsmeow example that pairs using a QR/pair code, stores session in PostgreSQL and replies "hi" to every incoming message.

## Setup

1. Create a PostgreSQL database. Example:

   createdb whatsmeow
   psql -c "CREATE USER whatsmeow WITH PASSWORD 'whatsmeow'; GRANT ALL PRIVILEGES ON DATABASE whatsmeow TO whatsmeow;"

2. Set the DSN environment variable:

   export POSTGRES_DSN="postgres://whatsmeow:whatsmeow@localhost:5432/whatsmeow?sslmode=disable"

3. Ensure Go dependencies are present. Update go.mod with your whatsmeow module path if needed (some versions use `go.mau.fi/whatsmeow`).

4. Run:

   go run main.go

   On first run you will get a QR in the terminal. Scan it with your WhatsApp to pair.

## Behaviour

- Session and needed data are persisted into PostgreSQL via the sqlstore. On subsequent runs the stored session will be used automatically.
- Each time a text message arrives, this program sends back "hi" to the sender.

## Notes / Adjustments

- Depending on the whatsmeow version in your repository, import paths and some type names may differ (e.g. `go.mau.fi/whatsmeow` vs `github.com/tulir/whatsmeow`, event/message type names). If you see compile errors, replace the whatsmeow-related imports with the correct module path used in your repo and adapt message/event types accordingly.
- The code is intentionally small/light. For production use you should add proper error handling, logging, message deduplication, rate-limiting and graceful session migration/backups.
```
