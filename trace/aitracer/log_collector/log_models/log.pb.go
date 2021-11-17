// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: log.proto

package log_models

import (
	fmt "fmt"

	proto "github.com/gogo/protobuf/proto"

	math "math"

	io "io"

	github_com_gogo_protobuf_proto "github.com/gogo/protobuf/proto"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion2 // please upgrade the proto package

type Log struct {
	// 原始日志信息
	Message     []byte `protobuf:"bytes,1,req,name=message" json:"message"`
	Timestamp   int64  `protobuf:"varint,2,req,name=timestamp" json:"timestamp"`
	Hostname    string `protobuf:"bytes,3,req,name=hostname" json:"hostname"`
	FileName    string `protobuf:"bytes,7,opt,name=file_name,json=fileName" json:"file_name"`
	FileLine    int64  `protobuf:"varint,8,opt,name=file_line,json=fileLine" json:"file_line"`
	LogLevel    string `protobuf:"bytes,9,opt,name=log_level,json=logLevel" json:"log_level"`
	TraceId     string `protobuf:"bytes,10,opt,name=trace_id,json=traceId" json:"trace_id"`
	Service     string `protobuf:"bytes,11,opt,name=service" json:"service"`
	Source      string `protobuf:"bytes,12,opt,name=source" json:"source"`
	ContainerId string `protobuf:"bytes,13,opt,name=container_id,json=containerId" json:"container_id"`
}

func (m *Log) Reset()         { *m = Log{} }
func (m *Log) String() string { return proto.CompactTextString(m) }
func (*Log) ProtoMessage()    {}
func (*Log) Descriptor() ([]byte, []int) {
	return fileDescriptor_log_05c2ae4f3496eb6d, []int{0}
}
func (m *Log) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Log) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Log.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalTo(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (dst *Log) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Log.Merge(dst, src)
}
func (m *Log) XXX_Size() int {
	return m.Size()
}
func (m *Log) XXX_DiscardUnknown() {
	xxx_messageInfo_Log.DiscardUnknown(m)
}

var xxx_messageInfo_Log proto.InternalMessageInfo

func (m *Log) GetMessage() []byte {
	if m != nil {
		return m.Message
	}
	return nil
}

func (m *Log) GetTimestamp() int64 {
	if m != nil {
		return m.Timestamp
	}
	return 0
}

func (m *Log) GetHostname() string {
	if m != nil {
		return m.Hostname
	}
	return ""
}

func (m *Log) GetFileName() string {
	if m != nil {
		return m.FileName
	}
	return ""
}

func (m *Log) GetFileLine() int64 {
	if m != nil {
		return m.FileLine
	}
	return 0
}

func (m *Log) GetLogLevel() string {
	if m != nil {
		return m.LogLevel
	}
	return ""
}

func (m *Log) GetTraceId() string {
	if m != nil {
		return m.TraceId
	}
	return ""
}

func (m *Log) GetService() string {
	if m != nil {
		return m.Service
	}
	return ""
}

func (m *Log) GetSource() string {
	if m != nil {
		return m.Source
	}
	return ""
}

func (m *Log) GetContainerId() string {
	if m != nil {
		return m.ContainerId
	}
	return ""
}

func init() {
	proto.RegisterType((*Log)(nil), "log_models.Log")
}
func (m *Log) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalTo(dAtA)
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Log) MarshalTo(dAtA []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	if m.Message != nil {
		dAtA[i] = 0xa
		i++
		i = encodeVarintLog(dAtA, i, uint64(len(m.Message)))
		i += copy(dAtA[i:], m.Message)
	}
	dAtA[i] = 0x10
	i++
	i = encodeVarintLog(dAtA, i, uint64(m.Timestamp))
	dAtA[i] = 0x1a
	i++
	i = encodeVarintLog(dAtA, i, uint64(len(m.Hostname)))
	i += copy(dAtA[i:], m.Hostname)
	dAtA[i] = 0x3a
	i++
	i = encodeVarintLog(dAtA, i, uint64(len(m.FileName)))
	i += copy(dAtA[i:], m.FileName)
	dAtA[i] = 0x40
	i++
	i = encodeVarintLog(dAtA, i, uint64(m.FileLine))
	dAtA[i] = 0x4a
	i++
	i = encodeVarintLog(dAtA, i, uint64(len(m.LogLevel)))
	i += copy(dAtA[i:], m.LogLevel)
	dAtA[i] = 0x52
	i++
	i = encodeVarintLog(dAtA, i, uint64(len(m.TraceId)))
	i += copy(dAtA[i:], m.TraceId)
	dAtA[i] = 0x5a
	i++
	i = encodeVarintLog(dAtA, i, uint64(len(m.Service)))
	i += copy(dAtA[i:], m.Service)
	dAtA[i] = 0x62
	i++
	i = encodeVarintLog(dAtA, i, uint64(len(m.Source)))
	i += copy(dAtA[i:], m.Source)
	dAtA[i] = 0x6a
	i++
	i = encodeVarintLog(dAtA, i, uint64(len(m.ContainerId)))
	i += copy(dAtA[i:], m.ContainerId)
	return i, nil
}

