package redistimeseries

import (
	"fmt"
	"time"

	"github.com/timescale/tsbs/cmd/tsbs_generate_queries/uses/devops"
	"github.com/timescale/tsbs/query"
)

// TODO: Remove the need for this by continuing to bubble up errors
func panicIfErr(err error) {
	if err != nil {
		panic(err.Error())
	}
}

const (
	oneMinuteMillis = 60 * 1000
	fiveMinuteMillis = 5 * oneMinuteMillis
	oneHourMillis   = oneMinuteMillis * 60
)

// Devops produces RedisTimeSeries-specific queries for all the devops query types.
type Devops struct {
	*BaseGenerator
	*devops.Core
}

// GenerateEmptyQuery returns an empty query.RedisTimeSeries
func (d *Devops) GenerateEmptyQuery() query.Query {
	return query.NewRedisTimeSeries()
}

// GroupByTime fetches the MAX for numMetrics metrics under 'cpu', per minute for nhosts hosts,
// every 5 mins for 1 hour
func (d *Devops) GroupByTime(qi query.Query, nHosts, numMetrics int, timeRange time.Duration) {
	interval := d.Interval.MustRandWindow(timeRange)

	redisQuery := fmt.Sprintf(`TS.MRANGE %d %d AGGREGATION max %d FILTER`,
		interval.StartUnixMillis(),
		interval.EndUnixMillis(),
		fiveMinuteMillis)

	metrics, err := devops.GetCPUMetricsSlice(numMetrics)
	panicIfErr(err)
	// we only need to filter if we we dont want all of them
	if numMetrics != devops.GetCPUMetricsLen() {
	redisQuery += " fieldname="
	if numMetrics > 1 {
		redisQuery += "("
	}

		for idx, value := range metrics {
			redisQuery += value
			if idx != (numMetrics - 1) {
				redisQuery += ","
			}
		}
		if numMetrics > 1 {
			redisQuery += ")"
		}
	}

	hostnames, err := d.GetRandomHosts(nHosts)
	panicIfErr(err)

	// add specific fieldname if needed.
	redisQuery += " hostname="
	if nHosts > 1 {
		redisQuery += "("
	}
	for idx, value := range hostnames {
		redisQuery += value
		if idx != (nHosts - 1) {
			redisQuery += ","
		}
	}
	if nHosts > 1 {
		redisQuery += ")"
	}

	humanLabel := fmt.Sprintf("RedisTimeSeries %d cpu metric(s), random %4d hosts, random %s by 1m", numMetrics, nHosts, timeRange)
	humanDesc := fmt.Sprintf("%s: %s", humanLabel, interval.StartString())
	d.fillInQuery(qi, humanLabel, humanDesc, redisQuery)
}

// GroupByTimeAndPrimaryTag selects the AVG of numMetrics metrics under 'cpu' per device per hour for a day
func (d *Devops) GroupByTimeAndPrimaryTag(qi query.Query, numMetrics int) {
	interval := d.Interval.MustRandWindow(devops.DoubleGroupByDuration)

	redisQuery := fmt.Sprintf(`TS.MRANGE %d %d AGGREGATION avg %d FILTER measurement=cpu`,
		interval.StartUnixMillis(),
		interval.EndUnixMillis(),
		oneHourMillis)

	metrics, err := devops.GetCPUMetricsSlice(numMetrics)
	panicIfErr(err)

	// add specific fieldname if needed.
	if numMetrics != devops.GetCPUMetricsLen() {
		redisQuery += " fieldname=("
		for idx, value := range metrics {
			redisQuery += value
			if idx != (numMetrics - 1) {
				redisQuery += ","
			}
		}
		redisQuery += ")"
	}

	humanLabel := devops.GetDoubleGroupByLabel("RedisTimeSeries", numMetrics)
	humanDesc := fmt.Sprintf("%s: %s", humanLabel, interval.StartString())
	d.fillInQuery(qi, humanLabel, humanDesc, redisQuery)
}

// MaxAllCPU fetches the aggregate across all CPU metrics per hour over 1 hour for a single host.
// Currently only one host is supported
func (d *Devops) MaxAllCPU(qi query.Query, nHosts int) {
	interval := d.Interval.MustRandWindow(devops.MaxAllDuration)
	hostnames, err := d.GetRandomHosts(nHosts)
	panicIfErr(err)

	redisQuery := fmt.Sprintf(`TS.MRANGE %d %d AGGREGATION max %d FILTER measurement=cpu hostname=`,
		interval.StartUnixMillis(),
		interval.EndUnixMillis(),
		oneHourMillis)

	if nHosts > 1 {
		redisQuery += "("
	}
	for idx, value := range hostnames {
		redisQuery += value
		if idx != (nHosts - 1) {
			redisQuery += ","
		}
	}
	if nHosts > 1 {
		redisQuery += ")"
	}

	humanLabel := devops.GetMaxAllLabel("RedisTimeSeries", nHosts)
	humanDesc := fmt.Sprintf("%s: %s", humanLabel, interval.StartString())
	d.fillInQuery(qi, humanLabel, humanDesc, redisQuery)
}
