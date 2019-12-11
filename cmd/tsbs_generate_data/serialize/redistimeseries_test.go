package serialize
import (
	"testing"
)

func TestRedisTimeSeriesSerializer(t *testing.T) {
	cases := []serializeCase{
		{
			desc:       "a regular Point",
			inputPoint: testPointDefault,
			output:     "TS.ADD cpu_usage_guest_nice{3840552926} 1451606400000 38.24311829 LABELS hostname host_0 region eu-west-1 datacenter eu-west-1b measurement cpu fieldname usage_guest_nice\n",
		},
		{
			desc:       "a regular Point using int as value",
			inputPoint: testPointInt,
			output:     "TS.ADD cpu_usage_guest{3840552926} 1451606400000 38 LABELS hostname host_0 region eu-west-1 datacenter eu-west-1b measurement cpu fieldname usage_guest\n",
		},
		{
			desc:       "a Point with no tags",
			inputPoint: testPointNoTags,
			output:     "TS.ADD cpu_usage_guest_nice{3558706393} 1451606400000 38.24311829 LABELS measurement cpu fieldname usage_guest_nice\n",
		},
	}

	testSerializer(t, cases, &RedisTimeSeriesSerializer{})
}

func TestRedisTimeSeriesSerializerErr(t *testing.T) {
	p := testPointMultiField
	s := &RedisTimeSeriesSerializer{}
	err := s.Serialize(p, &errWriter{})
	if err == nil {
		t.Errorf("no error returned when expected")
	} else if err.Error() != errWriterAlwaysErr {
		t.Errorf("unexpected writer error: %v", err)
	}
}
