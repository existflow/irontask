#!/bin/bash
# Reset server database (CAUTION: Deletes all data!)

# Load DATABASE_URL from .env if it exists
if [ -f .env ]; then
    export $(cat .env | grep DATABASE_URL | xargs)
fi

echo "⚠️  WARNING: This will DELETE ALL DATA in the irontask schema!"
echo "DATABASE_URL: $DATABASE_URL"
read -p "Continue? (yes/no): " confirm

if [ "$confirm" != "yes" ]; then
    echo "Aborted."
    exit 1
fi

echo "Dropping irontask schema..."
psql "$DATABASE_URL" -c "DROP SCHEMA IF EXISTS irontask CASCADE;"

echo "✓ Schema dropped"
echo ""
echo "Now start the server to recreate tables with new schema:"
echo "  ./irontask-server"
