package redistimeseries

import (
	"time"

	"github.com/timescale/tsbs/cmd/tsbs_generate_queries/uses/devops"
	"github.com/timescale/tsbs/cmd/tsbs_generate_queries/utils"
	"github.com/timescale/tsbs/query"
)


// BaseGenerator contains settings specific for Cassandra database.
type BaseGenerator struct {
}

// GenerateEmptyQuery returns an empty query.Cassandra.
func (g *BaseGenerator) GenerateEmptyQuery() query.Query {
	return query.NewRedisTimeSeries()
}


// fill Query fills the query struct with data
func (d *BaseGenerator) fillInQuery(qi query.Query, humanLabel, humanDesc, redisQuery string) {
	q := qi.(*query.RedisTimeSeries)
	q.HumanLabel = []byte(humanLabel)
	q.HumanDescription = []byte(humanDesc)
	q.RedisQuery = []byte(redisQuery)
}


// NewDevops creates a new devops use case query generator.
func (g *BaseGenerator) NewDevops(start, end time.Time, scale int) (utils.QueryGenerator, error) {
	core, err := devops.NewCore(start, end, scale)

	if err != nil {
		return nil, err
	}

	devops := &Devops{
		BaseGenerator: g,
		Core:          core,
	}

	return devops, nil
}
