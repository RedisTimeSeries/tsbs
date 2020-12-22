package main

import (
	"bytes"
	"github.com/bmizerany/assert"
	"github.com/timescale/tsbs/pkg/data"
	"sync"
	"testing"
	"time"
)

func emptyLog(_ string, _ ...interface{}) (int, error) {
	return 0, nil
}

//
//func TestProcessorInit(t *testing.T) {
//	daemonURLs = []string{"url1", "url2"}
//	printFn = emptyLog
//	p := &processor{}
//	p.Init(0, false, false)
//	p.Close(true)
//	if got := p.httpWriter.c.Host; got != daemonURLs[0] {
//		t.Errorf("incorrect host: got %s want %s", got, daemonURLs[0])
//	}
//	if got := p.httpWriter.c.Database; got != loader.DatabaseName() {
//		t.Errorf("incorrect database: got %s want %s", got, loader.DatabaseName())
//	}
//
//	p = &processor{}
//	p.Init(1, false, false)
//	p.Close(true)
//	if got := p.httpWriter.c.Host; got != daemonURLs[1] {
//		t.Errorf("incorrect host: got %s want %s", got, daemonURLs[1])
//	}
//
//	p = &processor{}
//	p.Init(len(daemonURLs), false, false)
//	p.Close(true)
//	if got := p.httpWriter.c.Host; got != daemonURLs[0] {
//		t.Errorf("incorrect host: got %s want %s", got, daemonURLs[0])
//	}
//
//}
//
//func TestProcessorInitWithHTTPWriterConfig(t *testing.T) {
//	var b bytes.Buffer
//	counter := int64(0)
//	var m sync.Mutex
//	printFn = func(s string, args ...interface{}) (n int, err error) {
//		atomic.AddInt64(&counter, 1)
//		m.Lock()
//		defer m.Unlock()
//		return fmt.Fprintf(&b, s, args...)
//	}
//	workerNum := 4
//	p := &processor{}
//	w := NewHTTPWriter(testConf, testConsistency)
//	p.initWithHTTPWriter(workerNum, w)
//	p.Close(true)
//
//	// Check p was initialized correctly with channels
//	if got := cap(p.backingOffChan); got != backingOffChanCap {
//		t.Errorf("backing off chan cap incorrect: got %d want %d", got, backingOffChanCap)
//	}
//	if got := cap(p.backingOffDone); got != 0 {
//		t.Errorf("backing off done chan cap not 0: got %d", got)
//	}
//
//	// Check p was initialized with correct writer given conf
//	err := testWriterMatchesConfig(p.httpWriter, testConf, testConsistency)
//	if err != nil {
//		t.Error(err)
//	}
//
//	// Check that backoff successfully shut down
//	if got := atomic.LoadInt64(&counter); got != 1 {
//		t.Errorf("printFn called incorrect # of times: got %d want %d", got, 1)
//	}
//	got := string(b.Bytes())
//	if !strings.Contains(got, fmt.Sprintf("worker %d", workerNum)) {
//		t.Errorf("printFn did not contain correct worker number: %s", got)
//	}
//}

func TestProcessorProcessBatch(t *testing.T) {
	bufPool = sync.Pool{
		New: func() interface{} {
			return bytes.NewBuffer(make([]byte, 0, 4*1024*1024))
		},
	}
	f := &factory{}
	b := f.New().(*batch)
	pt := data.LoadedPoint{
		Data: []byte("cpu_usage_guest_nice{1998426147} 1451606400000 38.24311829 hostname host_0 region eu-west-1 datacenter eu-west-1b measurement cpu fieldname usage_guest_nice\n"),
	}
	b.Append(pt)
	b.Len()
	assert.Equal(t, uint(1), b.Len())

	p := &processor{}
	host = "localhost"
	port = 6379
	p.Init(0, true, false)
	mCnt, rCnt := p.ProcessBatch(b, true)

	if mCnt != b.metrics {
		t.Errorf("process batch returned less metrics than batch: got %d want %d", mCnt, b.metrics)
	}
	if rCnt != uint64(b.rows) {
		t.Errorf("process batch returned less rows than batch: got %d want %d", rCnt, b.rows)
	}
	p.Close(true)

	time.Sleep(50 * time.Millisecond)

}
