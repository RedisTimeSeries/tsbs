package main

import (
	"fmt"
	redistimeseries "github.com/RedisTimeSeries/redistimeseries-go"
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

func ReduceSeriesOnTimestampBy(series []redistimeseries.Range, reducername string, reducer func(points []redistimeseries.DataPoint) redistimeseries.DataPoint) (outserie redistimeseries.Range) {
	allNames := make([]string, 0, len(series))
	for _, serie := range series {
		allNames = append(allNames, serie.Name)
	}

	name := fmt.Sprintf("%s reduction over %s", reducername, strings.Join(allNames, " "))
	//datapoints := make([]redistimeseries.DataPoint,0,0)
	//for idx, serie := range series {
	//	names[idx] = serie.Name
	//	labels[idx] = serie.Labels
	//	for _, datapoint := range serie.DataPoints {
	//		_, found := datapoints[datapoint.Timestamp]
	//		if found == true {
	//			var v = datapoint.Value
	//			datapoints[datapoint.Timestamp].Values[idx] = &v
	//		} else {
	//			multipointValues := make([]*float64, len(series), len(series))
	//			for ii := range multipointValues {
	//				multipointValues[ii] = nil
	//			}
	//			var v = datapoint.Value
	//			multipointValues[idx] = &v
	//			datapoints[datapoint.Timestamp] = MultiDataPoint{datapoint.Timestamp, nil, multipointValues}
	//		}
	//	}
	//}
	return redistimeseries.Range{name, nil, nil}
}
