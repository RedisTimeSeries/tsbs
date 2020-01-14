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
PREFIX=${PREFIX:-""}

OUT_FULL_FILE_NAME="${DIR}/${PREFIX}_load_result_${NO_EXT_DATA_FILE_NAME}.out"
EXE_DIR=${EXE_DIR:-$(dirname $0)}
COMPRESSION_ENABLED=${COMPRESSION_ENABLED:-true}

# Load parameters - common
source ${EXE_DIR}/load_common.sh

# Remove previous database
redis-cli -h ${DATABASE_HOST} -p ${DATABASE_PORT} flushall

# Retrieve command stats output
redis-cli -h ${DATABASE_HOST} -p ${DATABASE_PORT} config resetstat

echo "Using only 1 worker"
echo "Saving results to $OUT_FULL_FILE_NAME"

# Load new data
cat ${DATA_FILE} | $EXE_FILE_NAME \
  --workers=1 \
  --batch-size=${BATCH_SIZE} \
  --reporting-period=${REPORTING_PERIOD} \
  --host=${DATABASE_HOST}:${DATABASE_PORT} \
  --compression-enabled=${COMPRESSION_ENABLED} \
  --connections=${CONNECTIONS} --pipeline=${PIPELINE} |
    tee $OUT_FULL_FILE_NAME

# Retrieve command stats output
redis-cli -h ${DATABASE_HOST} -p ${DATABASE_PORT} info commandstats >> $OUT_FULL_FILE_NAME
redis-cli -h ${DATABASE_HOST} -p ${DATABASE_PORT} info >> $OUT_FULL_FILE_NAME
redis-cli -h ${DATABASE_HOST} -p ${DATABASE_PORT} info commandstats