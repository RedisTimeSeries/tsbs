# TSBS Supplemental Guide: RedisTimeSeries

RedisTimeSeries is a Redis Module adding a Time Series data structure to Redis. This supplemental guide explains how
the data generated for TSBS is stored, additional flags available when
using the data importer (`tsbs_load_redistimeseries`). **This
should be read *after* the main README.**


---


## Installation

TSBS is a collection of Go programs (with some auxiliary bash and Python
scripts). The easiest way to get and install the Go programs is to use
`go get` and then `go install`:
```bash
# Fetch TSBS and its dependencies
$ go get github.com/timescale/tsbs
$ cd $GOPATH/src/github.com/timescale/tsbs/cmd
$ go get ./...

# Install redistimeseries binaries. 
cd $GOPATH/src/github.com/timescale/tsbs
make
```

## Full cycle TSBS RedisTimeSeries scripts

Instead of calling tsbs redistimeseries binaries directly, we also supply scripts/*.sh for convenience with many of the flags set to a reasonable default for RedisTimeSeries database. 

So for a Full cycle TSBS RedisTimeSeries benchmark, ensure that RedisTimeSeries is running and then use:

# functional full cycle 

```
scripts/functional/debug_responses_full_cycle_minitest_redistimeseries.sh
```


# benchmark commands
```
# generate the dataset 
FORMATS="redistimeseries" SKIP_IF_EXISTS=FALSE  SCALE=100 SEED=123 \
    scripts/generate_data.sh

# generate the queries
FORMATS="redistimeseries" SKIP_IF_EXISTS=FALSE SCALE=4000 SEED=123 \
    scripts/generate_queries.sh

# load the data into RedisTimeSeries
scripts/load_redistimeseries.sh

# benchmark RedisTimeSeries query performance
scripts/run_queries_redistimeseries.sh

```

## Data format

Data generated by `tsbs_generate_data` for RedisTimeSeries is serialized in
`key timestamp value [key timestamp value ...]` format. Each metric reading is composed of `key timestamp value`. In addition to metric readings, 'tags' (including the location of the host, its operating system, etc) are added to each distinct time series at the moment of the first insertion, using [TS.ADD](https://oss.redislabs.com/redistimeseries/commands/#tsadd).

An example for the `cpu-only` use case, on the first metric reading for the `cpu_usage_user{3297394792}`, `cpu_usage_system{3297394792}`, and `cpu_usage_idle{3297394792}` metrics, they would be insert using [TS.ADD](https://oss.redislabs.com/redistimeseries/commands/#tsadd), in the following manner:
```text
TS.ADD cpu_usage_user{3297394792} 1451606400000 58 LABELS hostname host_0 region eu-central-1 datacenter eu-central-1a rack 6 os Ubuntu15.10 arch x86 team SF service 19 service_version 1 service_environment test measurement cpu fieldname usage_user
TS.ADD cpu_usage_system{3297394792} 1451606400000 2 LABELS hostname host_0 region eu-central-1 datacenter eu-central-1a rack 6 os Ubuntu15.10 arch x86 team SF service 19 service_version 1 service_environment test measurement cpu fieldname usage_system
TS.ADD cpu_usage_idle{3297394792} 1451606400000 24 LABELS hostname host_0 region eu-central-1 datacenter eu-central-1a rack 6 os Ubuntu15.10 arch x86 team SF service 19 service_version 1 service_environment test measurement cpu fieldname usage_idle
```

The second and following metric readings for the `cpu_usage_user{3297394792}`, `cpu_usage_system{3297394792}`, and `cpu_usage_idle{3297394792}` metrics would be inserted using [TS.MADD](https://oss.redislabs.com/redistimeseries/commands/#tsmadd) command:

```text
TS.MADD cpu_usage_user{3297394792} 1451606410000 57 cpu_usage_system{3297394792} 1451606410000 3 cpu_usage_idle{3297394792}
TS.MADD cpu_usage_user{3297394792} 1451606420000 58 cpu_usage_system{3297394792} 1451606420000 2 cpu_usage_idle{3297394792}
```

## Query types <a name="tsbs-query-types-mapping"></a> mapping to RedisTimeSeries query language

###  Client side work functors:
 - MergeSeriesOnTimestamp - self explanatory
 - FilterRangesByThresholdAbove - filter a MultiRange ( multiple series merged by timestamp ) on a threshold for one of the metrics in the multiseries
 - ReduceSeriesOnTimestampBy - reduce a MultiRange over a given function ( in our specific case, Max and Avg )

### Devops / cpu-only

#### Simple Rollups 

##### q1) single-groupby-1-1-1
Simple aggregrate (MAX) on one metric for 1 host, every 1 minute for 1 hour

###### Query language
```
TS.MRANGE 1451679382646 1451682982646 AGGREGATION MAX 60000 FILTER measurement=cpu fieldname=usage_user hostname=host_9
```

###### Sample Responses:
- [InfluxDB](./responses/influx_single-groupby-1-1-1.json)
- [TimescaleDB](./responses/timescaledb_single-groupby-1-1-1.json)
- [RedistimeSeries](./responses/redistimeseries_single-groupby-1-1-1.json)

##### q2) single-groupby-1-1-12

Simple aggregrate (MAX) on one metric for 1 host, every 1 minute for 12 hours

###### Query language
```
TS.MRANGE 1451628982646 1451672182646 AGGREGATION MAX 60000 FILTER measurement=cpu fieldname=usage_user hostname=host_9
```

###### Sample Responses:
- [InfluxDB](./responses/influx_single-groupby-1-1-12.json)
- [TimescaleDB](./responses/timescaledb_single-groupby-1-1-12.json)
- [RedistimeSeries](./responses/redistimeseries_single-groupby-1-1-12.json)

##### q3) single-groupby-1-8-1 ( *client side work for RedisTimeSeries )
Simple aggregrate (MAX) on one metric for 8 hosts, every 1 minute for 1 hour

###### Query language
```
TS.MRANGE 1451679382646 1451682982646 AGGREGATION MAX 60000 FILTER measurement=cpu fieldname=usage_user hostname=(host_9,host_3,host_5,host_1,host_7,host_2,host_8,host_4)
```
######  Client side work description: 
Max Reduction over time on multiple time-series with the same metric
###### Code: [github.com/timescale/tsbs/query.GroupByTimeAndMax](./../query/redistimeseries_functors.go#L42)

######  Sample Responses:
- [InfluxDB](./responses/influx_single-groupby-1-8-1.json)
- [TimescaleDB](./responses/timescaledb_single-groupby-1-8-1.json)
- [RedistimeSeries](./responses/redistimeseries_single-groupby-1-8-1.json)



##### q4) single-groupby-5-1-1 ( *client side work for RedisTimeSeries )
 Simple aggregrate (MAX) on 5 metrics for 1 host, every 5 mins for 1 hour

###### Query language
```
TS.MRANGE 1451679382646 1451682982646 AGGREGATION MAX 60000 FILTER measurement=cpu fieldname=(usage_user,usage_system,usage_idle,usage_nice,usage_iowait) hostname=host_9
```
######  Client side work description: 
Aggregation over time on multiple time-series datapoints with the same timestamp

###### Code: [github.com/timescale/tsbs/query.SingleGroupByTime](./../query/redistimeseries_functors.go#L33)

######  Sample Responses:
- [InfluxDB](./responses/influx_single-groupby-5-1-1.json)
- [TimescaleDB](./responses/timescaledb_single-groupby-5-1-1.json)
- [RedistimeSeries](./responses/redistimeseries_single-groupby-5-1-1.json)



##### q5) single-groupby-5-1-12 ( *client side work for RedisTimeSeries )
Simple aggregrate (MAX) on 5 metrics for 1 host, every 5 mins for 12 hours


###### Query language
```
TS.MRANGE 1451628982646 1451672182646 AGGREGATION MAX 60000 FILTER measurement=cpu fieldname=(usage_user,usage_system,usage_idle,usage_nice,usage_iowait) hostname=host_9
```
######  Client side work description: 
Aggregation over time on multiple time-series datapoints with the same timestamp

###### Code: [github.com/timescale/tsbs/query.SingleGroupByTime](./../query/redistimeseries_functors.go#L33)

######  Sample Responses:
- [InfluxDB](./responses/influx_single-groupby-5-1-12.json)
- [TimescaleDB](./responses/timescaledb_single-groupby-5-1-12.json)
- [RedistimeSeries](./responses/redistimeseries_single-groupby-5-1-12.json)



##### q6) single-groupby-5-8-1 ( *client side work for RedisTimeSeries )
Simple aggregrate (MAX) on 5 metrics for 8 hosts, every 5 mins for 1 hour



###### Query language
```
TS.MRANGE 1451679382646 1451682982646 AGGREGATION MAX 60000 FILTER measurement=cpu fieldname=(usage_user,usage_system,usage_idle,usage_nice,usage_iowait) hostname=(host_9,host_3,host_5,host_1,host_7,host_2,host_8,host_4)
```
######  Client side work description: 
Max Reduction over time on multiple time-series with the same metric, and aggregation over time on multiple time-series datapoints with the same timestamp but different metrics

###### Code: [github.com/timescale/tsbs/query.GroupByTimeAndTagMax](./../query/redistimeseries_functors.go#L51)

######  Sample Responses:
- [InfluxDB](./responses/influx_single-groupby-5-8-1.json)
- [TimescaleDB](./responses/timescaledb_single-groupby-5-8-1.json)
- [RedistimeSeries](./responses/redistimeseries_single-groupby-5-8-1.json)



#### Simple Aggregations 

##### q7) cpu-max-all-1 ( *client side work for RedisTimeSeries )
Aggregate across all CPU metrics per hour over 1 hour for a single host



###### Query language
```
TS.MRANGE 1451614582646 1451643382646 AGGREGATION MAX 3600000 FILTER measurement=cpu hostname=host_9
```
######  Client side work description: 
Aggregation over time on multiple time-series datapoints with the same timestamp

###### Code: [github.com/timescale/tsbs/query.SingleGroupByTime](./../query/redistimeseries_functors.go#L33)

######  Sample Responses:
- [InfluxDB](./responses/influx_cpu-max-all-1.json) *this query is being generated with the wrong parameters for InfluxDB ( will create issue upwards )*
- [TimescaleDB](./responses/timescaledb_cpu-max-all-1.json)
- [RedistimeSeries](./responses/redistimeseries_cpu-max-all-1.json)



##### q8) cpu-max-all-8 ( *client side work for RedisTimeSeries )
Aggregate across all CPU metrics per hour over 1 hour for eight hosts


###### Query language
```
TS.MRANGE 1451614582646 1451643382646 AGGREGATION MAX 3600000 FILTER measurement=cpu hostname=(host_9,host_3,host_5,host_1,host_7,host_2,host_8,host_4)
```
######  Client side work description: 
Max Reduction over time on multiple time-series with the same metric, and aggregation over time on multiple time-series datapoints with the same timestamp but different metrics

###### Code: [github.com/timescale/tsbs/query.GroupByTimeAndTagMax](./../query/redistimeseries_functors.go#L51)

######  Sample Responses:
- [InfluxDB](./responses/influx_cpu-max-all-8.json)
- [TimescaleDB](./responses/timescaledb_cpu-max-all-8.json)
- [RedistimeSeries](./responses/redistimeseries_cpu-max-all-8.json)



#### Double Rollups 
##### q9) double-groupby-1 ( *client side work for RedisTimeSeries )
Aggregate on across both time and host, giving the average of 1 CPU metric per host per hour for 24 hours


###### Query language
```
TS.MRANGE 1451628982646 1451672182646 AGGREGATION AVG 3600000 FILTER measurement=cpu fieldname=usage_user
```
######  Client side work description: 
Group on multiple time-series with the same tag ( hostname ), and aggregation over time on multiple time-series datapoints with the same timestamp but same metric

###### Code: [github.com/timescale/tsbs/query.GroupByTimeAndTagHostname](./../query/redistimeseries_functors.go#L76)

######  Sample Responses:
- [InfluxDB](./responses/influx_double-groupby-1.json)
- [TimescaleDB](./responses/timescaledb_double-groupby-1.json)
- [RedistimeSeries](./responses/redistimeseries_double-groupby-1.json)



##### q10) double-groupby-5 ( *client side work for RedisTimeSeries )
 Aggregate on across both time and host, giving the average of 5 CPU metrics per host per hour for 24 hours

###### Query language
```
TS.MRANGE 1451628982646 1451672182646 AGGREGATION AVG 3600000 FILTER measurement=cpu fieldname=(usage_user,usage_system,usage_idle,usage_nice,usage_iowait)
```
######  Client side work description: 
Group on multiple time-series with the same tag ( hostname ), and aggregation over time on multiple time-series datapoints with the same timestamp but different metrics

###### Code: [github.com/timescale/tsbs/query.GroupByTimeAndTagHostname](./../query/redistimeseries_functors.go#L76)

######  Sample Responses:
- [InfluxDB](./responses/influx_double-groupby-5.json)
- [TimescaleDB](./responses/timescaledb_double-groupby-5.json)
- [RedistimeSeries](./responses/redistimeseries_double-groupby-5.json)



##### q11) double-groupby-all ( *client side work for RedisTimeSeries )
 Aggregate on across both time and host, giving the average of all (10) CPU metrics per host per hour for 24 hours

###### Query language
```
TS.MRANGE 1451628982646 1451672182646 AGGREGATION AVG 3600000 FILTER measurement=cpu
```
######  Client side work description: 
Group on multiple time-series with the same tag ( hostname ), and aggregation over time on multiple time-series datapoints with the same timestamp but different metrics

###### Code: [github.com/timescale/tsbs/query.GroupByTimeAndTagHostname](./../query/redistimeseries_functors.go#L76)

######  Sample Responses:
- [InfluxDB](./responses/influx_double-groupby-all.json)
- [TimescaleDB](./responses/timescaledb_double-groupby-all.json)
- [RedistimeSeries](./responses/redistimeseries_double-groupby-all.json)


#### Thresholds

##### q12) high-cpu-all ( *client side work for RedisTimeSeries )
All the readings where one metric is above a threshold across all hosts

###### Query language
```
TS.MRANGE 1451628982646 1451672182646 FILTER measurement=cpu
```
######  Client side work description: 
Group on multiple time-series with the same tag ( hostname ), and aggregation over time on multiple time-series datapoints with the same timestamp but different metrics, if a specific time-series with one of the metrics (usage_user) is above a threshold

###### Code: [github.com/timescale/tsbs/query.HighCpu](./../query/redistimeseries_functors.go#L98)

######  Sample Responses:
- [InfluxDB](./responses/influx_high-cpu-all.json)
- [TimescaleDB](./responses/timescaledb_high-cpu-all.json)
- [RedistimeSeries](./responses/redistimeseries_high-cpu-all.json)


##### q13) high-cpu-1 ( *client side work for RedisTimeSeries )
 All the readings where one metric is above a threshold for a particular host

###### Query language
```
TS.MRANGE 1451649250138 1451692450138 FILTER measurement=cpu hostname=host_5
```
######  Client side work description: 
Group on multiple time-series with the same tag ( hostname ), and aggregation over time on multiple time-series datapoints with the same timestamp but different metrics, if a specific time-series with one of the metrics (usage_user) is above a threshold

###### Code: [github.com/timescale/tsbs/query.HighCpu](./../query/redistimeseries_functors.go#L98)

######  Sample Responses:
- [InfluxDB](./responses/influx_high-cpu-1.json)
- [TimescaleDB](./responses/timescaledb_high-cpu-1.json)
- [RedistimeSeries](./responses/redistimeseries_high-cpu-1.json)


#### Complex queries

##### q14) lastpoint ( *client side work for RedisTimeSeries )[REQUIRES MREVRANGE]
The last reading for each host

###### Query language
```
TS.MREVRANGE + - COUNT 1 FILTER measurement=cpu hostname!=
```
######  Client side work description: 

Group on multiple time-series with the same tag ( hostname ), and aggregation over time on multiple time-series datapoints with the same timestamp but different metrics

###### Code: [github.com/timescale/tsbs/query.GroupByTimeAndTagHostname](./../query/redistimeseries_functors.go#L76)

######  Sample Responses:
- [InfluxDB](./responses/influx_lastpoint.json)
- [TimescaleDB](./responses/timescaledb_lastpoint.json)
- [RedistimeSeries](./responses/redistimeseries_lastpoint.json)


##### q15) groupby-orderby-limit ( *client side work for RedisTimeSeries )[REQUIRES MREVRANGE] [STILL WIP]

The last 5 aggregate readings (across time) before a randomly chosen endpoint

###### Query language
```
TS.MREVRANGE 1451682982646 - COUNT 5 AGGREGATION MAX 60000 FILTER measurement=cpu fieldname=usage_user
```
######  Client side work description: 
WIP

###### Code: WIP

######  Sample Responses:
- [InfluxDB](./responses/influx_groupby-orderby-limit.json)
- [TimescaleDB](./responses/timescaledb_groupby-orderby-limit.json)
- [RedistimeSeries](./responses/redistimeseries_groupby-orderby-limit.json)



---

## `tsbs_load_redistimeseries` Additional Flags

### Database related

#### `-host` (type: `string`, default: `localhost:6379`)

The host:port for Redis connection.

#### `-connections` (type: `int`, default: `10`)

The number of connections per worker.

#### `-pipeline` (type: `int`, default: `50`)

The pipeline's size. Read the full documentation [here](https://redis.io/topics/pipelining).

### Miscellaneous

#### `-single-queue` (type: `boolean`, default: `true`)

Whether to use a single queue.

#### `-check-chunks` (type: `int`, default: `0`)

Whether to perform post ingestion chunck count.