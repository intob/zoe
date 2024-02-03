// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.32.0
// 	protoc        v4.25.1
// source: ev.proto

package ev

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type EvType int32

const (
	EvType_LOAD   EvType = 0 // Once per page load per session
	EvType_UNLOAD EvType = 1 // Once per page unload per session
	EvType_TIME   EvType = 2 // Time spent on a page
)

// Enum value maps for EvType.
var (
	EvType_name = map[int32]string{
		0: "LOAD",
		1: "UNLOAD",
		2: "TIME",
	}
	EvType_value = map[string]int32{
		"LOAD":   0,
		"UNLOAD": 1,
		"TIME":   2,
	}
)

func (x EvType) Enum() *EvType {
	p := new(EvType)
	*p = x
	return p
}

func (x EvType) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (EvType) Descriptor() protoreflect.EnumDescriptor {
	return file_ev_proto_enumTypes[0].Descriptor()
}

func (EvType) Type() protoreflect.EnumType {
	return &file_ev_proto_enumTypes[0]
}

func (x EvType) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use EvType.Descriptor instead.
func (EvType) EnumDescriptor() ([]byte, []int) {
	return file_ev_proto_rawDescGZIP(), []int{0}
}

// Ev represents a tracking event.
// As there are millions, we must aim for
// optimum use of space.
// User & session ids are fixed-length
// because we use a hash function
// to ensure uniform & complete distribution.
type Ev struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	EvType      EvType   `protobuf:"varint,1,opt,name=evType,proto3,enum=EvType" json:"evType,omitempty"`
	Time        uint32   `protobuf:"varint,2,opt,name=time,proto3" json:"time,omitempty"` // good until year 2106
	Usr         uint32   `protobuf:"fixed32,3,opt,name=usr,proto3" json:"usr,omitempty"`
	Sess        uint32   `protobuf:"fixed32,4,opt,name=sess,proto3" json:"sess,omitempty"`
	Cid         uint32   `protobuf:"varint,5,opt,name=cid,proto3" json:"cid,omitempty"`
	PageSeconds *uint32  `protobuf:"varint,6,opt,name=pageSeconds,proto3,oneof" json:"pageSeconds,omitempty"`
	Scrolled    *float32 `protobuf:"fixed32,7,opt,name=scrolled,proto3,oneof" json:"scrolled,omitempty"`
}

func (x *Ev) Reset() {
	*x = Ev{}
	if protoimpl.UnsafeEnabled {
		mi := &file_ev_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Ev) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Ev) ProtoMessage() {}

