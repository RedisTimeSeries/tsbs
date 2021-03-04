package main

import (
	"bufio"
	"crypto/md5"
	"encoding/binary"
	"fmt"
	"github.com/mediocregopher/radix/v3"
	"io"
	"log"
	"strconv"
	"strings"
	"sync"

	"github.com/blagojts/viper"
	"github.com/spf13/pflag"
	"github.com/timescale/tsbs/internal/utils"
	"github.com/timescale/tsbs/pkg/data/usecases/common"
	"github.com/timescale/tsbs/pkg/targets/constants"
	"github.com/timescale/tsbs/pkg/targets/initializers"

	"github.com/timescale/tsbs/load"
	"github.com/timescale/tsbs/pkg/data"
	"github.com/timescale/tsbs/pkg/targets"
)

// Program option vars:
var (
	host               string
	connections        uint64
	pipeline           uint64
	checkChunks        uint64
	singleQueue        bool
	dataModel          string
	compressionEnabled bool
	clusterMode        bool
)

// Global vars
var (
	loader load.BenchmarkRunner
	config load.BenchmarkRunnerConfig
	target targets.ImplementedTarget
)

// allows for testing
var fatal = log.Fatal
var md5h = md5.New()

// Parse args:
func init() {
	target = initializers.GetTarget(constants.FormatRedisTimeSeries)
	config = load.BenchmarkRunnerConfig{}
	config.AddToFlagSet(pflag.CommandLine)
	target.TargetSpecificFlags("", pflag.CommandLine)
	pflag.Parse()

	err := utils.SetupConfigFile()

	if err != nil {
		panic(fmt.Errorf("fatal error config file: %s", err))
	}

	if err := viper.Unmarshal(&config); err != nil {
		panic(fmt.Errorf("unable to decode config: %s", err))
	}
	host = viper.GetString("host")
	connections = viper.GetUint64("connections")
	pipeline = viper.GetUint64("pipeline")
	dataModel = "redistimeseries"
	compressionEnabled = true
	clusterMode = viper.GetBool("cluster")

	loader = load.GetBenchmarkRunner(config)

}

type benchmark struct {
	dbc *dbCreator
}

func (b *benchmark) GetDataSource() targets.DataSource {
	log.Printf("creating DS from %s", config.FileName)
	return &fileDataSource{scanner: bufio.NewScanner(load.GetBufferedReader(config.FileName))}
}

type RedisIndexer struct {
	partitions uint
}

func (i *RedisIndexer) GetIndex(p data.LoadedPoint) uint {
	row := p.Data.(string)
	key := strings.Split(row, " ")[1]
	start := strings.Index(key, "{")
	end := strings.Index(key, "}")
	_, _ = io.WriteString(md5h, key[start+1:end])
	hash := binary.LittleEndian.Uint32(md5h.Sum(nil))
	md5h.Reset()
	return uint(hash) % i.partitions
}

type fileDataSource struct {
	scanner *bufio.Scanner
}

func (d *fileDataSource) NextItem() data.LoadedPoint {
	ok := d.scanner.Scan()
	if !ok && d.scanner.Err() == nil { // nothing scanned & no error = EOF
		return data.LoadedPoint{}
	} else if !ok {
		fatal("scan error: %v", d.scanner.Err())
		return data.LoadedPoint{}
	}
	return data.NewLoadedPoint(d.scanner.Text())
}

func (d *fileDataSource) Headers() *common.GeneratedDataHeaders { return nil }

func (b *benchmark) GetBatchFactory() targets.BatchFactory {
	return &factory{}
}

func (b *benchmark) GetPointIndexer(maxPartitions uint) targets.PointIndexer {
	return &RedisIndexer{partitions: maxPartitions}
}

func (b *benchmark) GetProcessor() targets.Processor {
	return &processor{b.dbc, nil, nil, nil}
}

func (b *benchmark) GetDBCreator() targets.DBCreator {
	return b.dbc
}

type processor struct {
	dbc     *dbCreator
	rows    []chan string
	metrics chan uint64
	wg      *sync.WaitGroup
}

func connectionProcessor(wg *sync.WaitGroup, rows chan string, metrics chan uint64, conn radix.Client) {
	curPipe := uint64(0)
	cmds := make([]radix.CmdAction, 0, 0)
	currMetricCount := 0

	for row := range rows {
		cmd, tscreate,metricCount := buildCommand(row, compressionEnabled == false)
		currMetricCount += metricCount
		if tscreate {
			err := conn.Do(cmd)
			if err != nil {
				log.Fatalf("TS.CREATE failed with %v", err)
			}
		} else {
			cmds = append(cmds, cmd)
			curPipe++

			if curPipe == pipeline {
				err := conn.Do(radix.Pipeline(cmds...))
				if err != nil {
					log.Fatalf("Flush failed with %v", err)
				}
				metrics <- uint64(currMetricCount)
				currMetricCount = 0
				cmds = make([]radix.CmdAction, 0, 0)
				curPipe = 0
			}
		}
	}
	if curPipe > 0 {
		err := conn.Do(radix.Pipeline(cmds...))
		if err != nil {
			log.Fatalf("Flush failed with %v", err)
		}
		metrics <- uint64(currMetricCount)
		cmds = make([]radix.CmdAction, 0, 0)
		curPipe = 0
		currMetricCount = 0
	}
	wg.Done()
}

func (p *processor) Init(_ int, _ bool, _ bool) {}

// ProcessBatch reads eventsBatches which contain rows of data for TS.ADD redis command string
func (p *processor) ProcessBatch(b targets.Batch, doLoad bool) (uint64, uint64) {
	events := b.(*eventsBatch)
	rowCnt := uint64(len(events.rows))
	metricCnt := uint64(0)
	opts := make([]radix.DialOpt, 0)

	if doLoad {
		buflen := rowCnt + 1
		p.rows = make([]chan string, connections)
		p.metrics = make(chan uint64, buflen)
		p.wg = &sync.WaitGroup{}
		var cluster *radix.Cluster
		var standalone *radix.Pool
		if clusterMode {
			cluster = getOSSClusterConn(host, opts, connections)
			defer cluster.Close()
		} else {
			standalone = getStandaloneConn(host, opts, connections)
			defer standalone.Close()
		}

		for i := uint64(0); i < connections; i++ {
			p.rows[i] = make(chan string, buflen)
			p.wg.Add(1)
			if clusterMode {
				go connectionProcessor(p.wg, p.rows[i], p.metrics, cluster)
			} else {
				go connectionProcessor(p.wg, p.rows[i], p.metrics, standalone)
			}
		}
		for _, row := range events.rows {
			key := strings.Split(row, " ")[1]
			start := strings.Index(key, "{")
			end := strings.Index(key, "}")
			tag, _ := strconv.ParseUint(key[start+1:end], 10, 64)
			i := tag % connections
			p.rows[i] <- row
		}

		for i := uint64(0); i < connections; i++ {
			close(p.rows[i])
		}
		p.wg.Wait()
		close(p.metrics)
		for val := range p.metrics {
			metricCnt += val
		}
	}
	events.rows = events.rows[:0]
	ePool.Put(events)
	return metricCnt, rowCnt
}

func (p *processor) Close(_ bool) {
}

func main() {
	log.Println("Starting benchmark")
	config.NoFlowControl = true
	config.HashWorkers = false
	b := benchmark{dbc: &dbCreator{}}
	if config.Workers > 1 {
		panic(fmt.Errorf("You should only use 1 worker and multiple connections per worker (set via --connections)"))
	}
	loader.RunBenchmark(&b)
	log.Println("finished benchmark")
}
