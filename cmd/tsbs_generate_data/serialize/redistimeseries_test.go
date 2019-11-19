package serialize
import (
	"testing"
)

func TestRedisTimeSeriesSerializer(t *testing.T) {
	cases := []serializeCase{
		{
			desc:       "a regular Point",
			inputPoint: testPointDefault,
			output:     "cpu_usage_guest_nice{hostname=host_0|region=eu-west-1|datacenter=eu-west-1b} 1451606400 38.24311829 LABELS hostname host_0 region eu-west-1 datacenter eu-west-1b measurement cpu fieldname usage_guest_nice\n",
		},
		{
			desc:       "a regular Point using int as value",
			inputPoint: testPointInt,
			output:     "cpu_usage_guest{hostname=host_0|region=eu-west-1|datacenter=eu-west-1b} 1451606400 38 LABELS hostname host_0 region eu-west-1 datacenter eu-west-1b measurement cpu fieldname usage_guest\n",
		},
		{
			desc:       "a regular Point with multiple fields",
			inputPoint: testPointMultiField,
			output:     "cpu_big_usage_guest{hostname=host_0|region=eu-west-1|datacenter=eu-west-1b} 1451606400 5000000000 LABELS hostname host_0 region eu-west-1 datacenter eu-west-1b measurement cpu fieldname big_usage_guest\n" +
						"cpu_usage_guest{hostname=host_0|region=eu-west-1|datacenter=eu-west-1b} 1451606400 38 LABELS hostname host_0 region eu-west-1 datacenter eu-west-1b measurement cpu fieldname usage_guest\n" +
						"cpu_usage_guest_nice{hostname=host_0|region=eu-west-1|datacenter=eu-west-1b} 1451606400 38.24311829 LABELS hostname host_0 region eu-west-1 datacenter eu-west-1b measurement cpu fieldname usage_guest_nice\n",
		},
		{
			desc:       "a Point with no tags",
			inputPoint: testPointNoTags,
			output:     "cpu_usage_guest_nice 1451606400 38.24311829 LABELS measurement cpu fieldname usage_guest_nice\n",
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