func (x *Ev) ProtoReflect() protoreflect.Message {
	mi := &file_ev_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Ev.ProtoReflect.Descriptor instead.
func (*Ev) Descriptor() ([]byte, []int) {
	return file_ev_proto_rawDescGZIP(), []int{0}
}

func (x *Ev) GetEvType() EvType {
	if x != nil {
		return x.EvType
	}
	return EvType_LOAD
}

func (x *Ev) GetTime() uint32 {
	if x != nil {
		return x.Time
	}
	return 0
}

func (x *Ev) GetUsr() uint32 {
	if x != nil {
		return x.Usr
	}
	return 0
}

func (x *Ev) GetSess() uint32 {
	if x != nil {
		return x.Sess
	}
	return 0
}

func (x *Ev) GetCid() uint32 {
	if x != nil {
		return x.Cid
	}
	return 0
}

func (x *Ev) GetPageSeconds() uint32 {
	if x != nil && x.PageSeconds != nil {
		return *x.PageSeconds
	}
	return 0
}

func (x *Ev) GetScrolled() float32 {
	if x != nil && x.Scrolled != nil {
		return *x.Scrolled
	}
	return 0
}

var File_ev_proto protoreflect.FileDescriptor

var file_ev_proto_rawDesc = []byte{
	0x0a, 0x08, 0x65, 0x76, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0xd6, 0x01, 0x0a, 0x02, 0x45,
	0x76, 0x12, 0x1f, 0x0a, 0x06, 0x65, 0x76, 0x54, 0x79, 0x70, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x0e, 0x32, 0x07, 0x2e, 0x45, 0x76, 0x54, 0x79, 0x70, 0x65, 0x52, 0x06, 0x65, 0x76, 0x54, 0x79,
	0x70, 0x65, 0x12, 0x12, 0x0a, 0x04, 0x74, 0x69, 0x6d, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0d,
	0x52, 0x04, 0x74, 0x69, 0x6d, 0x65, 0x12, 0x10, 0x0a, 0x03, 0x75, 0x73, 0x72, 0x18, 0x03, 0x20,
	0x01, 0x28, 0x07, 0x52, 0x03, 0x75, 0x73, 0x72, 0x12, 0x12, 0x0a, 0x04, 0x73, 0x65, 0x73, 0x73,
	0x18, 0x04, 0x20, 0x01, 0x28, 0x07, 0x52, 0x04, 0x73, 0x65, 0x73, 0x73, 0x12, 0x10, 0x0a, 0x03,
	0x63, 0x69, 0x64, 0x18, 0x05, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x03, 0x63, 0x69, 0x64, 0x12, 0x25,
	0x0a, 0x0b, 0x70, 0x61, 0x67, 0x65, 0x53, 0x65, 0x63, 0x6f, 0x6e, 0x64, 0x73, 0x18, 0x06, 0x20,
	0x01, 0x28, 0x0d, 0x48, 0x00, 0x52, 0x0b, 0x70, 0x61, 0x67, 0x65, 0x53, 0x65, 0x63, 0x6f, 0x6e,
	0x64, 0x73, 0x88, 0x01, 0x01, 0x12, 0x1f, 0x0a, 0x08, 0x73, 0x63, 0x72, 0x6f, 0x6c, 0x6c, 0x65,
	0x64, 0x18, 0x07, 0x20, 0x01, 0x28, 0x02, 0x48, 0x01, 0x52, 0x08, 0x73, 0x63, 0x72, 0x6f, 0x6c,
	0x6c, 0x65, 0x64, 0x88, 0x01, 0x01, 0x42, 0x0e, 0x0a, 0x0c, 0x5f, 0x70, 0x61, 0x67, 0x65, 0x53,
	0x65, 0x63, 0x6f, 0x6e, 0x64, 0x73, 0x42, 0x0b, 0x0a, 0x09, 0x5f, 0x73, 0x63, 0x72, 0x6f, 0x6c,
	0x6c, 0x65, 0x64, 0x2a, 0x28, 0x0a, 0x06, 0x45, 0x76, 0x54, 0x79, 0x70, 0x65, 0x12, 0x08, 0x0a,
	0x04, 0x4c, 0x4f, 0x41, 0x44, 0x10, 0x00, 0x12, 0x0a, 0x0a, 0x06, 0x55, 0x4e, 0x4c, 0x4f, 0x41,
	0x44, 0x10, 0x01, 0x12, 0x08, 0x0a, 0x04, 0x54, 0x49, 0x4d, 0x45, 0x10, 0x02, 0x42, 0x06, 0x5a,
	0x04, 0x2e, 0x2f, 0x65, 0x76, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_ev_proto_rawDescOnce sync.Once
	file_ev_proto_rawDescData = file_ev_proto_rawDesc
)

func file_ev_proto_rawDescGZIP() []byte {
	file_ev_proto_rawDescOnce.Do(func() {
		file_ev_proto_rawDescData = protoimpl.X.CompressGZIP(file_ev_proto_rawDescData)
	})
	return file_ev_proto_rawDescData
}

var file_ev_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_ev_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_ev_proto_goTypes = []interface{}{
	(EvType)(0), // 0: EvType
	(*Ev)(nil),  // 1: Ev
}
var file_ev_proto_depIdxs = []int32{
	0, // 0: Ev.evType:type_name -> EvType
	1, // [1:1] is the sub-list for method output_type
	1, // [1:1] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_ev_proto_init() }
func file_ev_proto_init() {
	if File_ev_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_ev_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Ev); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	file_ev_proto_msgTypes[0].OneofWrappers = []interface{}{}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_ev_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_ev_proto_goTypes,
		DependencyIndexes: file_ev_proto_depIdxs,
		EnumInfos:         file_ev_proto_enumTypes,
		MessageInfos:      file_ev_proto_msgTypes,
	}.Build()
	File_ev_proto = out.File
	file_ev_proto_rawDesc = nil
	file_ev_proto_goTypes = nil
	file_ev_proto_depIdxs = nil
}