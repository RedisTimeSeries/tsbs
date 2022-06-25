#!/bin/bash

# Exit immediately if a command exits with a non-zero status.
set -e

# Ensure runner is available
EXE_FILE_NAME=${EXE_FILE_NAME:-$(which tsbs_run_queries_timescaledb)}
if [[ -z "$EXE_FILE_NAME" ]]; then
    echo "tsbs_run_queries_timescaledb not available. It is not specified explicitly and not found in \$PATH"
    exit 1
fi

EXE_DIR=${EXE_DIR:-$(dirname $0)}

source ${EXE_DIR}/run_common.sh

# Ensure RESULTS DIR available
mkdir -p ${RESULTS_DIR}
RPS_ARRAY=("cpu-max-all-1":"4139" "cpu-max-all-8":"588" "double-groupby-1":"324" "double-groupby-5":"70" "double-groupby-all":"35" "groupby-orderby-limit":"28" "high-cpu-1":"1816" "high-cpu-all":"18" "lastpoint":"488" "single-groupby-1-1-1":"2320" "single-groupby-1-1-12":"2320" "single-groupby-1-8-1":"4458" "single-groupby-5-1-1":"648" "single-groupby-5-1-12":"648" "single-groupby-5-8-1":"1158")

for FULL_DATA_FILE_NAME in ${BULK_DATA_DIR}/queries_timescaledb*; do
    # $FULL_DATA_FILE_NAME:  /full/path/to/file_with.ext
    # $DATA_FILE_NAME:       file_with.ext
    # $DIR:                  /full/path/to
    # $EXTENSION:            ext
    # NO_EXT_DATA_FILE_NAME: file_with

    DATA_FILE_NAME=$(basename -- "${FULL_DATA_FILE_NAME}")
    DIR=$(dirname "${FULL_DATA_FILE_NAME}")
    EXTENSION="${DATA_FILE_NAME##*.}"
    NO_EXT_DATA_FILE_NAME="${DATA_FILE_NAME%.*}"
    for run in $(seq ${REPETITIONS}); do

        # Several options on how to name results file
        #OUT_FULL_FILE_NAME="${DIR}/result_${DATA_FILE_NAME}"
        OUT_FULL_FILE_NAME="${RESULTS_DIR}/result_${NO_EXT_DATA_FILE_NAME}_${run}.out"
        #OUT_FULL_FILE_NAME="${DIR}/${NO_EXT_DATA_FILE_NAME}.out"
        HDR_FULL_FILE_NAME="${RESULTS_DIR}/HDR_TXT_result_${NO_EXT_DATA_FILE_NAME}_${run}.out"

        if [ "${EXTENSION}" == "gz" ]; then
            GUNZIP="gunzip"
        else
            GUNZIP="cat"
        fi

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

        echo "Running ${DATA_FILE_NAME}"
        cat $FULL_DATA_FILE_NAME |
            $GUNZIP |
            $EXE_FILE_NAME \
                --max-queries=$MAX_QUERIES \
                --max-rps=${MAX_QUERY_RPS} \
                --workers=$NUM_WORKERS \
                --print-interval=${QUERIES_PRINT_INTERVAL} \
                --hdr-latencies=${HDR_FULL_FILE_NAME} \
                --debug=${DEBUG} \
                --hosts=${DATABASE_HOST} \
                --user=postgres |
            tee $OUT_FULL_FILE_NAME
    done
done