func encodeVarintLog(dAtA []byte, offset int, v uint64) int {
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return offset + 1
}
func (m *Log) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.Message != nil {
		l = len(m.Message)
		n += 1 + l + sovLog(uint64(l))
	}
	n += 1 + sovLog(uint64(m.Timestamp))
	l = len(m.Hostname)
	n += 1 + l + sovLog(uint64(l))
	l = len(m.FileName)
	n += 1 + l + sovLog(uint64(l))
	n += 1 + sovLog(uint64(m.FileLine))
	l = len(m.LogLevel)
	n += 1 + l + sovLog(uint64(l))
	l = len(m.TraceId)
	n += 1 + l + sovLog(uint64(l))
	l = len(m.Service)
	n += 1 + l + sovLog(uint64(l))
	l = len(m.Source)
	n += 1 + l + sovLog(uint64(l))
	l = len(m.ContainerId)
	n += 1 + l + sovLog(uint64(l))
	return n
}

func sovLog(x uint64) (n int) {
	for {
		n++
		x >>= 7
		if x == 0 {
			break
		}
	}
	return n
}
func sozLog(x uint64) (n int) {
	return sovLog(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *Log) Unmarshal(dAtA []byte) error {
	var hasFields [1]uint64
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowLog
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: Log: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Log: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Message", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowLog
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthLog
			}
			postIndex := iNdEx + byteLen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Message = append(m.Message[:0], dAtA[iNdEx:postIndex]...)
			if m.Message == nil {
				m.Message = []byte{}
			}
			iNdEx = postIndex
			hasFields[0] |= uint64(0x00000001)
		case 2:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Timestamp", wireType)
			}
			m.Timestamp = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowLog
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Timestamp |= (int64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			hasFields[0] |= uint64(0x00000002)
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Hostname", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowLog
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= (uint64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthLog
			}
			postIndex := iNdEx + intStringLen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Hostname = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
			hasFields[0] |= uint64(0x00000004)
		case 7:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field FileName", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowLog
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= (uint64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthLog
			}
			postIndex := iNdEx + intStringLen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.FileName = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 8:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field FileLine", wireType)
			}
			m.FileLine = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowLog
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.FileLine |= (int64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 9:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field LogLevel", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowLog
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= (uint64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthLog
			}
			postIndex := iNdEx + intStringLen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.LogLevel = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 10:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field TraceId", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowLog
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= (uint64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthLog
			}
			postIndex := iNdEx + intStringLen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.TraceId = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 11:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Service", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowLog
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= (uint64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthLog
			}
			postIndex := iNdEx + intStringLen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Service = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 12:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Source", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowLog
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= (uint64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthLog
			}
			postIndex := iNdEx + intStringLen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Source = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 13:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ContainerId", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowLog
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= (uint64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthLog
			}
			postIndex := iNdEx + intStringLen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.ContainerId = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipLog(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthLog
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}
	if hasFields[0]&uint64(0x00000001) == 0 {
		return github_com_gogo_protobuf_proto.NewRequiredNotSetError("message")
	}
	if hasFields[0]&uint64(0x00000002) == 0 {
		return github_com_gogo_protobuf_proto.NewRequiredNotSetError("timestamp")
	}
	if hasFields[0]&uint64(0x00000004) == 0 {
		return github_com_gogo_protobuf_proto.NewRequiredNotSetError("hostname")
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func skipLog(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowLog
			}
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowLog
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if dAtA[iNdEx-1] < 0x80 {
					break
				}
			}
			return iNdEx, nil
		case 1:
			iNdEx += 8
			return iNdEx, nil
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowLog
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			iNdEx += length
			if length < 0 {
				return 0, ErrInvalidLengthLog
			}
			return iNdEx, nil
		case 3:
			for {
				var innerWire uint64
				var start int = iNdEx
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return 0, ErrIntOverflowLog
					}
					if iNdEx >= l {
						return 0, io.ErrUnexpectedEOF
					}
					b := dAtA[iNdEx]
					iNdEx++
					innerWire |= (uint64(b) & 0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				innerWireType := int(innerWire & 0x7)
				if innerWireType == 4 {
					break
				}
				next, err := skipLog(dAtA[start:])
				if err != nil {
					return 0, err
				}
				iNdEx = start + next
			}
			return iNdEx, nil
		case 4:
			return iNdEx, nil
		case 5:
			iNdEx += 4
			return iNdEx, nil
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
	}
	panic("unreachable")
}

