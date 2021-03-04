package main

import (
	"github.com/mediocregopher/radix/v3"

	"strings"
	"sync"

	"github.com/timescale/tsbs/pkg/data"
	"github.com/timescale/tsbs/pkg/targets"
)

func buildCommand(line string, forceUncompressed bool) (cmdA radix.CmdAction, tscreate bool, metricCount int) {
	t := strings.Split(line, " ")
	metricCount = 1
	tscreate = false
	cmdname := t[0]
	if cmdname == "TS.CREATE" {
		tscreate = true
		metricCount = 0
	}
	if cmdname == "TS.MADD" {
		metricCount = (len(t)-1)/3
	}
	key := t[1]
	cmdA = radix.FlatCmd(nil, cmdname, key, t[2:])
	return
}

type eventsBatch struct {
	rows []string
}

func (eb *eventsBatch) Len() uint {
	return uint(len(eb.rows))
}

func (eb *eventsBatch) Append(item data.LoadedPoint) {
	that := item.Data.(string)
	eb.rows = append(eb.rows, that)
}

var ePool = &sync.Pool{New: func() interface{} { return &eventsBatch{rows: []string{}} }}

type factory struct{}

func (f *factory) New() targets.Batch {
	return ePool.Get().(*eventsBatch)
}
