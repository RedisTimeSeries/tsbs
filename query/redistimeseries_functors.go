package query

import (
	"fmt"
	redistimeseries "github.com/RedisTimeSeries/redistimeseries-go"
	"log"
	"sort"
	"strings"
)

type ResponseFunctor func(interface{}) (interface{}, error)
type void struct{}

var member void

type MultiDataPoint struct {
	Timestamp        int64
	HumanReadbleTime *string
	Values           []*float64
}

type MultiRange struct {
	Names      []string
	Labels     []map[string]string
	DataPoints map[int64]MultiDataPoint
}

func SingleGroupByTime(res interface{}) (result interface{}, err error) {
	parsedRes, err := redistimeseries.ParseRanges(res)
	if err != nil {
		return
	}
	result = MergeSeriesOnTimestamp(parsedRes)
	return
}

func GroupByTimeAndMax(res interface{}) (result interface{}, err error) {
	parsedRes, err := redistimeseries.ParseRanges(res)
	if err != nil {
		return
	}
	result, err = ReduceSeriesOnTimestampBy(parsedRes, MaxReducerSeriesDatapoints)
	return
}

func GroupByTimeAndTag(res interface{}) (result interface{}, err error) {
	parsedRes, err := redistimeseries.ParseRanges(res)
	if err != nil {
		return
	}
	labels, err := GetUniqueLabelValue(parsedRes, "fieldname")
	if err != nil {
		return
	}
	//fmt.Println(result)
	log.Fatal(labels)

	return
}

func GetUniqueLabelValue(series []redistimeseries.Range, label string) (result []string, err error) {
	set := make(map[string]void) // New empty set
	result = make([]string, 0, 0)
	for _, serie := range series {
		value, found := serie.Labels[label]
		if found == true {
			set[value] = member
		}
	}
	for k := range set {
		result = append(result, k)
	}
	return
}

func MergeSeriesOnTimestamp(series []redistimeseries.Range) MultiRange {
	names := make([]string, len(series), len(series))
	labels := make([]map[string]string, len(series), len(series))
	datapoints := make(map[int64]MultiDataPoint)
	for idx, serie := range series {
		names[idx] = serie.Name
		labels[idx] = serie.Labels
		for _, datapoint := range serie.DataPoints {
			_, found := datapoints[datapoint.Timestamp]
			if found == true {
				var v = datapoint.Value
				datapoints[datapoint.Timestamp].Values[idx] = &v
			} else {
				multipointValues := make([]*float64, len(series), len(series))
				for ii := range multipointValues {
					multipointValues[ii] = nil
				}
				var v = datapoint.Value
				multipointValues[idx] = &v
				datapoints[datapoint.Timestamp] = MultiDataPoint{datapoint.Timestamp, nil, multipointValues}
			}
		}
	}
	return MultiRange{names, labels, datapoints}
}

func MaxReducerSeriesDatapoints(series [] redistimeseries.Range) (c redistimeseries.Range, err error) {
	allNames := make([]string, 0, len(series))
	for _, serie := range series {
		allNames = append(allNames, serie.Name)
	}
	var cPoints = make(map[int64]float64)
	pos := 0
	for pos < len(series) {
		serie := series[pos]
		for _, v := range serie.DataPoints {
			_, found := cPoints[v.Timestamp]
			if found == true {
				if cPoints[v.Timestamp] < v.Value {
					cPoints[v.Timestamp] = v.Value
				}
			} else {
				cPoints[v.Timestamp] = v.Value
			}
		}
		pos = pos + 1
	}
	var keys []int
	for k := range cPoints {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)
	datapoints := make([]redistimeseries.DataPoint, 0, len(keys))
	for _, k := range keys {
		dp := cPoints[int64(k)]
		datapoints = append(datapoints, redistimeseries.DataPoint{int64(k), dp})
	}
	name := fmt.Sprintf("max reduction over %s", strings.Join(allNames, " "))
	c = redistimeseries.Range{name, nil, datapoints}
	return
}

func ReduceSeriesOnTimestampBy(series []redistimeseries.Range, reducer func(series [] redistimeseries.Range) (redistimeseries.Range, error)) (outserie redistimeseries.Range, err error) {
	allNames := make([]string, 0, len(series))
	for _, serie := range series {
		allNames = append(allNames, serie.Name)
	}
	if len(series) == 0 {
		return
	}
	if len(series) == 1 {
		outserie = series[0]
		return
	}
	return reducer(series)
}
