#!/bin/sh
set -e

# Wait for MySQL to be ready by attempting migrations with retries
echo "Waiting for MySQL and running migrations..."
MAX_RETRIES=30
RETRY_COUNT=0

while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
    if migrate -path /migrations -database "mysql://${DB_USER}:${DB_PASS}@tcp(${DB_HOST}:${DB_PORT})/${DB_NAME}?multiStatements=true" up 2>/dev/null; then
        echo "Migrations completed successfully!"
        break
    fi

    RETRY_COUNT=$((RETRY_COUNT + 1))
    echo "Database not ready yet (attempt $RETRY_COUNT/$MAX_RETRIES), retrying in 2s..."
    sleep 2
done

if [ $RETRY_COUNT -eq $MAX_RETRIES ]; then
    echo "ERROR: Database did not become ready in time"
    exit 1
fi

# Start the application
echo "Starting application..."
exec "$@"
