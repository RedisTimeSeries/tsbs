package main

import (
	"fmt"
	redistimeseries "github.com/RedisTimeSeries/redistimeseries-go"
	"sort"
	"strings"
)

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

func maxReducerOnTimestamp(points []redistimeseries.DataPoint) (outpoint redistimeseries.DataPoint, err error) {
	if len(points) == 0 {
		return
	}
	ts := points[0].Timestamp
	value := points[0].Value
	for idx, v := range points {
		if ts != v.Timestamp {
			err = fmt.Errorf("there are at least two distinct timestamps on the datapoints slice. Error on slice pos %d", idx)
			return
		}
		if v.Value > value {
			value = v.Value
		}
	}
	outpoint = redistimeseries.DataPoint{ts, value}
	return
}

func maxReducerSeriesDatapoints(series [] redistimeseries.Range) (c redistimeseries.Range, err error) {
	allNames := make([]string, 0, len(series))
	for _, serie := range series {
		allNames = append(allNames, serie.Name)
	}
	var cPoints = make(map[int64]float64)
	pos := 0
	for ok := true; ok; ok = !(pos < len(series)) {
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
	outserie = series[0]
	return reducer(series)
}
