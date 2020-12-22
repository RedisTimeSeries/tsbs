package main

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/bmizerany/assert"
	"sync"
	"testing"

	"github.com/timescale/tsbs/pkg/data"
)

func TestBatch(t *testing.T) {
	bufPool = sync.Pool{
		New: func() interface{} {
			return bytes.NewBuffer(make([]byte, 0, 4*1024*1024))
		},
	}
	f := &factory{}
	b := f.New().(*batch)
	if b.Len() != 0 {
		t.Errorf("batch not initialized with count 0")
	}
	p := data.LoadedPoint{
		Data: []byte("cpu_usage_guest_nice{1998426147} 1451606400000 38.24311829 hostname host_0 region eu-west-1 datacenter eu-west-1b measurement cpu fieldname usage_guest_nice\n"),
	}
	b.Append(p)
	if b.Len() != 1 {
		t.Errorf("batch count is not 1 after first append")
	}
	if b.rows != 1 {
		t.Errorf("batch row count is not 1 after first append")
	}
	if b.metrics != 1 {
		t.Errorf("batch metric count is not 2 after first append")
	}

	p = data.LoadedPoint{
		Data: []byte("cpu_usage_guest_nice{1998426147} 1451606400000 38.24311829 hostname host_0"),
	}
	b.Append(p)
	if b.Len() != 2 {
		t.Errorf("batch count is not 2 after first append")
	}
	if b.rows != 2 {
		t.Errorf("batch row count is not 2 after first append")
	}
	if b.metrics != 2 {
		t.Errorf("batch metric count is not 2 after first append")
	}

	p = data.LoadedPoint{
		Data: []byte("bad_point"),
	}
	errMsg := ""
	fatal = func(args ...interface{}) {
		errMsg = fmt.Sprintf(args[0].(string), args[1:]...)
	}
	b.Append(p)
	assert.NotEqual(t, errMsg, "")
	assert.Equal(t, errMsg, "parse error: line does not have 3 tuples, has 1")
}

func TestFileDataSourceNextItem(t *testing.T) {
	cases := []struct {
		desc        string
		input       string
		result      []byte
		shouldFatal bool
	}{
		{
			desc:   "correct input",
			input:  "cpu_usage_guest_nice{1998426147} 1451606400000 38.24311829 hostname host_0 region eu-west-1 datacenter eu-west-1b measurement cpu fieldname usage_guest_nice\n\n",
			result: []byte("cpu_usage_guest_nice{1998426147} 1451606400000 38.24311829 hostname host_0 region eu-west-1 datacenter eu-west-1b measurement cpu fieldname usage_guest_nice"),
		},
		{
			desc:   "correct input with extra",
			input:  "cpu_usage_guest_nice{1998426147} 1451606400000 38.24311829 hostname host_0 region eu-west-1 datacenter eu-west-1b measurement cpu fieldname usage_guest_nice\n\nextra_is_ignored",
			result: []byte("cpu_usage_guest_nice{1998426147} 1451606400000 38.24311829 hostname host_0 region eu-west-1 datacenter eu-west-1b measurement cpu fieldname usage_guest_nice"),
		},
	}

	for _, c := range cases {
		br := bufio.NewReader(bytes.NewReader([]byte(c.input)))
		ds := &fileDataSource{scanner: bufio.NewScanner(br)}
		p := ds.NextItem()
		data := p.Data.([]byte)
		if !bytes.Equal(data, c.result) {
			t.Errorf("%s: incorrect result: got\n%v\nwant\n%v", c.desc, data, c.result)
		}
	}
}

func TestDecodeEOF(t *testing.T) {
	input := []byte("cpu_usage_guest_nice{1998426147} 1451606400000 38.24311829 hostname host_0 region eu-west-1 datacenter eu-west-1b measurement cpu fieldname usage_guest_nice")
	br := bufio.NewReader(bytes.NewReader([]byte(input)))
	ds := &fileDataSource{scanner: bufio.NewScanner(br)}
	_ = ds.NextItem()
	// nothing left, should be EOF
	p := ds.NextItem()
	if p.Data != nil {
		t.Errorf("expected p to be nil, got %v", p)
	}
}
