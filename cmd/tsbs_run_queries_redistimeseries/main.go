// tsbs_run_queries_siridb speed tests SiriDB using requests from stdin or file
//

// This program has no knowledge of the internals of the endpoint.
package main

import (
	"fmt"
	redistimeseries "github.com/RedisTimeSeries/redistimeseries-go"
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
	host        string
	showExplain bool
	//	scale        uint64
)

// Global vars:
var (
	runner *query.BenchmarkRunner
)

var (
	redisConnector *redistimeseries.Client
)

// Parse args:
func init() {
	var config query.BenchmarkRunnerConfig
	config.AddToFlagSet(pflag.CommandLine)

	pflag.StringVar(&host, "host", "localhost:6379", "Redis host address and port")
	//flag.Uint64Var(&scale, "scale", 8, "Scaling variable (Must be the equal to the scalevar used for data generation).")

	pflag.Parse()

	err := utils.SetupConfigFile()

	if err != nil {
		panic(fmt.Errorf("fatal error config file: %s", err))
	}

	if err := viper.Unmarshal(&config); err != nil {
		panic(fmt.Errorf("unable to decode config: %s", err))
	}
	runner = query.NewBenchmarkRunner(config)

	redisConnector = redistimeseries.NewClient(
		host, runner.DatabaseName(), nil)
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
	opts *queryExecutorOptions
}

func newProcessor() query.Processor { return &processor{} }

func (p *processor) Init(numWorker int) {
	p.opts = &queryExecutorOptions{
		showExplain:   showExplain,
		debug:         runner.DebugLevel() > 0,
		printResponse: runner.DoPrintResponses(),
	}
}

func (p *processor) ProcessQuery(q query.Query, isWarm bool) ([]*query.Stat, error) {

	// No need to run again for EXPLAIN
	if isWarm && p.opts.showExplain {
		return nil, nil
	}
	tq := q.(*query.RedisTimeSeries)

	qry := string(tq.RedisQuery)

	conn := redisConnector.Pool.Get()
	t := strings.Split(qry, " ")
	commandArgs := make([]interface{}, len(t)-1)
	for i := 1; i < len(t); i++ {
		commandArgs[i-1] = t[i]
	}
	start := time.Now()
	res, err := conn.Do(t[0], commandArgs...)
	if err != nil {
		log.Fatalf("Command failed:%v %v\n", res, err)
	}

	if p.opts.debug {
		fmt.Println(qry)
	}

	if p.opts.printResponse {
		fmt.Println("\n", res)
	}

	took := float64(time.Since(start).Nanoseconds()) / 1e6
	stat := query.GetStat()
	stat.Init(q.HumanLabelName(), took)

	return []*query.Stat{stat}, err
}
