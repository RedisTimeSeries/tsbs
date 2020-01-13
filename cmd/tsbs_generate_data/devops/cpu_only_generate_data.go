package devops

import (
	"fmt"
	"time"

	"github.com/timescale/tsbs/cmd/tsbs_generate_data/common"
	"github.com/timescale/tsbs/cmd/tsbs_generate_data/serialize"
)

// A CPUOnlySimulator generates data similar to telemetry from Telegraf for only CPU metrics.
// It fulfills the Simulator interface.
type CPUOnlySimulator struct {
	base            CPUOnlySimulatorConfig
	*commonDevopsSimulator
}

// Fields returns a map of subsystems to metrics collected
func (d *CPUOnlySimulator) Fields() map[string][][]byte {
	return d.fields(d.hosts[0].SimulatedMeasurements[:1])
}


// GetSummary returns a summary string after data has been generated.
func (s *CPUOnlySimulator) GetSummary() string {
	return fmt.Sprintf("CPUOnlySimulator. Generated a total of %d points. \n\tStart time for the Simulator %s\n\tEnd time for the Simulator%s\n\tNumber of hosts to start with in the first reporting period %d\n\tNumber of hosts to have in the last reporting period %d\n\tHostConstructor(function used to create a new Host given an id number and start time) %s\n\tNumber of epochs %d per host.", s.madePoints, s.timestampStart.String(), s.timestampEnd.String(), s.initHosts , s.epochHosts, "", s.epochs)
}

// Next advances a Point to the next state in the generator.
func (d *CPUOnlySimulator) Next(p *serialize.Point) bool {
	// Switch to the next metric if needed
	if d.hostIndex == uint64(len(d.hosts)) {
		d.hostIndex = 0

		for i := 0; i < len(d.hosts); i++ {
			d.hosts[i].TickAll(d.interval)
		}

		d.adjustNumHostsForEpoch()
	}

	return d.populatePoint(p, 0)
}

func (s *CPUOnlySimulator) MaxPoints() uint64 {
	return s.maxPoints
}


// CPUOnlySimulatorConfig is used to create a CPUOnlySimulator.
type CPUOnlySimulatorConfig commonDevopsSimulatorConfig

// NewSimulator produces a Simulator that conforms to the given SimulatorConfig over the specified interval
func (c *CPUOnlySimulatorConfig) NewSimulator(interval time.Duration, limit uint64) common.Simulator {
	hostInfos := make([]Host, c.HostCount)
	for i := 0; i < len(hostInfos); i++ {
		hostInfos[i] = c.HostConstructor(i, c.Start)
	}

	epochs := calculateEpochs(commonDevopsSimulatorConfig(*c), interval)
	maxPoints := epochs * c.HostCount
	if limit > 0 && limit < maxPoints {
		// Set specified points number limit
		maxPoints = limit
	}
	sim := &CPUOnlySimulator{
		*c,
		&commonDevopsSimulator{
		madePoints: 0,
		maxPoints:  maxPoints,

		hostIndex: 0,
		hosts:     hostInfos,

		epoch:          0,
		epochs:         epochs,
		epochHosts:     c.InitHostCount,
		initHosts:      c.InitHostCount,
		timestampStart: c.Start,
		timestampEnd:   c.End,
		interval:       interval,
	}}

	return sim
}
