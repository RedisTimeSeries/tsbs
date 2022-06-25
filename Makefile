# Go parameters
GOCMD=GO111MODULE=on go
GOBUILD=$(GOCMD) build
GOINSTALL=$(GOCMD) install
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt
DISTDIR= ./dist

.PHONY: all generators loaders runners lint fmt checkfmt

all: generators loaders runners

generators: tsbs_generate_data \
			tsbs_generate_queries

loaders: tsbs_load \
		 tsbs_load_akumuli \
		 tsbs_load_cassandra \
		 tsbs_load_clickhouse \
		 tsbs_load_cratedb \
		 tsbs_load_influx \
 		 tsbs_load_mongo \
 		 tsbs_load_prometheus \
 		 tsbs_load_redistimeseries \
 		 tsbs_load_siridb \
 		 tsbs_load_timescaledb \
 		 tsbs_load_victoriametrics \
 		 tsbs_load_questdb

runners: tsbs_run_queries_akumuli \
		 tsbs_run_queries_cassandra \
		 tsbs_run_queries_clickhouse \
		 tsbs_run_queries_cratedb \
		 tsbs_run_queries_influx \
		 tsbs_run_queries_mongo \
		 tsbs_run_queries_redistimeseries \
		 tsbs_run_queries_siridb \
		 tsbs_run_queries_timescaledb \
		 tsbs_run_queries_timestream \
		 tsbs_run_queries_victoriametrics \
		 tsbs_run_queries_questdb

test:
	$(GOTEST) -v ./...

coverage:
	$(GOTEST) -race -coverprofile=coverage.txt -covermode=atomic ./...

tsbs_%: $(wildcard ./cmd/$@/*.go)
	$(GOGET) ./cmd/$@
	$(GOBUILD) -o bin/$@ ./cmd/$@
	$(GOINSTALL) ./cmd/$@

checkfmt:
	@echo 'Checking gofmt';\
 	bash -c "diff -u <(echo -n) <(gofmt -d .)";\
	EXIT_CODE=$$?;\
	if [ "$$EXIT_CODE"  -ne 0 ]; then \
		echo '$@: Go files must be formatted with gofmt'; \
	fi && \
	exit $$EXIT_CODE

lint:
	$(GOGET) github.com/golangci/golangci-lint/cmd/golangci-lint
	golangci-lint run

fmt:
	$(GOFMT) ./...

release-redistimeseries:
	$(GOGET) github.com/mitchellh/gox
	$(GOGET) github.com/tcnksm/ghr
	GO111MODULE=on gox  -osarch "linux/amd64 darwin/amd64" -output "${DISTDIR}/tsbs_run_queries_redistimeseries_{{.OS}}_{{.Arch}}" ./cmd/tsbs_run_queries_redistimeseries
	GO111MODULE=on gox  -osarch "linux/amd64 darwin/amd64" -output "${DISTDIR}/tsbs_load_redistimeseries_{{.OS}}_{{.Arch}}" ./cmd/tsbs_load_redistimeseries


redistimeseries: tsbs_generate_data tsbs_generate_queries tsbs_load_redistimeseries tsbs_run_queries_redistimeseries

publish-redistimeseries: release-redistimeseries
	@for f in $(shell ls ${DISTDIR}); \
	do \
	echo "copying ${DISTDIR}/$${f}"; \
	aws s3 cp ${DISTDIR}/$${f} s3://benchmarks.redislabs/redistimeseries/tools/tsbs/$${f} --acl public-read; \
	done

publish-redistimeseries-queries:
	@for f in $(shell ls /tmp/bulk_queries); \
	do \
	echo "copying $${f}"; \
	aws s3 cp /tmp/bulk_queries/$${f} s3://benchmarks.redislabs/redistimeseries/tsbs/queries/devops/scale100/devops-scale100-4days/$${f} --acl public-read; \
	done

publish-redistimeseries-data:
	@for f in $(shell ls /tmp/bulk_data_redistimeseries); \
	do \
	echo "copying $${f}"; \
	aws s3 cp /tmp/bulk_data_redistimeseries/$${f} s3://benchmarks.redislabs/redistimeseries/tsbs/devops/bulk_data_redistimeseries/$${f} --acl public-read; \
	done