var (
	ErrInvalidLengthLog = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowLog   = fmt.Errorf("proto: integer overflow")
)

func init() { proto.RegisterFile("log.proto", fileDescriptor_log_05c2ae4f3496eb6d) }

var fileDescriptor_log_05c2ae4f3496eb6d = []byte{
	// 265 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x4c, 0xcf, 0x31, 0x6e, 0x83, 0x30,
	0x14, 0xc6, 0x71, 0x0c, 0x55, 0x81, 0x17, 0xba, 0x78, 0xf2, 0x50, 0x39, 0x6e, 0x96, 0x32, 0xf5,
	0x10, 0xd9, 0x22, 0xa1, 0x0e, 0xbd, 0x00, 0x42, 0xf0, 0x4a, 0x2d, 0x19, 0x1c, 0x61, 0x37, 0xe7,
	0xe8, 0xb1, 0x32, 0x66, 0xec, 0x50, 0x55, 0x15, 0x5c, 0xa4, 0x32, 0x22, 0xc4, 0xeb, 0xff, 0xfb,
	0xc9, 0xd6, 0x83, 0x54, 0xe9, 0xf6, 0xe5, 0x38, 0x68, 0xab, 0x29, 0x28, 0xdd, 0x96, 0x9d, 0x6e,
	0x50, 0x99, 0xdd, 0x4f, 0x08, 0x51, 0xa1, 0x5b, 0xca, 0x21, 0xee, 0xd0, 0x98, 0xaa, 0x45, 0x46,
	0x44, 0x98, 0x67, 0xfb, 0xbb, 0xf3, 0xef, 0x36, 0x78, 0xbb, 0x46, 0xba, 0x83, 0xd4, 0xca, 0x0e,
	0x8d, 0xad, 0xba, 0x23, 0x0b, 0x45, 0x98, 0x47, 0x8b, 0xb8, 0x65, 0x2a, 0x20, 0xf9, 0xd0, 0xc6,
	0xf6, 0x55, 0x87, 0x2c, 0x12, 0x61, 0x9e, 0x2e, 0x64, 0xad, 0xf4, 0x09, 0xd2, 0x77, 0xa9, 0xb0,
	0x9c, 0x49, 0x2c, 0xc8, 0x8d, 0xb8, 0xfc, 0xea, 0x13, 0x25, 0x7b, 0x64, 0x89, 0x20, 0xeb, 0x47,
	0x33, 0x29, 0x64, 0x3f, 0x13, 0x77, 0x81, 0xc2, 0x13, 0x2a, 0x96, 0xfa, 0xaf, 0x28, 0xdd, 0x16,
	0xae, 0xd2, 0x2d, 0x24, 0x76, 0xa8, 0x6a, 0x2c, 0x65, 0xc3, 0xc0, 0x13, 0xf1, 0x5c, 0x0f, 0x8d,
	0xbb, 0xd7, 0xe0, 0x70, 0x92, 0x35, 0xb2, 0x8d, 0xbf, 0x2f, 0x91, 0x3e, 0xc2, 0xbd, 0xd1, 0x9f,
	0x43, 0x8d, 0x2c, 0xf3, 0xe6, 0xa5, 0xd1, 0x67, 0xc8, 0x6a, 0xdd, 0xdb, 0x4a, 0xf6, 0x38, 0xb8,
	0x2f, 0x1e, 0x3c, 0xb3, 0x59, 0x97, 0x43, 0xb3, 0x67, 0xe7, 0x91, 0x93, 0xcb, 0xc8, 0xc9, 0xdf,
	0xc8, 0xc9, 0xd7, 0xc4, 0x83, 0xcb, 0xc4, 0x83, 0xef, 0x89, 0x07, 0xff, 0x01, 0x00, 0x00, 0xff,
	0xff, 0x2b, 0x7f, 0x05, 0x44, 0x90, 0x01, 0x00, 0x00,
}