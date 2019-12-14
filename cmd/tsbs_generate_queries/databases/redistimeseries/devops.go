package redistimeseries

import (
	"fmt"
	"reflect"
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
	oneMinuteMillis  = 60 * 1000
	fiveMinuteMillis = 5 * oneMinuteMillis
	oneHourMillis    = oneMinuteMillis * 60
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
	redisQuery := [][]byte{
		//[]byte("TS.MRANGE"), Just to help understanding
		[]byte(fmt.Sprintf("%d", interval.StartUnixMillis())),
		[]byte(fmt.Sprintf("%d", interval.EndUnixMillis())),
		[]byte("AGGREGATION"),
		[]byte("MAX"),
		[]byte(fmt.Sprintf("%d", oneMinuteMillis)),
		[]byte("FILTER"),
		[]byte("measurement=cpu"),
	}

	metrics, err := devops.GetCPUMetricsSlice(numMetrics)
	panicIfErr(err)

	// we only need to filter if we we dont want all of them
	if numMetrics != devops.GetCPUMetricsLen() {
		redisArg := "fieldname="
		if numMetrics > 1 {
			redisArg += "("
		}
		for idx, value := range metrics {
			redisArg += value
			if idx != (numMetrics - 1) {
				redisArg += ","
			}
		}
		if numMetrics > 1 {
			redisArg += ")"
		}
		redisQuery = append(redisQuery, []byte(redisArg ))
	}

	hostnames, err := d.GetRandomHosts(nHosts)
	panicIfErr(err)

	// add specific fieldname if needed.
	redisArg := "hostname="
	if nHosts > 1 {
		redisArg += "("
	}
	for idx, value := range hostnames {
		redisArg += value
		if idx != (nHosts - 1) {
			redisArg += ","
		}
	}
	if nHosts > 1 {
		redisArg += ")"
	}
	redisQuery = append(redisQuery, []byte(redisArg ))

	humanLabel := devops.GetSingleGroupByLabel("RedisTimeSeries",numMetrics, nHosts, string(timeRange))
	humanDesc := fmt.Sprintf("%s: %s", humanLabel, interval.StartString())
	d.fillInQueryStrings(qi, humanLabel, humanDesc)
	d.AddQuery(qi, redisQuery, []byte("TS.MRANGE"))
	if numMetrics > 1 && nHosts == 1 {
		functorName := reflect.ValueOf(query.SingleGroupByTime).String()
		d.SetApplyFunctor(qi, true, functorName )
	}
	if nHosts > 1 && numMetrics == 1 {
		functorName := reflect.ValueOf(query.GroupByTimeAndMax).String()
		d.SetApplyFunctor(qi, true, functorName )
	}
	if nHosts > 1 && numMetrics > 1 {
		functorName := reflect.ValueOf(query.GroupByTimeAndTag).String()
		d.SetApplyFunctor(qi, true, functorName )
	}
}

// GroupByTimeAndPrimaryTag selects the AVG of numMetrics metrics under 'cpu' per device per hour for a day
func (d *Devops) GroupByTimeAndPrimaryTag(qi query.Query, numMetrics int) {
	interval := d.Interval.MustRandWindow(devops.DoubleGroupByDuration)
	redisQuery := [][]byte{
		//[]byte("TS.MRANGE"), Just to help understanding
		[]byte(fmt.Sprintf("%d", interval.StartUnixMillis())),
		[]byte(fmt.Sprintf("%d", interval.EndUnixMillis())),
		[]byte("AGGREGATION"),
		[]byte("AVG"),
		[]byte(fmt.Sprintf("%d", oneHourMillis)),
		[]byte("FILTER"),
		[]byte("measurement=cpu"),
	}

	metrics, err := devops.GetCPUMetricsSlice(numMetrics)
	panicIfErr(err)

	// add specific fieldname if needed.
	if numMetrics != devops.GetCPUMetricsLen() {
		redisArg := "fieldname="
		if numMetrics > 1 {
			redisArg += "("
		}
		for idx, value := range metrics {
			redisArg += value
			if idx != (numMetrics - 1) {
				redisArg += ","
			}
		}
		if numMetrics > 1 {
			redisArg += ")"
		}
		redisQuery = append(redisQuery, []byte(redisArg ))
	}

	humanLabel := devops.GetDoubleGroupByLabel("RedisTimeSeries", numMetrics)
	humanDesc := fmt.Sprintf("%s: %s", humanLabel, interval.StartString())
	d.fillInQueryStrings(qi, humanLabel, humanDesc)
	d.AddQuery(qi, redisQuery, []byte("TS.MRANGE"))
}

