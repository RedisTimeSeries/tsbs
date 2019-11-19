package query

import (
	"fmt"
	"sync"
)

// RedisTimeSeries encodes a RedisTimeSeries request. This will be serialized for use
// by the tsbs_run_queries_redistimeseries program.
type RedisTimeSeries struct {
	HumanLabel       []byte
	HumanDescription []byte

	RedisQuery []byte
	id         uint64
}

// RedisTimeSeriesPool is a sync.Pool of RedisTimeSeries Query types
var RedisTimeSeriesPool = sync.Pool{
	New: func() interface{} {
		return &RedisTimeSeries{
			HumanLabel:       make([]byte, 0, 1024),
			HumanDescription: make([]byte, 0, 1024),
			RedisQuery:       make([]byte, 0, 1024),
		}
	},
}

// NewRedisTimeSeries returns a new RedisTimeSeries Query instance
func NewRedisTimeSeries() *RedisTimeSeries {
	return RedisTimeSeriesPool.Get().(*RedisTimeSeries)
}

// GetID returns the ID of this Query
func (q *RedisTimeSeries) GetID() uint64 {
	return q.id
}

// SetID sets the ID for this Query
func (q *RedisTimeSeries) SetID(n uint64) {
	q.id = n
}

// String produces a debug-ready description of a Query.
func (q *RedisTimeSeries) String() string {
	return fmt.Sprintf("HumanLabel: %s, HumanDescription: %s, Query: %s", q.HumanLabel, q.HumanDescription, q.RedisQuery)
}

// HumanLabelName returns the human readable name of this Query
func (q *RedisTimeSeries) HumanLabelName() []byte {
	return q.HumanLabel
}

// HumanDescriptionName returns the human readable description of this Query
func (q *RedisTimeSeries) HumanDescriptionName() []byte {
	return q.HumanDescription
}

// Release resets and returns this Query to its pool
func (q *RedisTimeSeries) Release() {
	q.HumanLabel = q.HumanLabel[:0]
	q.HumanDescription = q.HumanDescription[:0]
	q.id = 0

	q.RedisQuery = q.RedisQuery[:0]

	RedisTimeSeriesPool.Put(q)
}
