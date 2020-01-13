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
var hashSoFar map[interface{}][]byte

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

	if hashSoFar == nil {
		hashSoFar = make(map[interface{}][]byte)
	}

	var labelBytes []byte
	var hashExists bool

	//
	if labelBytes, hashExists = hashSoFar[p.tagValues[0]]; hashExists == false {
		//do something here
		bb := fastFormatAppend(p.tagValues[0], []byte{})
		labelsHash := md5.Sum([]byte(bb))
		labelBytes = fastFormatAppend(int(binary.BigEndian.Uint32(labelsHash[:])), []byte{})
		hashSoFar[p.tagValues[0]] = labelBytes
	}

	// Write new line for each fieldKey in the form of: measurementName_fieldName{md5 of labels} timestamp fieldValue LABELS ....
	buf := make([]byte, 0, 256)
	buf = append(buf, []byte("TS.MADD ")...)

	for fieldID := 0; fieldID < len(p.fieldKeys); fieldID++ {

		fieldName := p.fieldKeys[fieldID]
		fieldValue := p.fieldValues[fieldID]

		keyName := fmt.Sprintf("%s_%s%s", p.measurementName, fieldName, labelBytes)

		// if this key was already inserted and created, we don't to specify the labels again
		if keysSoFar[keyName] == false {
			lbuf := make([]byte, 0, 256)
			lbuf = append(lbuf, []byte("TS.CREATE ")...)
			lbuf = appendKeyName(lbuf, p, fieldName, labelBytes)
			lbuf = append(lbuf, []byte("LABELS")...)
			for i, v := range p.tagValues {
				lbuf = append(lbuf, ' ')
				lbuf = append(lbuf, p.tagKeys[i]...)
				lbuf = append(lbuf, ' ')
				lbuf = fastFormatAppend(v, lbuf)
			}

			// add measurement name as additional label to be used in queries
			lbuf = append(lbuf, []byte(" measurement ")...)
			lbuf = append(lbuf, p.measurementName...)
			// additional label of fieldname
			lbuf = append(lbuf, []byte(" fieldname ")...)
			lbuf = fastFormatAppend(fieldName, lbuf)
			lbuf = append(lbuf, '\n')
			_, err = w.Write(lbuf)
			w.Write()
			keysSoFar[keyName] = true
		}

		buf = appendKeyName(buf, p, fieldName, labelBytes)
		buf = appendTS_and_Value(buf, p, fieldValue)
		buf = append(buf, ' ')

	}
	if buf[len(buf)-1] == ' ' {
		buf[len(buf)-1] = '\n'
	}
	_, err = w.Write(buf)

	return err
}

func appendTS_and_Value(lbuf []byte, p *Point, fieldValue interface{}) []byte {
	// write timestamp in ms
	lbuf = fastFormatAppend(p.timestamp.UTC().Unix()*1000, lbuf)
	lbuf = append(lbuf, ' ')
	// write value
	lbuf = fastFormatAppend(fieldValue, lbuf)
	return lbuf
}

func appendKeyName(lbuf []byte, p *Point, fieldName []byte, labelBytes []byte) []byte {
	lbuf = append(lbuf, p.measurementName..., )
	lbuf = append(lbuf, '_')
	lbuf = append(lbuf, fieldName...)

	lbuf = append(lbuf, '{')
	lbuf = append(lbuf, labelBytes...)
	lbuf = append(lbuf, '}', ' ')
	return lbuf
}
