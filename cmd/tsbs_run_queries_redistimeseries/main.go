// tsbs_run_queries_redistimeseries speed tests RedisTimeSeries using requests from stdin or file
//

// This program has no knowledge of the internals of the endpoint.
package tsbs_run_queries_redistimeseries

import (
	"fmt"
	//"github.com/gomodule/redigo/redis"
	radix "github.com/mediocregopher/radix/v3"

	_ "github.com/lib/pq"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/timescale/tsbs/internal/utils"
	"github.com/timescale/tsbs/query"
	"log"
	"strings"
	"time"
)

// Program option vars:
var (
	host                  string
	showExplain           bool
	applyResponseFunctors bool
)

// Global vars:
var (
	runner                            *query.BenchmarkRunner
	cmdMrange                         = []byte("TS.MRANGE")
	cmdMRevRange                      = []byte("TS.MREVRANGE")
	cmdQueryIndex                     = []byte("TS.QUERYINDEX")
	reflect_SingleGroupByTime         = query.GetFunctionName(query.SingleGroupByTime)
	reflect_GroupByTimeAndMax         = query.GetFunctionName(query.GroupByTimeAndMax)
	reflect_GroupByTimeAndTagMax      = query.GetFunctionName(query.GroupByTimeAndTagMax)
	reflect_GroupByTimeAndTagHostname = query.GetFunctionName(query.GroupByTimeAndTagHostname)
	reflect_HighCpu                   = query.GetFunctionName(query.HighCpu)
)

// Parse args:
func init() {
	var config query.BenchmarkRunnerConfig
	config.AddToFlagSet(pflag.CommandLine)

	pflag.StringVar(&host, "host", "localhost:6379", "Redis host address and port")
	pflag.BoolVar(&applyResponseFunctors, "apply-response-functors", false, "Apply response functors. ( False by default )")

	pflag.Parse()

	err := utils.SetupConfigFile()

	if err != nil {
		panic(fmt.Errorf("fatal error config file: %s", err))
	}

	if err := viper.Unmarshal(&config); err != nil {
		panic(fmt.Errorf("unable to decode config: %s", err))
	}
	runner = query.NewBenchmarkRunner(config)
}

func main() {
	runner.Run(&query.RedisTimeSeriesPool, newProcessor)
}

type queryExecutorOptions struct {
	showExplain   bool
	debug         bool
	printResponse bool
}

type processor struct {
	opts        *queryExecutorOptions
	redisClient radix.Client
}

func newProcessor() query.Processor {
	conn, _ := radix.Dial("tcp", host)
	client := radix.Client(conn)
	return &processor{redisClient: client,}
}

func (p *processor) Init(numWorker int) {
	p.opts = &queryExecutorOptions{
		showExplain:   showExplain,
		debug:         runner.DebugLevel() > 0,
		printResponse: runner.DoPrintResponses(),
	}
}

func (p *processor) ProcessQuery(q query.Query, isWarm bool) (queryStats []*query.Stat, err error) {

	// No need to run again for EXPLAIN
	if isWarm && p.opts.showExplain {
		return nil, nil
	}
	tq := q.(*query.RedisTimeSeries)
	var parsedResponses = make([]interface{}, 0, 0)

	var cmds = make([][]string, 0, 0)
	for _, qry := range tq.RedisQueries {
		stringquery := make([]string, 0, 0)
		for _, arg := range qry {
			stringquery = append(stringquery, string(arg))
		}
		cmds = append(cmds, stringquery)

	}

	start := time.Now()
	for idx, commandArgs := range cmds {
		//var result interface{}
		if p.redisClient == nil {
			conn, _ := radix.Dial("tcp", host)
			client := radix.Client(conn)
			p.redisClient = client

		}
		if p.opts.debug {
			fmt.Println(fmt.Sprintf("Issuing command (%s %s)", string(tq.CommandNames[idx]), strings.Join(ByteArrayToStringArray(tq.RedisQueries[idx]), " ")))
		}
		err = p.redisClient.Do(radix.Cmd(nil, string(tq.CommandNames[idx]), commandArgs...))
		//res, err := p.redisClient.Do(string(tq.CommandNames[idx]), commandArgs...)
		if err != nil {
			log.Fatalf("Command (%s %s) failed with error: %v\n", string(tq.CommandNames[idx]), strings.Join(ByteArrayToStringArray(tq.RedisQueries[idx]), " "), err)
		}
		if err != nil {
			return nil, err
		}
		//if bytes.Compare(tq.CommandNames[idx], cmdMrange) == 0 || bytes.Compare(tq.CommandNames[idx], cmdMRevRange) == 0 {
		//
		//	if err != nil {
		//		return nil, err
		//	}
		//	if tq.ApplyFunctor && applyResponseFunctors {
		//		_, err = p.applyResponseFunctions(tq)
		//		if err != nil {
		//			return nil, err
		//		}
		//	} else {
		//		result, err = redistimeseries.ParseRanges(res)
		//		if err != nil {
		//			return nil, err
		//		}
		//	}
		//
		//} else if bytes.Compare(tq.CommandNames[idx], cmdQueryIndex) == 0 {
		//	var parsedRes = make([]redistimeseries.Range, 0, 0)
		//	parsedResponses = append(parsedResponses, parsedRes)
		//}
		//parsedResponses = append(parsedResponses, result)
	}
	took := float64(time.Since(start).Nanoseconds()) / 1e6
	if p.opts.printResponse {
		prettyPrintResponseRange(parsedResponses, tq)
	}
	stat := query.GetStat()
	stat.Init(q.HumanLabelName(), took)
	queryStats = []*query.Stat{stat}

	return queryStats, err
}
