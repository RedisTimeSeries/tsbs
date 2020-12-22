package main

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"fmt"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/timescale/tsbs/internal/utils"
	"github.com/timescale/tsbs/load"
	"github.com/timescale/tsbs/pkg/targets"
	"github.com/timescale/tsbs/pkg/targets/constants"
	"github.com/timescale/tsbs/pkg/targets/initializers"
	"log"
	"sync"
)

// Global vars
var (
	loader  load.BenchmarkRunner
	config  load.BenchmarkRunnerConfig
	bufPool sync.Pool
	target  targets.ImplementedTarget
)

// Program option vars:
var (
	host               string
	pipeline           uint64
	port               uint64
	compressionEnabled bool
	cluster            bool
	password           string
)

// allows for testing
var fatal = log.Fatal
var md5h = md5.New()

// Parse args:
func init() {
	target = initializers.GetTarget(constants.FormatRedisTimeSeries)
	config = load.BenchmarkRunnerConfig{}
	config.AddToFlagSet(pflag.CommandLine)
	target.TargetSpecificFlags("", pflag.CommandLine)
	pflag.Parse()

	err := utils.SetupConfigFile()

	if err != nil {
		panic(fmt.Errorf("fatal error config file: %s", err))
	}

	if err := viper.Unmarshal(&config); err != nil {
		panic(fmt.Errorf("unable to decode config: %s", err))
	}
	host = viper.GetString("host")
	port = viper.GetUint64("port")
	pipeline = viper.GetUint64("pipeline")
	compressionEnabled = viper.GetBool("compression-enabled")
	cluster = viper.GetBool("cluster")

	if err := viper.Unmarshal(&config); err != nil {
		panic(fmt.Errorf("unable to decode config: %s", err))
	}
	config.HashWorkers = false
	loader = load.GetBenchmarkRunner(config)
}

type benchmark struct{}

func (b *benchmark) GetDataSource() targets.DataSource {
	return &fileDataSource{scanner: bufio.NewScanner(load.GetBufferedReader(config.FileName))}
}

func (b *benchmark) GetBatchFactory() targets.BatchFactory {
	return &factory{}
}

func (b *benchmark) GetPointIndexer(_ uint) targets.PointIndexer {
	return &targets.ConstantIndexer{}
}

func (b *benchmark) GetProcessor() targets.Processor {
	return &processor{}
}

func (b *benchmark) GetDBCreator() targets.DBCreator {
	return &dbCreator{}
}

func main() {
	bufPool = sync.Pool{
		New: func() interface{} {
			return bytes.NewBuffer(make([]byte, 0, 4*1024*1024))
		},
	}

	loader.RunBenchmark(&benchmark{})
}
