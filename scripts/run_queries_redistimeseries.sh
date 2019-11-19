#!/bin/bash

# Exit immediately if a command exits with a non-zero status.
set -e

# Ensure runner is available
EXE_FILE_NAME=${EXE_FILE_NAME:-$(which tsbs_run_queries_redistimeseries)}
if [[ -z "$EXE_FILE_NAME" ]]; then
  echo "tsbs_run_queries_redistimeseries not available. It is not specified explicitly and not found in \$PATH"
  exit 1
fi

DATABASE_PORT=${DATABASE_PORT:-6379}

# Queries folder
BULK_DATA_DIR=${BULK_DATA_DIR:-"/tmp/bulk_queries"}

# Print timing stats to stderr after this many queries (0 to disable)
QUERIES_PRINT_INTERVAL=${QUERIES_PRINT_INTERVAL:-"0"}

# How many queries would be run
MAX_QUERIES=${MAX_QUERIES:-"0"}

# How many concurrent worker would run queries - match num of cores, or default to 4
NUM_WORKERS=${NUM_WORKERS:-$(grep -c ^processor /proc/cpuinfo 2>/dev/null || echo 4)}

for FULL_DATA_FILE_NAME in ${BULK_DATA_DIR}/queries_redistimeseries*; do
  # $FULL_DATA_FILE_NAME:  /full/path/to/file_with.ext
  # $DATA_FILE_NAME:       file_with.ext
  # $DIR:                  /full/path/to
  # $EXTENSION:            ext
  # NO_EXT_DATA_FILE_NAME: file_with

  DATA_FILE_NAME=$(basename -- "${FULL_DATA_FILE_NAME}")
  DIR=$(dirname "${FULL_DATA_FILE_NAME}")
  EXTENSION="${DATA_FILE_NAME##*.}"
  NO_EXT_DATA_FILE_NAME="${DATA_FILE_NAME%.*}"

  OUT_FULL_FILE_NAME="${DIR}/result_${NO_EXT_DATA_FILE_NAME}.out"

  if [ "${EXTENSION}" == "gz" ]; then
    GUNZIP="gunzip"
  else
    GUNZIP="cat"
  fi

  echo "Running ${DATA_FILE_NAME}"
  cat $FULL_DATA_FILE_NAME |
    $GUNZIP |
    $EXE_FILE_NAME \
      --max-queries $MAX_QUERIES \
      --workers $NUM_WORKERS \
      --print-interval ${QUERIES_PRINT_INTERVAL} \
      --host=${DATABASE_HOST}:${DATABASE_PORT} |
    tee $OUT_FULL_FILE_NAME
done
