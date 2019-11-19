#!/bin/bash

# Ensure loader is available
EXE_FILE_NAME=${EXE_FILE_NAME:-$(which tsbs_load_redistimeseries)}
if [[ -z "$EXE_FILE_NAME" ]]; then
  echo "tsbs_load_redistimeseries not available. It is not specified explicitly and not found in \$PATH"
  exit 1
fi
FORMAT="redistimeseries"

DATA_FILE_NAME=${DATA_FILE_NAME:-${FORMAT}-data.gz}
DATABASE_PORT=${DATABASE_PORT:-6379}
CONNECTIONS=${CONNECTIONS:-10}
PIPELINE=${PIPELINE:-100}
EXTENSION="${DATA_FILE_NAME##*.}"
DIR=$(dirname "${DATA_FILE_NAME}")
NO_EXT_DATA_FILE_NAME="${DATA_FILE_NAME%.*}"
OUT_FULL_FILE_NAME="${DIR}/load_result_${NO_EXT_DATA_FILE_NAME}.out"
EXE_DIR=${EXE_DIR:-$(dirname $0)}

# Load parameters - common
source ${EXE_DIR}/load_common.sh

# Remove previous database
redis-cli -h ${DATABASE_HOST} -p ${DATABASE_PORT} flushall

# Retrieve command stats output
redis-cli -h ${DATABASE_HOST} -p ${DATABASE_PORT} config resetstat



# Load new data
cat ${DATA_FILE} | gunzip | $EXE_FILE_NAME \
  --workers=${NUM_WORKERS} \
  --batch-size=${BATCH_SIZE} \
  --reporting-period=${REPORTING_PERIOD} \
  --host=${DATABASE_HOST}:${DATABASE_PORT} \
  --connections=${CONNECTIONS} --pipeline=${PIPELINE} |
    tee $OUT_FULL_FILE_NAME

# Retrieve command stats output
redis-cli -h ${DATABASE_HOST} -p ${DATABASE_PORT} info commandstats >> $OUT_FULL_FILE_NAME
redis-cli -h ${DATABASE_HOST} -p ${DATABASE_PORT} info commandstats