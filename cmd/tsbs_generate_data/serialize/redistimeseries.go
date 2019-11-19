package serialize

import (
	"crypto/md5"
	"encoding/binary"
	"fmt"
	"io"
)

// RedisTimeSeriesSerializer writes a Point in a serialized form for RedisTimeSeries
type RedisTimeSeriesSerializer struct{}

var keysSoFar map[string]bool

// Serialize writes Point data to the given writer, in a format that will be easy to create a redis-timeseries command
// from.
//
// This function writes output that looks like:
//cpu_usage_user{md5(hostname=host_0|region=eu-central-1...)} 1451606400 58 LABELS hostname host_0 region eu-central-1 ... measurement cpu fieldname usage_user
//
// Which the loader will decode into a set of TS.ADD commands for each fieldKey. Once labels have been created for a each fieldKey,
// subsequent rows are ommitted with them and are ingested with TS.MADD for a row's metrics.
func (s *RedisTimeSeriesSerializer) Serialize(p *Point, w io.Writer) (err error) {
	if keysSoFar == nil {
		keysSoFar = make(map[string]bool)
	}
	// Construct labels text, prefixed with name 'LABELS', following pairs of label names and label values
	// This will be added to each new key, with additional "fieldname" tag
	labels := make([]byte, 0, 256)
	labelsForKeyName := make([]byte, 0, 256)
	labels = append(labels, []byte(" LABELS")...)
	for i, v := range p.tagValues {
		labels = append(labels, ' ')
		labels = append(labels, p.tagKeys[i]...)
		labels = append(labels, ' ')
		labels = fastFormatAppend( v, labels )

		// construct a string of {hostname=host_1,region=us-west-1,...} to be used for unique name for key
		if i > 0 {
			labelsForKeyName = append(labelsForKeyName, '|')
		} else {
			labelsForKeyName = append(labelsForKeyName, '{')
		}
		labelsForKeyName = append(labelsForKeyName, p.tagKeys[i]...)
		labelsForKeyName = append(labelsForKeyName, '=')
		labelsForKeyName = fastFormatAppend( v, labelsForKeyName )

	}

	if len(labelsForKeyName) > 0 {
		labelsForKeyName = append(labelsForKeyName, '}')
	}
	// add measurement name as additional label to be used in queries
	labels = append(labels, []byte(" measurement ")...)
	labels = append(labels, p.measurementName...)

	// Write new line for each fieldKey in the form of: measurementName_fieldName{md5 of labels} timestamp fieldValue LABELS ....
	buf := make([]byte, 0, 256)
	for fieldID := 0; fieldID < len(p.fieldKeys); fieldID++ {
		lbuf := make([]byte, 0, 256)
		fieldName := p.fieldKeys[fieldID]
		fieldValue := p.fieldValues[fieldID]
		keyName := fmt.Sprintf("%s_%s%s", p.measurementName, fieldName, labelsForKeyName)
		// write unique key name
		labelsHash := md5.Sum([]byte(labelsForKeyName))
		lbuf = append(lbuf, p.measurementName...)
		lbuf = append(lbuf, '_')
		lbuf = append(lbuf, fieldName...)

		lbuf = append(lbuf, '{')
		lbuf = fastFormatAppend(int(binary.BigEndian.Uint32(labelsHash[:])), lbuf)
		lbuf = append(lbuf, '}')

		lbuf = append(lbuf, ' ')

		// write timestamp in ms
		lbuf = fastFormatAppend(p.timestamp.UTC().Unix()*1000, lbuf)
		lbuf = append(lbuf, ' ')
		// write value
		lbuf = fastFormatAppend(fieldValue, lbuf)

		// if this key was already inserted and created, we don't to specify the labels again
		if keysSoFar[keyName] {
			lbuf = append(lbuf, ' ')
			buf = append(buf, lbuf...)
			continue
		}
		keysSoFar[keyName] = true
		lbuf = append([]byte("TS.ADD "), lbuf...)
		lbuf = append(lbuf, labels...)
		// additional label of fieldname
		lbuf = append(lbuf, []byte(" fieldname ")...)
		lbuf = fastFormatAppend(fieldName, lbuf)
		lbuf = append(lbuf, '\n')
		buf = append(buf, lbuf...)
	}
	if buf[len(buf)-1] == ' ' {
		buf[len(buf)-1] = '\n'
		buf = append([]byte("TS.MADD "), buf...)
	}
	_, err = w.Write(buf)

	return err
}
