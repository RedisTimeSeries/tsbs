#!/bin/bash
# showcases the ftsb 3 phases for timescaledb
# - 1) data and query generation
# - 2) data loading/insertion
# - 3) query execution

SCALE=${SCALE:-"10"}
SEED=${SEED:-"123"}
PASSWORD=${PASSWORD:-"password"}

mkdir -p /tmp/bulk_data

# generate data
$GOPATH/bin/tsbs_generate_data --format timescaledb --use-case cpu-only --scale=${SCALE} --seed=${SEED} --file /tmp/bulk_data/timescaledb_data

for queryName in "single-groupby-1-1-1" ; do
  echo "generating query: $queryName"
  if [ ! -f "/tmp/bulk_data/timescaledb_query_$queryName" ]; then
    $GOPATH/bin/tsbs_generate_queries --format timescaledb --use-case cpu-only --scale=${SCALE} --seed=${SEED} \
      --query-type $queryName \
      --file /tmp/bulk_data/timescaledb_query_$queryName
  else
    echo "query file found at /tmp/bulk_data/timescaledb_query_$queryName. No need to regenerate."
  fi
done

# insert benchmark
$GOPATH/bin/tsbs_load_timescaledb --pass=${PASSWORD} \
    --postgres="sslmode=disable port=5433" --db-name=benchmark \
    --host=127.0.0.1 --user=postgres --workers=1 \
    --file=/tmp/bulk_data/timescaledb_data

# queries benchmark
for queryName in "single-groupby-1-1-1" ; do
  echo "generating query: $queryName"
  $GOPATH/bin/tsbs_run_queries_timescaledb --print-responses \
    --pass=${PASSWORD} --postgres="sslmode=disable port=5433" \
    --db-name=benchmark --hosts=127.0.0.1 --user=postgres --workers=1 \
    --file /tmp/bulk_data/timescaledb_query_$queryName
done
