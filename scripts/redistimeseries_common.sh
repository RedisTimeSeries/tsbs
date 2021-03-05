#!/bin/bash

# Database credentials
DATABASE_HOST=${DATABASE_HOST:-"localhost"}
DATABASE_PORT=${DATABASE_PORT:-6379}
PIPELINE=${PIPELINE:-100}
CONNECTIONS=${CONNECTIONS:-50}

# Data folder
BULK_DATA_DIR=${BULK_DATA_DIR:-"/tmp/bulk_data_redistimeseries"}

# Data folder
RESULTS_DIR=${RESULTS_DIR:-"./results"}

# Load parameters
BATCH_SIZE=${BATCH_SIZE:-10000}
# Debug
DEBUG=${DEBUG:-0}

FORMAT="redistimeseries"

SCALE=${SCALE:-"100"}

# How many concurrent worker would load data - match num of cores, or default to 8
NUM_WORKERS=${NUM_WORKERS:-$(grep -c ^processor /proc/cpuinfo 2>/dev/null || echo 8)}
REPORTING_PERIOD=${REPORTING_PERIOD:-1s}

DATA_FILE_NAME=${DATA_FILE_NAME:-${FORMAT}-data.gz}
REPETITIONS=${REPETITIONS:-3}

# Start and stop time for generated timeseries
TS_START=${TS_START:-"2021-01-01T00:00:00Z"}
TS_END=${TS_END:-"2021-01-05T00:00:01Z"}

# Rand seed
SEED=${SEED:-"123"}

# Print timing stats to stderr after this many queries (0 to disable)
QUERIES_PRINT_INTERVAL=${QUERIES_PRINT_INTERVAL:-"1000"}

# How many queries would be run
MAX_QUERIES=${MAX_QUERIES:-"0"}
REPETITIONS=${REPETITIONS:-1}
PREFIX=${PREFIX:-""}

# How many queries would be run
SLEEP_BETWEEN_RUNS=${SLEEP_BETWEEN_RUNS:-"0"}

# What set of data to generate: devops (multiple data), cpu-only (cpu-usage data)
USE_CASE=${USE_CASE:-"cpu-only"}

# Step to generate data
LOG_INTERVAL=${LOG_INTERVAL:-"10s"}

# Max number of points to generate data. 0 means "use TS_START TS_END with LOG_INTERVAL"
MAX_DATA_POINTS=${MAX_DATA_POINTS:-"0"}

FORMAT=redistimeseries
DATA_FILE_NAME="${BULK_DATA_DIR}/data_${FORMAT}_${USE_CASE}_${SCALE}_${TS_START}_${TS_END}_${LOG_INTERVAL}_${SEED}.dat"
