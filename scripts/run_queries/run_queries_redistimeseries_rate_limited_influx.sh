#!/bin/bash

# Exit immediately if a command exits with a non-zero status.
set -e

EXE_FILE_NAME=./bin/tsbs_run_queries_redistimeseries

# set -x

EXE_DIR=${EXE_DIR:-$(dirname $0)}
source ${EXE_DIR}/../query_common.sh
source ${EXE_DIR}/../redistimeseries_common.sh

# Ensure RESULTS DIR available
mkdir -p ${RESULTS_DIR}
RPS_ARRAY=("cpu-max-all-1":"4139" "cpu-max-all-8":"588" "double-groupby-1":"324" "double-groupby-5":"70" "double-groupby-all":"35" "groupby-orderby-limit":"28" "high-cpu-1":"1816" "high-cpu-all":"18" "lastpoint":"488" "single-groupby-1-1-1":"2320" "single-groupby-1-1-12":"2320" "single-groupby-1-8-1":"4458" "single-groupby-5-1-1":"648" "single-groupby-5-1-12":"648" "single-groupby-5-8-1":"1158")

for FULL_DATA_FILE_NAME in ${BULK_DATA_DIR}/queries_${USE_CASE}_${FORMAT}_${SCALE}_*; do
  for run in $(seq ${REPETITIONS}); do

    DATA_FILE_NAME=$(basename -- "${FULL_DATA_FILE_NAME}")
    OUT_FULL_FILE_NAME="${RESULTS_DIR}/${PREFIX}result_${DATA_FILE_NAME}_${run}.out"
    HDR_FULL_FILE_NAME="${RESULTS_DIR}/${PREFIX}HDR_TXT_result_${DATA_FILE_NAME}_${run}.out"
    MAX_QUERY_RPS="0"
    for rps_settings in "${RPS_ARRAY[@]}"; do
      KEY="${rps_settings%%:*}"
      VALUE="${rps_settings##*:}"
      if [[ $DATA_FILE_NAME == *"$KEY"* ]]; then
        echo "%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%"
        echo "Using rate-limit info to run query $KEY. Input file data $DATA_FILE_NAME"
        echo "Rate limit: $VALUE"
        echo "%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%"
        MAX_QUERY_RPS=$VALUE
      fi
    done

    $EXE_FILE_NAME \
      --file $FULL_DATA_FILE_NAME \
      --max-queries=${MAX_QUERIES} \
      --max-rps=${MAX_QUERY_RPS} \
      --workers=${NUM_WORKERS} \
      --print-interval=${QUERIES_PRINT_INTERVAL} \
      --debug=${DEBUG} \
      --hdr-latencies=${HDR_FULL_FILE_NAME} \
      --host=${DATABASE_HOST}:${DATABASE_PORT} ${CLUSTER_FLAG} |
      tee $OUT_FULL_FILE_NAME

    echo "Sleeping for ${SLEEP_BETWEEN_RUNS} seconds"
    sleep ${SLEEP_BETWEEN_RUNS}
  done
  echo "Sleeping for ${SLEEP_BETWEEN_RUNS} seconds"
  sleep ${SLEEP_BETWEEN_RUNS}

done
