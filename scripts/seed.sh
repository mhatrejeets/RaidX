#!/usr/bin/env sh
set -eu

MONGO_URI="${MONGO_URI:-mongodb://mongo:27017/raidx}"
SEED_DATA_DIR="${SEED_DATA_DIR:-/seed-data}"
MAX_WAIT_SECONDS="${MAX_WAIT_SECONDS:-60}"

log() {
  echo "[seed] $1"
}

wait_for_mongo() {
  elapsed=0
  log "Waiting for MongoDB at ${MONGO_URI} ..."
  while [ "$elapsed" -lt "$MAX_WAIT_SECONDS" ]; do
    if mongosh "$MONGO_URI" --quiet --eval "db.runCommand({ ping: 1 }).ok" >/dev/null 2>&1; then
      log "MongoDB is ready."
      return 0
    fi
    sleep 2
    elapsed=$((elapsed + 2))
  done

  log "MongoDB did not become ready within ${MAX_WAIT_SECONDS}s"
  return 1
}

seed_collection() {
  collection="$1"
  file="$2"

  if [ ! -f "$file" ]; then
    log "Skipping ${collection}: file not found at ${file}"
    return 0
  fi

  log "Seeding ${collection} from ${file} (idempotent upsert by _id)"
  mongoimport \
    --uri "$MONGO_URI" \
    --collection "$collection" \
    --file "$file" \
    --jsonArray \
    --mode upsert \
    --upsertFields _id
}

main() {
  wait_for_mongo

  seed_collection "players" "${SEED_DATA_DIR}/raidx.players.json"
  seed_collection "rbac_teams" "${SEED_DATA_DIR}/raidx.rbac_teams.json"

  log "Seeding completed."
}

main "$@"
