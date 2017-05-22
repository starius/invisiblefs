// Code generated by protoc-gen-go.
// source: db.proto
// DO NOT EDIT!

/*
Package zipkv is a generated protocol buffer package.

It is generated from these files:
	db.proto

It has these top-level messages:
	Location
	PutRecord
	DeleteRecord
	HistoryRecord
	Db
*/
package zipkv

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type Location struct {
	BackendFile int32  `protobuf:"zigzag32,1,opt,name=backend_file,json=backendFile" json:"backend_file,omitempty"`
	Offset      int32  `protobuf:"zigzag32,2,opt,name=offset" json:"offset,omitempty"`
	Size        int32  `protobuf:"zigzag32,3,opt,name=size" json:"size,omitempty"`
	Metadata    []byte `protobuf:"bytes,4,opt,name=metadata,proto3" json:"metadata,omitempty"`
}

func (m *Location) Reset()                    { *m = Location{} }
func (m *Location) String() string            { return proto.CompactTextString(m) }
func (*Location) ProtoMessage()               {}
func (*Location) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

func (m *Location) GetBackendFile() int32 {
	if m != nil {
		return m.BackendFile
	}
	return 0
}

func (m *Location) GetOffset() int32 {
	if m != nil {
		return m.Offset
	}
	return 0
}

func (m *Location) GetSize() int32 {
	if m != nil {
		return m.Size
	}
	return 0
}

func (m *Location) GetMetadata() []byte {
	if m != nil {
		return m.Metadata
	}
	return nil
}

type PutRecord struct {
	Filename string    `protobuf:"bytes,1,opt,name=filename" json:"filename,omitempty"`
	Location *Location `protobuf:"bytes,2,opt,name=location" json:"location,omitempty"`
}

func (m *PutRecord) Reset()                    { *m = PutRecord{} }
func (m *PutRecord) String() string            { return proto.CompactTextString(m) }
func (*PutRecord) ProtoMessage()               {}
func (*PutRecord) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{1} }

func (m *PutRecord) GetFilename() string {
	if m != nil {
		return m.Filename
	}
	return ""
}

func (m *PutRecord) GetLocation() *Location {
	if m != nil {
		return m.Location
	}
	return nil
}

type DeleteRecord struct {
	Filename string `protobuf:"bytes,1,opt,name=filename" json:"filename,omitempty"`
}

func (m *DeleteRecord) Reset()                    { *m = DeleteRecord{} }
func (m *DeleteRecord) String() string            { return proto.CompactTextString(m) }
func (*DeleteRecord) ProtoMessage()               {}
func (*DeleteRecord) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{2} }

func (m *DeleteRecord) GetFilename() string {
	if m != nil {
		return m.Filename
	}
	return ""
}

type HistoryRecord struct {
	// Types that are valid to be assigned to Record:
	//	*HistoryRecord_Put
	//	*HistoryRecord_Delete
	Record isHistoryRecord_Record `protobuf_oneof:"record"`
}

func (m *HistoryRecord) Reset()                    { *m = HistoryRecord{} }
func (m *HistoryRecord) String() string            { return proto.CompactTextString(m) }
func (*HistoryRecord) ProtoMessage()               {}
func (*HistoryRecord) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{3} }

type isHistoryRecord_Record interface {
	isHistoryRecord_Record()
}

type HistoryRecord_Put struct {
	Put *PutRecord `protobuf:"bytes,1,opt,name=put,oneof"`
}
type HistoryRecord_Delete struct {
	Delete *DeleteRecord `protobuf:"bytes,2,opt,name=delete,oneof"`
}

func (*HistoryRecord_Put) isHistoryRecord_Record()    {}
func (*HistoryRecord_Delete) isHistoryRecord_Record() {}

func (m *HistoryRecord) GetRecord() isHistoryRecord_Record {
	if m != nil {
		return m.Record
	}
	return nil
}

func (m *HistoryRecord) GetPut() *PutRecord {
	if x, ok := m.GetRecord().(*HistoryRecord_Put); ok {
		return x.Put
	}
	return nil
}

