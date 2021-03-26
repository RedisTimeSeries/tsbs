#!/bin/bash

# Ensure generator is available
EXE_FILE_NAME=./bin/tsbs_generate_data

set -x

EXE_DIR=${EXE_DIR:-$(dirname $0)}
source ${EXE_DIR}/query_common.sh
source ${EXE_DIR}/redistimeseries_common.sh

# Ensure DATA DIR available
mkdir -p ${BULK_DATA_DIR}

echo "Generating ${DATA_FILE_NAME}:"
${EXE_FILE_NAME} \
    --format=${FORMAT} \
    --use-case=${USE_CASE} \
    --scale=${SCALE} \
    --timestamp-start=${TS_START} \
    --timestamp-end=${TS_END} \
    --debug=${DEBUG} \
    --seed=${SEED} \
    --log-interval=${LOG_INTERVAL} \
    --interleaved-generation-groups=${INTERLEAVED_GENERATION_GROUPS} \
    --max-data-points=${MAX_DATA_POINTS} >${DATA_FILE_NAME}
