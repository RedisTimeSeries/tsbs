package main

import (
	"bufio"
	"crypto/md5"
	"flag"
	"github.com/pkg/profile"
	"github.com/timescale/tsbs/load"
	"log"
	"math/rand"
)

// Program option vars:
var (
	host        string
)

// Global vars
var (
	loader *load.BenchmarkRunner
	//bufPool sync.Pool
)
// allows for testing
var fatal = log.Fatal
var md5h = md5.New()

// Parse args:
func init() {
	loader = load.GetBenchmarkRunnerWithBatchSize(1000)
	flag.StringVar(&host, "host", "localhost:6379", "Provide host:port for redis connection")
	flag.Parse()
}

type benchmark struct {
	dbc *dbCreator
}

type RedisIndexer struct{
	partitions uint
}

func (i *RedisIndexer) GetIndex(p *load.Point) int {
	//row := p.Data.(string)
	//key := strings.Split(row, " ")[0]
	//_, _ = io.WriteString(md5h, key)
	//hash := binary.LittleEndian.Uint32(md5h.Sum(nil))
	//md5h.Reset()
	//return int(uint(hash) % i.partitions)
	return rand.Int() % int(i.partitions)
}

func (b *benchmark) GetPointDecoder(br *bufio.Reader) load.PointDecoder {
	return &decoder{scanner: bufio.NewScanner(br)}
}

func (b *benchmark) GetBatchFactory() load.BatchFactory {
	return &factory{}
}

func (b *benchmark) GetPointIndexer(maxPartitions uint) load.PointIndexer {
	return &RedisIndexer{partitions: maxPartitions}
}

func (b *benchmark) GetProcessor() load.Processor {
	return &processor{b.dbc}
}

func (b *benchmark) GetDBCreator() load.DBCreator {
	return b.dbc
}
type processor struct {
	dbc *dbCreator
}

func (p *processor) Init(_ int, _ bool) {}

// ProcessBatch reads eventsBatches which contain rows of data for TS.ADD redis command string
func (p *processor) ProcessBatch(b load.Batch, doLoad bool) (uint64, uint64) {
	events := b.(*eventsBatch)
	cmdLen := 0
	if doLoad {
		conn := p.dbc.client.Pool.Get()
		for _, row := range events.rows {
			sendRedisCommand(string(row), conn)
			cmdLen++
		}

		err := conn.Flush()
		if err != nil {
			log.Fatalf("Error while inserting: %v", err)
		}

		for i := 0; i < cmdLen; i++ {
			_, err = conn.Receive()
			if err != nil {
				log.Fatalf("Error while inserting: %v", err)
			}
		}
	}
	rowCnt := uint64(events.len)
	//ePool.Put(events)
	return rowCnt, rowCnt
}

func (p *processor) Close(_ bool) {}

func main() {
	defer profile.Start().Stop()
	loader.RunBenchmark(&benchmark{dbc: &dbCreator{}}, load.WorkerPerQueue)
}
