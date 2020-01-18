package tsbs_run_queries_redistimeseries

import (
	"database/sql"
	"encoding/json"
	redistimeseries "github.com/RedisTimeSeries/redistimeseries-go"
	"github.com/pkg/errors"
	"github.com/timescale/tsbs/query"
	"sort"
	"strings"
	"time"
	"fmt"
)

func ByteArrayToInterfaceArray(qry [][]byte) []interface{} {
	commandArgs := make([]interface{}, len(qry))
	for i := 0; i < len(qry); i++ {
		commandArgs[i] = qry[i]
	}
	return commandArgs
}

func ByteArrayToStringArray(qry [][]byte) []string {
	commandArgs := make([]string, len(qry))
	for i := 0; i < len(qry); i++ {
		commandArgs[i] = string(qry[i])
	}
	return commandArgs
}

func mapRows(r *sql.Rows) []map[string]interface{} {
	rows := []map[string]interface{}{}
	cols, _ := r.Columns()
	for r.Next() {
		row := make(map[string]interface{})
		values := make([]interface{}, len(cols))
		for i := range values {
			values[i] = new(interface{})
		}

		err := r.Scan(values...)
		if err != nil {
			panic(errors.Wrap(err, "error while reading values"))
		}

		for i, column := range cols {
			row[column] = *values[i].(*interface{})
		}
		rows = append(rows, row)
	}
	return rows
}

// prettyPrintResponseRange prints a Query and its response in JSON format with two
// keys: 'query' which has a value of the RedisTimeseries query used to generate the second key
// 'results' which is an array of each element in the return set.
func prettyPrintResponseRange(responses []interface{}, q *query.RedisTimeSeries) {
	full := make(map[string]interface{})
	for idx, qry := range q.RedisQueries {
		resp := make(map[string]interface{})
		fullcmd := append([][]byte{q.CommandNames[idx]}, qry...)
		resp["query"] = strings.Join(ByteArrayToStringArray(fullcmd), " ")

		res := responses[idx]
		switch v := res.(type) {
		case []redistimeseries.Range:
			resp["client_side_work"] = q.ApplyFunctor
			rows := []map[string]interface{}{}
			for _, r := range res.([]redistimeseries.Range) {
				row := make(map[string]interface{})
				values := make(map[string]interface{})
				values["datapoints"] = r.DataPoints
				values["labels"] = r.Labels
				row[r.Name] = values
				rows = append(rows, row)
			}
			resp["results"] = rows
		case redistimeseries.Range:
			resp["client_side_work"] = q.ApplyFunctor
			resp["results"] = res.(redistimeseries.Range)
		case []query.MultiRange:
			resp["client_side_work"] = q.ApplyFunctor
			rows := []map[string]interface{}{}
			for _, converted := range res.([]query.MultiRange) {
				query_result := map[string]interface{}{}
				//converted := r.(query.MultiRange)
				query_result["names"] = converted.Names
				query_result["labels"] = converted.Labels
				datapoints := make([]query.MultiDataPoint, 0, len(converted.DataPoints))
				var keys []int
				for k := range converted.DataPoints {
					keys = append(keys, int(k))
				}
				sort.Ints(keys)
				for _, k := range keys {
					dp := converted.DataPoints[int64(k)]
					time_str := time.Unix(dp.Timestamp/1000, 0).Format(time.RFC3339)
					dp.HumanReadbleTime = &time_str
					datapoints = append(datapoints, dp)
				}
				query_result["datapoints"] = datapoints
				rows = append(rows, query_result)
			}
			resp["results"] = rows
		case query.MultiRange:
			resp["client_side_work"] = q.ApplyFunctor
			query_result := map[string]interface{}{}
			converted := res.(query.MultiRange)
			query_result["names"] = converted.Names
			query_result["labels"] = converted.Labels
			datapoints := make([]query.MultiDataPoint, 0, len(converted.DataPoints))
			var keys []int
			for k := range converted.DataPoints {
				keys = append(keys, int(k))
			}
			sort.Ints(keys)
			for _, k := range keys {
				dp := converted.DataPoints[int64(k)]
				time_str := time.Unix(dp.Timestamp/1000, 0).Format(time.RFC3339)
				dp.HumanReadbleTime = &time_str
				datapoints = append(datapoints, dp)
			}
			query_result["datapoints"] = datapoints
			resp["results"] = query_result
		default:
			fmt.Printf("I don't know about type %T!\n", v)
		}

		full[fmt.Sprintf("query %d", idx+1)] = resp
	}

	line, err := json.MarshalIndent(full, "", "  ")
	if err != nil {
		panic(err)
	}

	fmt.Println(string(line) + "\n")
}

func (p *processor) applyResponseFunctions(tq *query.RedisTimeSeries) (res []*query.Stat, err error) {
	//if p.opts.debug {
	//	fmt.Println(fmt.Sprintf("Applying functor %s on %s", tq.Functor, tq.HumanLabel))
	//}
	//switch tq.Functor {
	//case reflect_SingleGroupByTime:
	//	if p.opts.debug {
	//		fmt.Println(fmt.Sprintf("Applying functor reflect_SingleGroupByTime %s", reflect_SingleGroupByTime))
	//	}
	//	result, err = query.SingleGroupByTime(res)
	//	if err != nil {
	//		return nil, nil, nil, err
	//	}
	//case reflect_GroupByTimeAndMax:
	//	if p.opts.debug {
	//		fmt.Println(fmt.Sprintf("Applying functor reflect_GroupByTimeAndMax %s", reflect_GroupByTimeAndMax))
	//	}
	//	result, err = query.GroupByTimeAndMax(res)
	//	if err != nil {
	//		return nil, nil, nil, err
	//	}
	//case reflect_GroupByTimeAndTagMax:
	//	if p.opts.debug {
	//		fmt.Println(fmt.Sprintf("Applying functor reflect_GroupByTimeAndTagMax %s", reflect_GroupByTimeAndTagMax))
	//	}
	//	result, err = query.GroupByTimeAndTagMax(res)
	//	if err != nil {
	//		return nil, nil, nil, err
	//	}
	//case reflect_GroupByTimeAndTagHostname:
	//	if p.opts.debug {
	//		fmt.Println(fmt.Sprintf("Applying functor reflect_GroupByTimeAndTagHostname %s", reflect_GroupByTimeAndTagHostname))
	//	}
	//	result, err = query.GroupByTimeAndTagHostname(res)
	//	if err != nil {
	//		return nil, nil, nil, err
	//	}
	//case reflect_HighCpu:
	//	if p.opts.debug {
	//		fmt.Println(fmt.Sprintf("Applying functor reflect_HighCpu %s", reflect_HighCpu))
	//	}
	//	result, err = query.HighCpu(res)
	//	if err != nil {
	//		return nil, nil, nil, err
	//	}
	//default:
	//	errors.Errorf("The selected functor %s is not known!\n", tq.Functor)
	//}
	return nil, err
}
