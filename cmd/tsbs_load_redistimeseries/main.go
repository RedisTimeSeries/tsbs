package main

import (
	"bufio"
	"crypto/md5"
	"fmt"
	"github.com/mediocregopher/radix/v3"
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
	loader     load.BenchmarkRunner
	config     load.BenchmarkRunnerConfig
	target     targets.ImplementedTarget
	cluster    *radix.Cluster
	standalone *radix.Pool
	addresses  []string
	slots      [][][2]uint16
	conns      []radix.Client
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

	opts := make([]radix.DialOpt, 0)
	if clusterMode {
		cluster = getOSSClusterConn(host, opts, connections)
		cluster.Sync()
		topology := cluster.Topo().Primaries().Map()
		addresses = make([]string, 0)
		slots = make([][][2]uint16, 0)
		conns = make([]radix.Client, 0)
		for nodeAddress, node := range topology {
			addresses = append(addresses, nodeAddress)
			slots = append(slots, node.Slots)
			conn, _ := cluster.Client(nodeAddress)
			conns = append(conns, conn)
		}
		fmt.Println(addresses)
		fmt.Println(slots)
		fmt.Println(conns)
	} else {
		standalone = getStandaloneConn(host, opts, connections)
	}
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
	slotS := strings.Split(row, " ")[0]
	clusterSlot, _ := strconv.ParseInt(slotS, 10, 0)
	return uint(clusterSlot) % i.partitions
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

func nodeThatContainsSlot(slots [][][2]uint16, slot uint16) (result int) {
	result = -1
	for nodePos, slotGroup := range slots {
		for _, i2 := range slotGroup {
			if slot >= i2[0] && slot <= i2[1] {
				result = nodePos
				return
			}
		}
	}
	return
}

func connectionProcessorCluster(wg *sync.WaitGroup, rows chan string, metrics chan uint64, cluster *radix.Cluster, clusterNodes int, addresses []string, slots [][][2]uint16, conns []radix.Client) {
	cmds := make([][]radix.CmdAction, clusterNodes, clusterNodes)
	curPipe := make([]uint64, clusterNodes, clusterNodes)
	currMetricCount := make([]int, clusterNodes, clusterNodes)
	for i := 0; i < clusterNodes; i++ {
		cmds[i] = make([]radix.CmdAction, 0, 0)
		curPipe[i] = 0
		currMetricCount[i] = 0
	}

	for row := range rows {
		slot, cmd, _, metricCount := buildCommand(row, compressionEnabled == false)
		comdPos := nodeThatContainsSlot(slots, slot)
		currMetricCount[comdPos] += metricCount
		cmds[comdPos] = append(cmds[comdPos], cmd)
		curPipe[comdPos]++

		if curPipe[comdPos] == pipeline {
			err := conns[comdPos].Do(radix.Pipeline(cmds[comdPos]...))
			if err != nil {
				log.Fatalf("Flush failed with %v", err)
			}
			metrics <- uint64(currMetricCount[comdPos])
			currMetricCount[comdPos] = 0
			cmds[comdPos] = make([]radix.CmdAction, 0, 0)
			curPipe[comdPos] = 0
		}

	}
	for comdPos, u := range curPipe {
		if u > 0 {
			err := conns[comdPos].Do(radix.Pipeline(cmds[comdPos]...))
			if err != nil {
				log.Fatalf("Flush failed with %v", err)
			}
			metrics <- uint64(currMetricCount[comdPos])
		}
	}
	wg.Done()
}

func connectionProcessor(wg *sync.WaitGroup, rows chan string, metrics chan uint64, conn radix.Client) {
	cmds := make([][]radix.CmdAction, 1, 1)
	cmds[0] = make([]radix.CmdAction, 0, 0)
	curPipe := make([]uint64, 1, 1)
	curPipe[0] = 0
	currMetricCount := 0
	comdPos := 0

	for row := range rows {
		_, cmd, _, metricCount := buildCommand(row, compressionEnabled == false)
		currMetricCount += metricCount
		cmds[comdPos] = append(cmds[comdPos], cmd)
		curPipe[comdPos]++

		if curPipe[comdPos] == pipeline {
			err := conn.Do(radix.Pipeline(cmds[comdPos]...))
			if err != nil {
				log.Fatalf("Flush failed with %v", err)
			}
			metrics <- uint64(currMetricCount)
			currMetricCount = 0
			cmds[comdPos] = make([]radix.CmdAction, 0, 0)
			curPipe[comdPos] = 0
		}
	}
	for comdPos, u := range curPipe {
		if u > 0 {
			err := conn.Do(radix.Pipeline(cmds[comdPos]...))
			if err != nil {
				log.Fatalf("Flush failed with %v", err)
			}
			metrics <- uint64(currMetricCount)
		}
	}
	wg.Done()
}

func (p *processor) Init(_ int, _ bool, _ bool) {}

// ProcessBatch reads eventsBatches which contain rows of data for TS.ADD redis command string
func (p *processor) ProcessBatch(b targets.Batch, doLoad bool) (uint64, uint64) {
	events := b.(*eventsBatch)
	rowCnt := uint64(len(events.rows))
	metricCnt := uint64(0)

	if doLoad {
		buflen := rowCnt + 1
		p.rows = make([]chan string, connections)
		p.metrics = make(chan uint64, buflen)
		p.wg = &sync.WaitGroup{}

		for i := uint64(0); i < connections; i++ {
			p.rows[i] = make(chan string, buflen)
			p.wg.Add(1)
			if clusterMode {
				go connectionProcessorCluster(p.wg, p.rows[i], p.metrics, cluster, len(addresses), addresses, slots, conns)
			} else {
				go connectionProcessor(p.wg, p.rows[i], p.metrics, standalone)
			}
		}
		for _, row := range events.rows {
			slotS := strings.Split(row, " ")[0]
			clusterSlot, _ := strconv.ParseInt(slotS, 10, 0)
			i := uint64(clusterSlot) % connections
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
