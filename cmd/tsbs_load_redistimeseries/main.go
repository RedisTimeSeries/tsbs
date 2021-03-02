package main

import (
	"bufio"
	"crypto/md5"
	"encoding/binary"
	"fmt"
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

	"github.com/gomodule/redigo/redis"
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
)

// Global vars
var (
	loader load.BenchmarkRunner
	config load.BenchmarkRunnerConfig
	target targets.ImplementedTarget
	//bufPool sync.Pool
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

	//pflag.Uint64Var(&pipeline, "pipeline", 50, "The pipeline's size")
	//pflag.BoolVar(&singleQueue, "single-queue", true, "Whether to use a single queue")
	//pflag.BoolVar(&compressionEnabled, "compression-enabled", true, "Whether to use compressed time series")
	//pflag.Uint64Var(&checkChunks, "check-chunks", 0, "Whether to perform post ingestion chunck count")
	//pflag.StringVar(&dataModel, "data-model", "redistimeseries", "Data model (redistimeseries, rediszsetdevice, rediszsetmetric, redisstream)")
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

	loader = load.GetBenchmarkRunner(config)

}

type benchmark struct {
	dbc *dbCreator
}

func (b *benchmark) GetDataSource() targets.DataSource {
	log.Printf("creating DS from %s", config.FileName);
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
//
//func (b *benchmark) GetPointDecoder(br *bufio.Reader) load.PointDecoder {
//	return &decoder{scanner: bufio.NewScanner(br)}
//}
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

func connectionProcessor(wg *sync.WaitGroup, rows chan string, metrics chan uint64, conn redis.Conn, id uint64) {
	curPipe := uint64(0)
	//fmt.Println(fmt.Sprintf("wg started for id %d\n",id))

	for row := range rows {
		cmdname, s := buildCommand(row, compressionEnabled == false)
		var err error

		if curPipe == pipeline {
			cnt, err := sendRedisFlush(curPipe, conn)
			if err != nil {
				log.Fatalf("Flush failed with %v", err)
			}
			metrics <- cnt
			curPipe = 0
		}
		err = sendRedisCommand(conn, cmdname, s)
		if err != nil {
			log.Fatalf("sendRedisCommand failed with %v", err)
		}
		curPipe++

	}
	if curPipe > 0 {
		cnt, err := sendRedisFlush(curPipe, conn)
		if err != nil {
			log.Fatalf("Flush failed with %v", err)
		}
		metrics <- cnt
	}
	wg.Done()
	//fmt.Println(fmt.Sprintf("wg done for id %d\n",id))
}

func (p *processor) Init(_ int, _ bool, _ bool) {}

// ProcessBatch reads eventsBatches which contain rows of data for TS.ADD redis command string
func (p *processor) ProcessBatch(b targets.Batch, doLoad bool) (uint64, uint64) {
	events := b.(*eventsBatch)
	rowCnt := uint64(len(events.rows))
	metricCnt := uint64(0)
	// indexer := &RedisIndexer{partitions: uint(connections)}
	if doLoad {
		buflen := rowCnt + 1
		p.rows = make([]chan string, connections)
		p.metrics = make(chan uint64, buflen)
		p.wg = &sync.WaitGroup{}
		for i := uint64(0); i < connections; i++ {
			conn := p.dbc.client.Pool.Get()
			defer conn.Close()
			p.rows[i] = make(chan string, buflen)
			p.wg.Add(1)
			go connectionProcessor(p.wg, p.rows[i], p.metrics, conn, i)
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
		//fmt.Println("out\n")

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

func runCheckData() {
	log.Println("Running post ingestion data check")
	conn, err := redis.DialURL(fmt.Sprintf("redis://%s", host))
	if err != nil {
		log.Fatalf("Error while dialing %v", err)
	}
	_, err = conn.Do("PING")
	if err != nil {
		log.Fatalf("Error while PING %v", err)
	}

	cursor := 0
	total := 0
	for {
		rep, _ := redis.Values(conn.Do("SCAN", cursor))
		cursor, _ = redis.Int(rep[0], nil)
		keys, _ := redis.Strings(rep[1], nil)
		for _, key := range keys {
			total++
			info, _ := redis.Values(conn.Do("TS.INFO", key))
			chunks, _ := redis.Int(info[5], nil)
			if chunks != int(checkChunks) {
				log.Printf("Verification error: key %v has %v chunks", key, chunks)
			}
		}
		if cursor == 0 {
			break
		}
	}
	log.Printf("Verified %v keys", total)
}

func main() {
	//workQueues := uint(load.WorkerPerQueue)
	//if singleQueue {
	//	workQueues = load.SingleQueue
	//}
	log.Println("Starting benchmark")
	config.NoFlowControl = true;
	loader.RunBenchmark(&benchmark{dbc: &dbCreator{}})
	log.Println("finished benchmark")
	if checkChunks > 0 {
		runCheckData()
	}
}
