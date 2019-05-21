// Code generated by protoc-gen-go. DO NOT EDIT.
// source: login.proto

package pb

import (
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	any "github.com/golang/protobuf/ptypes/any"
	math "math"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion3 // please upgrade the proto package

//登录返回值
type LoginResponse struct {
	//登录结果
	Result bool `protobuf:"varint,1,opt,name=result,proto3" json:"result,omitempty"`
	//用户名
	UserId string `protobuf:"bytes,2,opt,name=user_id,json=userId,proto3" json:"user_id,omitempty"`
	//自定义参数
	Params               *any.Any `protobuf:"bytes,3,opt,name=params,proto3" json:"params,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *LoginResponse) Reset()         { *m = LoginResponse{} }
func (m *LoginResponse) String() string { return proto.CompactTextString(m) }
func (*LoginResponse) ProtoMessage()    {}
func (*LoginResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_67c21677aa7f4e4f, []int{0}
}

func (m *LoginResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_LoginResponse.Unmarshal(m, b)
}
func (m *LoginResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_LoginResponse.Marshal(b, m, deterministic)
}
func (m *LoginResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_LoginResponse.Merge(m, src)
}
func (m *LoginResponse) XXX_Size() int {
	return xxx_messageInfo_LoginResponse.Size(m)
}
func (m *LoginResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_LoginResponse.DiscardUnknown(m)
}

var xxx_messageInfo_LoginResponse proto.InternalMessageInfo

func (m *LoginResponse) GetResult() bool {
	if m != nil {
		return m.Result
	}
	return false
}

func (m *LoginResponse) GetUserId() string {
	if m != nil {
		return m.UserId
	}
	return ""
}

func (m *LoginResponse) GetParams() *any.Any {
	if m != nil {
		return m.Params
	}
	return nil
}

func init() {
	proto.RegisterType((*LoginResponse)(nil), "user.LoginResponse")
}

func init() { proto.RegisterFile("login.proto", fileDescriptor_67c21677aa7f4e4f) }

var fileDescriptor_67c21677aa7f4e4f = []byte{
	// 154 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0xe2, 0xce, 0xc9, 0x4f, 0xcf,
	0xcc, 0xd3, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0x62, 0x29, 0x2d, 0x4e, 0x2d, 0x92, 0x92, 0x4c,
	0xcf, 0xcf, 0x4f, 0xcf, 0x49, 0xd5, 0x07, 0x8b, 0x25, 0x95, 0xa6, 0xe9, 0x27, 0xe6, 0x55, 0x42,
	0x14, 0x28, 0xe5, 0x71, 0xf1, 0xfa, 0x80, 0xd4, 0x07, 0xa5, 0x16, 0x17, 0xe4, 0xe7, 0x15, 0xa7,
	0x0a, 0x89, 0x71, 0xb1, 0x15, 0xa5, 0x16, 0x97, 0xe6, 0x94, 0x48, 0x30, 0x2a, 0x30, 0x6a, 0x70,
	0x04, 0x41, 0x79, 0x42, 0xe2, 0x5c, 0xec, 0x20, 0xb3, 0xe2, 0x33, 0x53, 0x24, 0x98, 0x14, 0x18,
	0x35, 0x38, 0x83, 0xd8, 0x40, 0x5c, 0xcf, 0x14, 0x21, 0x1d, 0x2e, 0xb6, 0x82, 0xc4, 0xa2, 0xc4,
	0xdc, 0x62, 0x09, 0x66, 0x05, 0x46, 0x0d, 0x6e, 0x23, 0x11, 0x3d, 0x88, 0x6d, 0x7a, 0x30, 0xdb,
	0xf4, 0x1c, 0xf3, 0x2a, 0x83, 0xa0, 0x6a, 0x92, 0xd8, 0xc0, 0xa2, 0xc6, 0x80, 0x00, 0x00, 0x00,
	0xff, 0xff, 0x6c, 0x55, 0x7d, 0x27, 0xa6, 0x00, 0x00, 0x00,
}
