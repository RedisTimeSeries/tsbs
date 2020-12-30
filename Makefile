# Go parameters
GOCMD=GO111MODULE=on go
GOBUILD=$(GOCMD) build
GOINSTALL=$(GOCMD) install
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt

.PHONY: all generators loaders runners lint fmt checkfmt

all: generators loaders runners

redistimeseries: generators tsbs_load_redistimeseries

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
 		 tsbs_load_victoriametrics

runners: tsbs_run_queries_akumuli \
		 tsbs_run_queries_cassandra \
		 tsbs_run_queries_clickhouse \
		 tsbs_run_queries_cratedb \
		 tsbs_run_queries_influx \
		 tsbs_run_queries_mongo \
		 tsbs_run_queries_siridb \
		 tsbs_run_queries_timescaledb \
		 tsbs_run_queries_timestream \
		 tsbs_run_queries_victoriametrics

test:
	$(GOTEST) -v ./...

rts-test:
	$(GOTEST) --count=1 -v ./cmd/tsbs_load_redistimeseries/.
	$(GOTEST) --count=1 -v ./pkg/targets/redistimeseries/.


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


##################################################################################################
# DOCKER TASKS
# Build the container
docker-build:
	docker build -t $(DOCKER_APP_NAME):latest -f  Dockerfile .

# Build the container without caching
docker-build-nc:
	docker build --no-cache -t $(DOCKER_APP_NAME):latest -f Dockerfile .

# Make a release by building and publishing the `{version}` ans `latest` tagged containers to ECR
docker-release: docker-build-nc docker-publish

# Docker publish
docker-publish: docker-publish-latest

## login to DockerHub with credentials found in env
docker-repo-login:
	docker login -u ${DOCKER_USERNAME} -p ${DOCKER_PASSWORD}

## Publish the `latest` tagged container to ECR
docker-publish-latest: docker-tag-latest
	@echo 'publish latest to $(DOCKER_REPO)'
	docker push $(DOCKER_LATEST)

# Docker tagging
docker-tag: docker-tag-latest

## Generate container `{version}` tag
docker-tag-latest:
	@echo 'create tag latest'
	docker tag $(DOCKER_APP_NAME) $(DOCKER_LATEST)