package main

import (
	"fmt"
	"github.com/mediocregopher/radix/v3"
	"github.com/timescale/tsbs/pkg/targets"
	"io"
	"log"
	"strings"
	"sync"
)

type processor struct {
	dbc             *dbCreator
	rows            []chan string
	metrics         chan uint64
	wg              *sync.WaitGroup
	redisCluster    *radix.Cluster
	redisStandalone *radix.Pool
}

func (p *processor) Init(workerNum int, doLoad, hashWorkers bool) {
	if doLoad {
		connectionStr := fmt.Sprintf("%s:%d", host, port)
		opts := make([]radix.DialOpt, 0)
		if password != "" {
			opts = append(opts, radix.DialAuthPass(password))
		}
		if cluster {
			p.redisCluster = getOSSClusterConn(connectionStr, opts, 1)
		} else {
			var err error = nil
			p.redisStandalone, err = getStandaloneConn(opts, p, connectionStr)
			if err != nil {
				log.Fatalf("Error preparing for benchmark, while creating new connection. error = %v", err)
			}
		}
	}

}

func (p *processor) ProcessBatch(b targets.Batch, doLoad bool) (uint64, uint64) {
	batch := b.(*batch)

	// Write the batch: try until backoff is not needed.
	readRows := uint(0)
	pipeCmds := make([]radix.CmdAction, 0, 0)
	for {
		line, err := batch.buf.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Fatalf("Error writing: %v\n", err.Error())
		}
		if line == "\n" {
			break
		}
		fmt.Println(readRows)
		args := strings.Split(line[0:len(line)-1], " ")
		cmd := radix.FlatCmd(nil, "TS.ADD", args[0], args[1], args[2], "LABELS", args[3:])
		if doLoad {
			pipeCmds = append(pipeCmds, cmd)
			if len(pipeCmds) >= int(pipeline) {
				print(len(pipeCmds))
				if cluster {
					err = p.redisCluster.Do(radix.Pipeline(pipeCmds...))
				} else {
					err = p.redisStandalone.Do(radix.Pipeline(pipeCmds...))
				}
				if err != nil {
					log.Fatalf("Error sending command to Redis: %v\n", err.Error())
				}
				pipeCmds = pipeCmds[:0]
				pipeCmds = make([]radix.CmdAction, 0, 0)
			}
		}
		readRows++
	}
	if readRows != batch.rows {
		log.Fatalf("Expected the total batch read rows and batch.rows to match. readRows: %d batch.rows %d\n", readRows, batch.rows)
	}
	metricCnt := batch.metrics
	rowCnt := batch.rows

	// Return the batch buffer to the pool.
	batch.buf.Reset()
	bufPool.Put(batch.buf)
	return metricCnt, uint64(rowCnt)
}

func (p *processor) Close(_ bool) {
}
