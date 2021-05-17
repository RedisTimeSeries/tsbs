#!/bin/bash

# Database credentials
DATABASE_HOST=${DATABASE_HOST:-"localhost"}
DATABASE_NAME=${DATABASE_NAME:-"benchmark"}


# Default queries folder
BULK_DATA_DIR=${BULK_DATA_DIR:-"/tmp/bulk_queries"}

# Results folder
RESULTS_DIR=${RESULTS_DIR:-"./results"}

# Debug
DEBUG=${DEBUG:-0}

# How many queries would be run
MAX_QUERIES=${MAX_QUERIES:-"0"}

# How many concurrent worker would run queries - match num of cores, or default to 4
NUM_WORKERS=${NUM_WORKERS:-$(grep -c ^processor /proc/cpuinfo 2>/dev/null || echo 4)}

# Print timing stats to stderr after this many queries (0 to disable)
QUERIES_PRINT_INTERVAL=${QUERIES_PRINT_INTERVAL:-"0"}

REPETITIONS=${REPETITIONS:-3}