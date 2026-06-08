package wal

import (
	"fmt"

	walpb "github.com/imPranav14/Distributed-KV-Store/proto/wal"
	"google.golang.org/protobuf/proto"
)

type OpType int

const (
	OpTypeUnknown OpType = iota
	OpTypePut
	OpTypeAppend
)

func (t OpType) toProto() walpb.OpType {
	switch t {
	case OpTypePut:
		return walpb.OpType_PUT
	case OpTypeAppend:
		return walpb.OpType_APPEND
	default:
		return walpb.OpType_OP_TYPE_UNSPECIFIED
	}
}

func opTypeFromProto(t walpb.OpType) (OpType, error) {
	switch t {
	case walpb.OpType_PUT:
		return OpTypePut, nil
	case walpb.OpType_APPEND:
		return OpTypeAppend, nil
	case walpb.OpType_OP_TYPE_UNSPECIFIED:
		fallthrough
	default:
		return OpTypeUnknown, fmt.Errorf("unknown op type %v", t)
	}
}

type Entry struct {
	Type      OpType
	Key       string
	Value     string
	ClientID  string
	RequestID uint64
}

func (e *Entry) ToProto() *walpb.WalEntry {
	return &walpb.WalEntry{
		Type:      e.Type.toProto(),
		Key:       e.Key,
		Value:     e.Value,
		ClientId:  e.ClientID,
		RequestId: e.RequestID,
	}
}

func EntryFromProto(msg *walpb.WalEntry) (*Entry, error) {
	if msg == nil {
		return nil, fmt.Errorf("nil WalEntry")
	}
	opType, err := opTypeFromProto(msg.Type)
	if err != nil {
		return nil, err
	}
	if msg.Key == "" {
		return nil, fmt.Errorf("wal entry key cannot be empty")
	}

	return &Entry{
		Type:      opType,
		Key:       msg.Key,
		Value:     msg.Value,
		ClientID:  msg.ClientId,
		RequestID: msg.RequestId,
	}, nil
}

func (e *Entry) Marshal() ([]byte, error) {
	return proto.Marshal(e.ToProto())
}

func UnmarshalEntry(data []byte) (*Entry, error) {
	msg := new(walpb.WalEntry)
	if err := proto.Unmarshal(data, msg); err != nil {
		return nil, err
	}
	return EntryFromProto(msg)
}