func (m *HistoryRecord) GetDelete() *DeleteRecord {
	if x, ok := m.GetRecord().(*HistoryRecord_Delete); ok {
		return x.Delete
	}
	return nil
}

// XXX_OneofFuncs is for the internal use of the proto package.
func (*HistoryRecord) XXX_OneofFuncs() (func(msg proto.Message, b *proto.Buffer) error, func(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error), func(msg proto.Message) (n int), []interface{}) {
	return _HistoryRecord_OneofMarshaler, _HistoryRecord_OneofUnmarshaler, _HistoryRecord_OneofSizer, []interface{}{
		(*HistoryRecord_Put)(nil),
		(*HistoryRecord_Delete)(nil),
	}
}

func _HistoryRecord_OneofMarshaler(msg proto.Message, b *proto.Buffer) error {
	m := msg.(*HistoryRecord)
	// record
	switch x := m.Record.(type) {
	case *HistoryRecord_Put:
		b.EncodeVarint(1<<3 | proto.WireBytes)
		if err := b.EncodeMessage(x.Put); err != nil {
			return err
		}
	case *HistoryRecord_Delete:
		b.EncodeVarint(2<<3 | proto.WireBytes)
		if err := b.EncodeMessage(x.Delete); err != nil {
			return err
		}
	case nil:
	default:
		return fmt.Errorf("HistoryRecord.Record has unexpected type %T", x)
	}
	return nil
}

func _HistoryRecord_OneofUnmarshaler(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error) {
	m := msg.(*HistoryRecord)
	switch tag {
	case 1: // record.put
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		msg := new(PutRecord)
		err := b.DecodeMessage(msg)
		m.Record = &HistoryRecord_Put{msg}
		return true, err
	case 2: // record.delete
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		msg := new(DeleteRecord)
		err := b.DecodeMessage(msg)
		m.Record = &HistoryRecord_Delete{msg}
		return true, err
	default:
		return false, nil
	}
}

func _HistoryRecord_OneofSizer(msg proto.Message) (n int) {
	m := msg.(*HistoryRecord)
	// record
	switch x := m.Record.(type) {
	case *HistoryRecord_Put:
		s := proto.Size(x.Put)
		n += proto.SizeVarint(1<<3 | proto.WireBytes)
		n += proto.SizeVarint(uint64(s))
		n += s
	case *HistoryRecord_Delete:
		s := proto.Size(x.Delete)
		n += proto.SizeVarint(2<<3 | proto.WireBytes)
		n += proto.SizeVarint(uint64(s))
		n += s
	case nil:
	default:
		panic(fmt.Sprintf("proto: unexpected type %T in oneof", x))
	}
	return n
}

type Db struct {
	NextBackendFile int32            `protobuf:"zigzag32,2,opt,name=next_backend_file,json=nextBackendFile" json:"next_backend_file,omitempty"`
	History         []*HistoryRecord `protobuf:"bytes,3,rep,name=history" json:"history,omitempty"`
}

func (m *Db) Reset()                    { *m = Db{} }
func (m *Db) String() string            { return proto.CompactTextString(m) }
func (*Db) ProtoMessage()               {}
func (*Db) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{4} }

func (m *Db) GetNextBackendFile() int32 {
	if m != nil {
		return m.NextBackendFile
	}
	return 0
}

func (m *Db) GetHistory() []*HistoryRecord {
	if m != nil {
		return m.History
	}
	return nil
}

func init() {
	proto.RegisterType((*Location)(nil), "zipkv.Location")
	proto.RegisterType((*PutRecord)(nil), "zipkv.PutRecord")
	proto.RegisterType((*DeleteRecord)(nil), "zipkv.DeleteRecord")
	proto.RegisterType((*HistoryRecord)(nil), "zipkv.HistoryRecord")
	proto.RegisterType((*Db)(nil), "zipkv.Db")
}

func init() { proto.RegisterFile("db.proto", fileDescriptor0) }