// MaxAllCPU fetches the aggregate across all CPU metrics per hour over 1 hour for a single host.
// Currently only one host is supported
func (d *Devops) MaxAllCPU(qi query.Query, nHosts int) {
	interval := d.Interval.MustRandWindow(devops.MaxAllDuration)
	hostnames, err := d.GetRandomHosts(nHosts)
	panicIfErr(err)
	redisQuery := [][]byte{
		//[]byte("TS.MRANGE"), Just to help understanding
		[]byte(fmt.Sprintf("%d", interval.StartUnixMillis())),
		[]byte(fmt.Sprintf("%d", interval.EndUnixMillis())),
		[]byte("AGGREGATION"),
		[]byte("MAX"),
		[]byte(fmt.Sprintf("%d", oneHourMillis)),
		[]byte("FILTER"),
		[]byte("measurement=cpu"),
	}

	redisArg := "hostname="
	if nHosts > 1 {
		redisArg += "("
	}
	for idx, value := range hostnames {
		redisArg += value
		if idx != (nHosts - 1) {
			redisArg += ","
		}
	}
	if nHosts > 1 {
		redisArg += ")"
	}
	redisQuery = append(redisQuery, []byte(redisArg ))

	humanLabel := devops.GetMaxAllLabel("RedisTimeSeries", nHosts)
	humanDesc := fmt.Sprintf("%s: %s", humanLabel, interval.StartString())
	d.fillInQueryStrings(qi, humanLabel, humanDesc)
	d.AddQuery(qi, redisQuery, []byte("TS.MRANGE"))
	if nHosts == 1 {
		functorName := reflect.ValueOf(query.SingleGroupByTime).String()
		d.SetApplyFunctor(qi, true, functorName )
		}
}

// LastPointPerHost finds the last row for every host in the dataset
func (d *Devops) LastPointPerHost(qi query.Query) {
	redisQuery := [][]byte{
		//[]byte("TS.QUERYINDEX"), Just to help understanding
		[]byte("measurement=cpu"),
		[]byte("hostname!="),
	}
	//redisQuery := fmt.Sprintf(`TS.QUERYINDEX measurement=cpu hostname!=`)
	humanLabel := "RedisTimeSeries last row per host"
	humanDesc := fmt.Sprintf("%s", humanLabel)
	d.fillInQueryStrings(qi, humanLabel, humanDesc)
	d.AddQuery(qi, redisQuery, []byte("TS.QUERYINDEX"))
}

func (d *Devops) HighCPUForHosts(qi query.Query, nHosts int) {
	hostnames, err := d.GetRandomHosts(nHosts)
	interval := d.Interval.MustRandWindow(devops.HighCPUDuration)
	redisQuery := [][]byte{
		//[]byte("TS.MRANGE"), Just to help understanding
		[]byte(fmt.Sprintf("%d", interval.StartUnixMillis())),
		[]byte(fmt.Sprintf("%d", interval.EndUnixMillis())),
		[]byte("FILTER"),
		[]byte("measurement=cpu"),
	}

	if nHosts > 0 {
		redisArg := "hostname="
		if nHosts > 1 {
			redisArg += "("
		}
		for idx, value := range hostnames {
			redisArg += value
			if idx != (nHosts - 1) {
				redisArg += ","
			}
		}
		if nHosts > 1 {
			redisArg += ")"
		}
		redisQuery = append(redisQuery, []byte(redisArg ))
	}
	humanLabel, err := devops.GetHighCPULabel("RedisTimeSeries", nHosts)
	panicIfErr(err)
	humanDesc := fmt.Sprintf("%s: %s", humanLabel, interval.StartString())
	d.fillInQueryStrings(qi, humanLabel, humanDesc)
	d.AddQuery(qi, redisQuery, []byte("TS.MRANGE"))
}

// GroupByOrderByLimit populates a query.Query that has a time WHERE clause, that groups by a truncated date, orders by that date, and takes a limit:
func (d *Devops) GroupByOrderByLimit(qi query.Query) {

	interval := d.Interval.MustRandWindow(time.Hour)
	redisQuery := [][]byte{
		//[]byte("TS.MREVRANGE"), Just to help understanding
		[]byte(fmt.Sprintf("%d", interval.EndUnixMillis())),
		[]byte("-"),
		[]byte("AGGREGATION"),
		[]byte("MAX"),
		[]byte(fmt.Sprintf("%d", oneMinuteMillis)),
		[]byte("FILTER"),
		[]byte("measurement=cpu"),
		[]byte("LIMIT"),
		[]byte("5"),
	}

	humanLabel := devops.GetGroupByOrderByLimitLabel("RedisTimeSeries")
	humanDesc := fmt.Sprintf("%s: %s", humanLabel, interval.EndString())

	d.fillInQueryStrings(qi, humanLabel, humanDesc)
	d.AddQuery(qi, redisQuery, []byte("TS.MREVRANGE"))

}
