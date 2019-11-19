.PHONY: all generators loaders runners
all: generators loaders runners

generators: tsbs_generate_data tsbs_generate_queries

loaders: tsbs_load_cassandra tsbs_load_clickhouse tsbs_load_influx tsbs_load_mongo tsbs_load_redistimeseries tsbs_load_siridb tsbs_load_timescaledb

runners: tsbs_run_queries_cassandra tsbs_run_queries_clickhouse tsbs_run_queries_influx tsbs_run_queries_mongo tsbs_run_queries_redistimeseries tsbs_run_queries_siridb tsbs_run_queries_timescaledb


%: $(wildcard ./cmd/$@/*.go)
	go build -o bin/$@ ./cmd/$@