var fileDescriptor0 = []byte{
	// 291 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x84, 0x91, 0xc1, 0x4e, 0x83, 0x40,
	0x10, 0x86, 0x4b, 0xa9, 0x48, 0x07, 0x4c, 0xed, 0x6a, 0x0c, 0xf1, 0x84, 0xc4, 0x03, 0xa9, 0x91,
	0x03, 0xbe, 0x41, 0xd3, 0x18, 0x0e, 0x1e, 0xcc, 0xc6, 0x7b, 0x5d, 0x60, 0x88, 0x9b, 0x52, 0x96,
	0xc0, 0x62, 0xb4, 0x4f, 0x6f, 0x58, 0xb6, 0xa4, 0x3d, 0x79, 0x63, 0x66, 0xfe, 0xf0, 0x7d, 0xf9,
	0x17, 0xec, 0x3c, 0x8d, 0xea, 0x46, 0x48, 0x41, 0x2e, 0x0e, 0xbc, 0xde, 0x7d, 0x07, 0x1d, 0xd8,
	0x6f, 0x22, 0x63, 0x92, 0x8b, 0x8a, 0x3c, 0x80, 0x9b, 0xb2, 0x6c, 0x87, 0x55, 0xbe, 0x2d, 0x78,
	0x89, 0x9e, 0xe1, 0x1b, 0xe1, 0x92, 0x3a, 0x7a, 0xf7, 0xca, 0x4b, 0x24, 0x77, 0x60, 0x89, 0xa2,
	0x68, 0x51, 0x7a, 0x53, 0x75, 0xd4, 0x13, 0x21, 0x30, 0x6b, 0xf9, 0x01, 0x3d, 0x53, 0x6d, 0xd5,
	0x37, 0xb9, 0x07, 0x7b, 0x8f, 0x92, 0xe5, 0x4c, 0x32, 0x6f, 0xe6, 0x1b, 0xa1, 0x4b, 0xc7, 0x39,
	0xf8, 0x80, 0xf9, 0x7b, 0x27, 0x29, 0x66, 0xa2, 0xc9, 0xfb, 0x60, 0xcf, 0xab, 0xd8, 0x7e, 0x60,
	0xce, 0xe9, 0x38, 0x93, 0x27, 0xb0, 0x4b, 0xed, 0xa7, 0x90, 0x4e, 0xbc, 0x88, 0x94, 0x79, 0x74,
	0xd4, 0xa6, 0x63, 0x20, 0x58, 0x81, 0xbb, 0xc1, 0x12, 0x25, 0xfe, 0xff, 0xe3, 0xa0, 0x82, 0xab,
	0x84, 0xb7, 0x52, 0x34, 0xbf, 0x3a, 0xfc, 0x08, 0x66, 0xdd, 0x49, 0x95, 0x73, 0xe2, 0x6b, 0x0d,
	0x19, 0x25, 0x93, 0x09, 0xed, 0xcf, 0xe4, 0x19, 0xac, 0x5c, 0x21, 0xb4, 0xcd, 0x8d, 0x0e, 0x9e,
	0x72, 0x93, 0x09, 0xd5, 0xa1, 0xb5, 0x0d, 0x56, 0xa3, 0x76, 0xc1, 0x27, 0x4c, 0x37, 0x29, 0x59,
	0xc1, 0xb2, 0xc2, 0x1f, 0xb9, 0x3d, 0xeb, 0x79, 0xa8, 0x72, 0xd1, 0x1f, 0xd6, 0x27, 0x5d, 0x47,
	0x70, 0xf9, 0x35, 0x18, 0x7a, 0xa6, 0x6f, 0x86, 0x4e, 0x7c, 0xab, 0x59, 0x67, 0xde, 0xf4, 0x18,
	0x4a, 0x2d, 0xf5, 0xb0, 0x2f, 0x7f, 0x01, 0x00, 0x00, 0xff, 0xff, 0xee, 0x65, 0x76, 0x40, 0xe4,
	0x01, 0x00, 0x00,
}
