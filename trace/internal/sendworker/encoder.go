package sendworker

import (
	"bytes"
	"encoding/binary"
)

const (
	ConnectionMagicNum    = uint16(31843)
	Version1              = uint8(1)
	CommonPrefixLen       = 6  // magicNum 2byte + Length 4Byte
	Version1DataPrefixLen = 13 //prefix len of version 1, not including magicNum and Length filed
)

// Encode [[magicNum,Length],[codec_version,body_offset,body_length,header_length,header],[body]]
func Encode(body []byte, headerBytes []byte) []byte {
	bodyOffset := Version1DataPrefixLen + len(headerBytes) // body's offset with respect to codec_version
	dataLen := bodyOffset + len(body)                      // data length, not including magicNum and length filed
	totalLen := CommonPrefixLen + dataLen                  // total length, including magicNum length filed

	// payload to be sent. here we try to reuse body's underlay slice to avoid memory allocate
	var (
		payload []byte
	)

	if cap(body) < totalLen { //case1: body's capacity less than totalLen, we need to grow. For better performance, body should have larger cap than totalLen.
		payload = make([]byte, totalLen)
	} else { // case2: body's cap is sufficient, we can use body as payload directly
		payload = body[0:totalLen]
	}

	// body. we must write body first. if not, in case2, memory of small index will be overwritten by Prefix
	copy(payload[totalLen-len(body):], body)

	// CommonPrefix
	binary.LittleEndian.PutUint16(payload[0:2], ConnectionMagicNum)
	binary.LittleEndian.PutUint32(payload[2:6], uint32(dataLen))

	// version
	payload[6] = Version1

	// bodyOffset+bodyLen
	binary.LittleEndian.PutUint32(payload[7:11], uint32(bodyOffset))
	binary.LittleEndian.PutUint32(payload[11:15], uint32(len(body)))

	// header
	binary.LittleEndian.PutUint32(payload[15:19], uint32(len(headerBytes)))
	copy(payload[19:19+len(headerBytes)], headerBytes)

	return payload
}

// EncodePreAllocated [[magicNum,Length],[codec_version,body_offset,body_length,header_length,header],[body]]
func EncodePreAllocated(payload []byte, headerBytes []byte) []byte {
	totalPrefixLen := GetPrefixLen(1, headerBytes)
	if len(payload) < totalPrefixLen {
		return payload
	}
	bodyOffset := Version1DataPrefixLen + len(headerBytes) // body's offset with respect to codec_version
	bodyLen := len(payload) - totalPrefixLen               // body's length
	dataLen := len(payload) - CommonPrefixLen              // data sector length, not including magicNum and length filed

	// CommonPrefix
	binary.LittleEndian.PutUint16(payload[0:2], ConnectionMagicNum)
	binary.LittleEndian.PutUint32(payload[2:6], uint32(dataLen))

	// version
	payload[6] = Version1

	// bodyOffset+bodyLen
	binary.LittleEndian.PutUint32(payload[7:11], uint32(bodyOffset))
	binary.LittleEndian.PutUint32(payload[11:15], uint32(bodyLen))

	// header
	binary.LittleEndian.PutUint32(payload[15:19], uint32(len(headerBytes)))
	copy(payload[19:19+len(headerBytes)], headerBytes)

	return payload
}

func FormatMap(m map[string]string) []byte {
	b := bytes.NewBuffer(nil)
	for k, v := range m {
		if len(k) == 0 || len(v) == 0 {
			continue
		}
		kLen, vLen := make([]byte, 4), make([]byte, 4)
		binary.LittleEndian.PutUint32(kLen, uint32(len(k)))
		binary.LittleEndian.PutUint32(vLen, uint32(len(v)))
		b.Write(kLen)
		b.WriteString(k)
		b.Write(vLen)
		b.WriteString(v)
	}
	return b.Bytes()
}

func GetPrefixLen(version uint8, headerBytes []byte) int {
	if version == Version1 {
		return CommonPrefixLen + Version1DataPrefixLen + len(headerBytes)
	}
	return -1
}
