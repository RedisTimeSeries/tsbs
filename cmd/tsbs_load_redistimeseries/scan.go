package main

import (
	"bufio"
	"bytes"
	"github.com/timescale/tsbs/pkg/data"
	"github.com/timescale/tsbs/pkg/data/usecases/common"
	"github.com/timescale/tsbs/pkg/targets"

	"strings"
)

type fileDataSource struct {
	scanner *bufio.Scanner
}

const errNotThreeTuplesFmt = "parse error: line does not have 3 tuples, has %d"

var newLine = []byte("\n")

func (d *fileDataSource) NextItem() data.LoadedPoint {
	ok := d.scanner.Scan()
	if !ok && d.scanner.Err() == nil { // nothing scanned & no error = EOF
		return data.LoadedPoint{}
	} else if !ok {
		fatal("scan error: %v", d.scanner.Err())
		return data.LoadedPoint{}
	}
	return data.NewLoadedPoint(d.scanner.Bytes())
}

func (d *fileDataSource) Headers() *common.GeneratedDataHeaders { return nil }

type batch struct {
	buf     *bytes.Buffer
	rows    uint
	metrics uint64
}

func (b *batch) Len() uint {
	return b.rows
}

func (b *batch) Append(item data.LoadedPoint) {
	that := item.Data.([]byte)
	thatStr := string(that)
	b.rows++
	// Each redistimeseries line is format "key timestamp value [tags...]", so we split by space
	args := strings.Split(thatStr, " ")
	// at least we should have 3 args
	if len(args) < 3 {
		fatal(errNotThreeTuplesFmt, len(args))
		return
	}
	b.metrics++
	b.buf.Write(that)
	b.buf.Write(newLine)
}

type factory struct{}

func (f *factory) New() targets.Batch {
	return &batch{buf: bufPool.Get().(*bytes.Buffer)}
}
