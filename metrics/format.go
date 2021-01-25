package metrics

import (
	"bytes"
	"encoding/binary"
	"errors"
	"math"
)

const (
	maxStringLen = 255
)

var (
	mtCounter = uint8(1)
	mtTimer   = uint8(2)
	mtGauge   = uint8(3)
)

func formatCommon(buf *bytes.Buffer, mt uint8, prefix string, name string, value float64, tags []t) error {
	buf.WriteByte(byte(mt))
	err := writeName(buf, prefix, name)
	if err != nil {
		return err
	}
	err = writeFloat64(buf, value)
	if err != nil {
		return err
	}
	err = writeTags(buf, tags)
	if err != nil {
		return err
	}
	return nil
}

func writeName(buf *bytes.Buffer, prefix string, name string) error {
	if prefix != "" {
		length := len(prefix) + len(name) + 1
		if length > maxStringLen {
			return errors.New("<prefix>.<name> must smaller than 256")
		}
		buf.WriteByte(uint8(length))
		buf.WriteString(prefix)
		buf.WriteByte('.')
		buf.WriteString(name)
		return nil
	} else {
		return writeString(buf, name)
	}
}

func writeTags(buf *bytes.Buffer, tags []t) error {
	if len(tags) > maxStringLen {
		return errors.New("tag map must smaller than 256")
	}
	buf.WriteByte(uint8(len(tags)))
	for _, tag := range tags {
		err := writeString(buf, tag.key)
		if err != nil {
			return err
		}
		err = writeString(buf, tag.value)
		if err != nil {
			return err
		}
	}
	return nil
}

func writeString(buf *bytes.Buffer, name string) error {
	if len(name) > maxStringLen {
		return errors.New("string length must less than 256")
	}
	buf.WriteByte(uint8(len(name)))
	buf.WriteString(name)
	return nil
}

func writeFloat64(buf *bytes.Buffer, value float64) error {
	vv := [8]byte{}
	binary.LittleEndian.PutUint64(vv[:], math.Float64bits(value))
	buf.Write(vv[:])
	return nil
}
