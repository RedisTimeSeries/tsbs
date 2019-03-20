package main

import (
	"bufio"
	"github.com/gomodule/redigo/redis"
	"github.com/timescale/tsbs/load"
	"log"
	"strings"
)

type decoder struct {
	scanner *bufio.Scanner
}

// Reads and returns a text line that encodes a data point for a specif field name.
// Since scanning happens in a single thread, we hold off on transforming it
// to an INSERT statement until it's being processed concurrently by a worker.
func (d *decoder) Decode(_ *bufio.Reader) *load.Point {
	ok := d.scanner.Scan()
	if !ok && d.scanner.Err() == nil { // nothing scanned & no error = EOF
		return nil
	} else if !ok {
		log.Fatalf("scan error: %v", d.scanner.Err())
	}
	return load.NewPoint(d.scanner.Bytes())
}

func sendRedisCommand(line string, conn redis.Conn) {
	t := strings.Split(line, " ")
	s := make([]interface{}, len(t))
	for i, v := range t {
		s[i] = v
	}
	err := conn.Send("TS.ADD", s...)
	//log.Fatalf("cmd: %s", s)
	if err != nil {
		log.Fatalf("TS.ADD failed: %s", err)
	}
}
type eventsBatch struct {
	rows [][]byte
	len int
}

func (eb *eventsBatch) Len() int {
	return eb.len
}

func (eb *eventsBatch) Append(item *load.Point) {
	//that := item.Data.([]byte)
	//eb.rows = append(eb.rows, []byte{})
	eb.len++
}

//var ePool = &sync.Pool{New: func() interface{} { return &eventsBatch{rows: [][]byte{}} }}

type factory struct{}

func (f *factory) New() load.Batch {
	//eb := ePool.Get().(*eventsBatch)
	return &eventsBatch{}
}
