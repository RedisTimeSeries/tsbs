# Go parameters
GOCMD=GO111MODULE=on go
GOBUILD=$(GOCMD) build
GOINSTALL=$(GOCMD) install
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt
DISTDIR = ./dist

# DOCKER
DOCKER_APP_NAME=tsbs
DOCKER_ORG=redisbench
DOCKER_REPO:=${DOCKER_ORG}/${DOCKER_APP_NAME}
DOCKER_IMG:="$(DOCKER_REPO):$(DOCKER_TAG)"
DOCKER_LATEST:="${DOCKER_REPO}:latest"

.PHONY: all generators loaders runners
all: generators loaders runners

fmt:
	$(GOFMT) ./cmd/tsbs_load_redistimeseries/*.go
	$(GOFMT) ./cmd/tsbs_run_queries_redistimeseries/*.go

generators: tsbs_generate_data tsbs_generate_queries

influx: tsbs_generate_data tsbs_generate_queries tsbs_load_influx tsbs_run_queries_influx

redistimeseries: tsbs_generate_data tsbs_generate_queries tsbs_load_redistimeseries tsbs_run_queries_redistimeseries

loaders: tsbs_load_redistimeseries tsbs_load_cassandra tsbs_load_clickhouse tsbs_load_influx tsbs_load_mongo tsbs_load_siridb tsbs_load_timescaledb

runners: tsbs_run_queries_redistimeseries tsbs_run_queries_cassandra tsbs_run_queries_clickhouse tsbs_run_queries_influx tsbs_run_queries_mongo tsbs_run_queries_siridb tsbs_run_queries_timescaledb

datagen-rts:
	./scripts/generate_queries.sh

test-rts:
	$(GOTEST) -count=1 ./pkg/targets/redistimeseries/.

test:
	$(GOTEST) -v -race -coverprofile=coverage.txt -covermode=atomic ./...

tsbs_%: $(wildcard ./cmd/$@/*.go)
	$(GOGET) ./cmd/$@
	$(GOBUILD) -o bin/$@ ./cmd/$@
	$(GOINSTALL) ./cmd/$@

# DOCKER TASKS
# Build the container
docker-build:
	docker build -t $(DOCKER_APP_NAME):latest -f  docker/Dockerfile .

# Build the container without caching
docker-build-nc:
	docker build --no-cache -t $(DOCKER_APP_NAME):latest -f docker/Dockerfile .

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

release-redistimeseries:
	$(GOGET) github.com/mitchellh/gox
	$(GOGET) github.com/tcnksm/ghr
	GO111MODULE=on gox  -osarch "linux/amd64 darwin/amd64" -output "${DISTDIR}/tsbs_run_queries_redistimeseries_{{.OS}}_{{.Arch}}" ./cmd/tsbs_run_queries_redistimeseries
	GO111MODULE=on gox  -osarch "linux/amd64 darwin/amd64" -output "${DISTDIR}/tsbs_load_redistimeseries_{{.OS}}_{{.Arch}}" ./cmd/tsbs_load_redistimeseries

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
