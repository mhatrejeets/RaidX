#!/bin/bash
# Seed MongoDB with JSON files using mongoimport
# Usage: bash scripts/seed.sh

set -e

MONGO_URI=${MONGO_URI:-"mongodb://localhost:27017"}
DB_NAME=${DB_NAME:-"raidx"}
DATA_DIR="dummydata"

mongoimport --uri "$MONGO_URI/$DB_NAME" --collection matches --file "$DATA_DIR/raidx.matches.json" --jsonArray
mongoimport --uri "$MONGO_URI/$DB_NAME" --collection players --file "$DATA_DIR/raidx.players.json" --jsonArray
mongoimport --uri "$MONGO_URI/$DB_NAME" --collection sessions --file "$DATA_DIR/raidx.sessions.json" --jsonArray
mongoimport --uri "$MONGO_URI/$DB_NAME" --collection teams --file "$DATA_DIR/raidx.teams.json" --jsonArray

echo "Seeding complete."
