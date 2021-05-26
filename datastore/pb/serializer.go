package pb

import (
	"github.com/gogo/protobuf/proto"
	"github.com/hashicorp/raft"
)

func EncodeLog(log *raft.Log) ([]byte, error) {
	obj := &LogPb{
		Index:      log.Index,
		Term:       log.Term,
		Type:       uint64(log.Type),
		Data:       log.Data,
		Extensions: log.Extensions,
	}
	return proto.Marshal(obj)
}

func MustEncodeLog(log *raft.Log) []byte {
	bs, err := EncodeLog(log)
	if err != nil {
		panic(err)
	}
	return bs
}

func DecodeLog(bs []byte, log *raft.Log) error {
	obj := &LogPb{}
	err := proto.Unmarshal(bs, obj)
	if err != nil {
		return err
	}
	log.Index = obj.Index
	log.Term = obj.Term
	log.Type = raft.LogType(obj.Type)
	log.Data = obj.Data
	log.Extensions = obj.Extensions
	return nil
}

func MustDecodeLog(bs []byte, log *raft.Log) {
	err := DecodeLog(bs, log)
	if err != nil {
		panic(err)
	}
}
