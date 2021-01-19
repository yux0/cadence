// The MIT License (MIT)

// Copyright (c) 2017-2020 Uber Technologies Inc.

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
// 
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
// 
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: uber/cadence/shared/v1/queue.proto

package sharedv1

import (
	fmt "fmt"
	math "math"
	strconv "strconv"

	proto "github.com/gogo/protobuf/proto"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion3 // please upgrade the proto package

type TaskType int32

const (
	TASK_TYPE_INVALID     TaskType = 0
	TASK_TYPE_TRANSFER    TaskType = 1
	TASK_TYPE_TIMER       TaskType = 2
	TASK_TYPE_REPLICATION TaskType = 3
)

var TaskType_name = map[int32]string{
	0: "TASK_TYPE_INVALID",
	1: "TASK_TYPE_TRANSFER",
	2: "TASK_TYPE_TIMER",
	3: "TASK_TYPE_REPLICATION",
}

var TaskType_value = map[string]int32{
	"TASK_TYPE_INVALID":     0,
	"TASK_TYPE_TRANSFER":    1,
	"TASK_TYPE_TIMER":       2,
	"TASK_TYPE_REPLICATION": 3,
}

func (TaskType) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_8958fa454fc8f819, []int{0}
}

func init() {
	proto.RegisterEnum("uber.cadence.shared.v1.TaskType", TaskType_name, TaskType_value)
}

func init() {
	proto.RegisterFile("github.com/uber/cadence/.gen/proto/shared/v1/queue.proto", fileDescriptor_8958fa454fc8f819)
}

var fileDescriptor_8958fa454fc8f819 = []byte{
	// 223 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x52, 0x2a, 0x4d, 0x4a, 0x2d,
	0xd2, 0x4f, 0x4e, 0x4c, 0x49, 0xcd, 0x4b, 0x4e, 0xd5, 0x2f, 0xce, 0x48, 0x2c, 0x4a, 0x4d, 0xd1,
	0x2f, 0x33, 0xd4, 0x2f, 0x2c, 0x4d, 0x2d, 0x4d, 0xd5, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0x12,
	0x03, 0xa9, 0xd1, 0x83, 0xaa, 0xd1, 0x83, 0xa8, 0xd1, 0x2b, 0x33, 0xd4, 0xca, 0xe4, 0xe2, 0x08,
	0x49, 0x2c, 0xce, 0x0e, 0xa9, 0x2c, 0x48, 0x15, 0x12, 0xe5, 0x12, 0x0c, 0x71, 0x0c, 0xf6, 0x8e,
	0x0f, 0x89, 0x0c, 0x70, 0x8d, 0xf7, 0xf4, 0x0b, 0x73, 0xf4, 0xf1, 0x74, 0x11, 0x60, 0x10, 0x12,
	0xe3, 0x12, 0x42, 0x08, 0x87, 0x04, 0x39, 0xfa, 0x05, 0xbb, 0xb9, 0x06, 0x09, 0x30, 0x0a, 0x09,
	0x73, 0xf1, 0x23, 0x89, 0x7b, 0xfa, 0xba, 0x06, 0x09, 0x30, 0x09, 0x49, 0x72, 0x89, 0x22, 0x04,
	0x83, 0x5c, 0x03, 0x7c, 0x3c, 0x9d, 0x1d, 0x43, 0x3c, 0xfd, 0xfd, 0x04, 0x98, 0x9d, 0xec, 0x2e,
	0x3c, 0x94, 0x63, 0xb8, 0xf1, 0x50, 0x8e, 0xe1, 0xc3, 0x43, 0x39, 0xc6, 0x86, 0x47, 0x72, 0x8c,
	0x2b, 0x1e, 0xc9, 0x31, 0x9e, 0x78, 0x24, 0xc7, 0x78, 0xe1, 0x91, 0x1c, 0xe3, 0x83, 0x47, 0x72,
	0x8c, 0x2f, 0x1e, 0xc9, 0x31, 0x7c, 0x78, 0x24, 0xc7, 0x38, 0xe1, 0xb1, 0x1c, 0xc3, 0x85, 0xc7,
	0x72, 0x0c, 0x37, 0x1e, 0xcb, 0x31, 0x44, 0x71, 0x40, 0xdc, 0x5a, 0x66, 0x98, 0xc4, 0x06, 0xf6,
	0x89, 0x31, 0x20, 0x00, 0x00, 0xff, 0xff, 0xdc, 0x43, 0x71, 0x2c, 0xef, 0x00, 0x00, 0x00,
}

func (x TaskType) String() string {
	s, ok := TaskType_name[int32(x)]
	if ok {
		return s
	}
	return strconv.Itoa(int(x))
}
