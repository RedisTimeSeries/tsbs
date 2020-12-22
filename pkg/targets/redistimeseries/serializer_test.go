package redistimeseries

import (
	"github.com/timescale/tsbs/pkg/data/serialize"
	"testing"
)

func TestInfluxSerializerSerialize(t *testing.T) {
	cases := []serialize.SerializeCase{
		{
			Desc:       "a regular Point",
			InputPoint: serialize.TestPointDefault(),
			Output:     "cpu_usage_guest_nice{1998426147} 1451606400000 38.24311829 hostname host_0 region eu-west-1 datacenter eu-west-1b measurement cpu fieldname usage_guest_nice\n",
		},
		{
			Desc:       "a regular Point using int as value",
			InputPoint: serialize.TestPointInt(),
			Output:     "cpu_usage_guest{1998426147} 1451606400000 38 hostname host_0 region eu-west-1 datacenter eu-west-1b measurement cpu fieldname usage_guest\n",
		},
		{
			Desc:       "a regular Point with multiple fields",
			InputPoint: serialize.TestPointMultiField(),
			Output:     "cpu_big_usage_guest{1998426147} 1451606400000 5000000000 hostname host_0 region eu-west-1 datacenter eu-west-1b measurement cpu fieldname big_usage_guest\ncpu_usage_guest{1998426147} 1451606400000 38 hostname host_0 region eu-west-1 datacenter eu-west-1b measurement cpu fieldname usage_guest\ncpu_usage_guest_nice{1998426147} 1451606400000 38.24311829 hostname host_0 region eu-west-1 datacenter eu-west-1b measurement cpu fieldname usage_guest_nice\n",
		},
		{
			Desc:       "a Point with no tags",
			InputPoint: serialize.TestPointNoTags(),
			Output:     "cpu_usage_guest_nice{2761257603} 1451606400000 38.24311829 measurement cpu fieldname usage_guest_nice\n",
		},
		{
			Desc:       "a Point with a nil tag",
			InputPoint: serialize.TestPointWithNilTag(),
			Output:     "cpu_usage_guest_nice{2761257603} 1451606400000 38.24311829 hostname  measurement cpu fieldname usage_guest_nice\n",
		},
		{
			Desc:       "a Point with a nil field",
			InputPoint: serialize.TestPointWithNilField(),
			Output:     "cpu_big_usage_guest{2761257603} 1451606400000  measurement cpu fieldname big_usage_guest\ncpu_usage_guest_nice{2761257603} 1451606400000 38.24311829 measurement cpu fieldname usage_guest_nice\n",
		},
	}

	serialize.SerializerTest(t, cases, &Serializer{})
}
