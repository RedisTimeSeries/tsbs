package main

import (
	redistimeseries "github.com/RedisTimeSeries/redistimeseries-go"
	"reflect"
	"testing"
)

func TestMergeSeriesOnTimestamp(t *testing.T) {
	type args struct {
		series []redistimeseries.Range
	}
	ts := []int64{1, 2, 3, 4, 5}
	vals := []float64{1.0, 2.0, 3.0, 4.0, 5.0}
	tests := []struct {
		name string
		args args
		want MultiRange
	}{
		//Name       string
		//Labels     map[string]string
		//DataPoints []DataPoint
		{"test 1 series empty labels and datapoints",
			args{
				[]redistimeseries.Range{
					{"serie1", map[string]string{},
						[]redistimeseries.DataPoint{},
					},},},
			MultiRange{[]string{"serie1",}, []map[string]string{{},}, map[int64]MultiDataPoint{}},
		},
		{"test 2 series empty labels and datapoints",
			args{
				[]redistimeseries.Range{
					{"serie1", map[string]string{}, []redistimeseries.DataPoint{},},
					{"serie2", map[string]string{}, []redistimeseries.DataPoint{},},
				},},
			MultiRange{[]string{"serie1", "serie2"}, []map[string]string{{}, {},}, map[int64]MultiDataPoint{}},
		},
		{"test 2 series with labels and empty datapoints",
			args{
				[]redistimeseries.Range{
					{"serie1", map[string]string{"host": "1"}, []redistimeseries.DataPoint{},},
					{"serie2", map[string]string{"host": "2"}, []redistimeseries.DataPoint{},},
				},},
			MultiRange{[]string{"serie1", "serie2"}, []map[string]string{{"host": "1"}, {"host": "2"},}, map[int64]MultiDataPoint{}},
		},
		{"test 2 series with labels and datapoints",
			args{
				[]redistimeseries.Range{
					{"serie1", map[string]string{"host": "1"}, []redistimeseries.DataPoint{{ts[0], vals[0]}},},
					{"serie2", map[string]string{"host": "2"}, []redistimeseries.DataPoint{{ts[0], vals[0]}},}},
			},
			MultiRange{
				[]string{"serie1", "serie2"},
				[]map[string]string{{"host": "1"}, {"host": "2"},},
				map[int64]MultiDataPoint{ts[0]: {
					Timestamp: ts[0],
					Values:    []*float64{&vals[0], &vals[0],},
				}},},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MergeSeriesOnTimestamp(tt.args.series)
			if !reflect.DeepEqual(got.Names, tt.want.Names) {
				t.Errorf("MergeSeriesOnTimestamp() Error on Names got %v, want %v", got.Names, tt.want.Names)
			}
			if !reflect.DeepEqual(got.Labels, tt.want.Labels) {
				t.Errorf("MergeSeriesOnTimestamp() Error on Labels got %v, want %v", got.Labels, tt.want.Labels)
			}
			if !reflect.DeepEqual(got.DataPoints, tt.want.DataPoints) {
				t.Errorf("MergeSeriesOnTimestamp() Error on DataPoints got %v, want %v", got.DataPoints, tt.want.DataPoints)
			}
		})
	}
}
