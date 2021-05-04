// The MIT License (MIT)
//
// Copyright (c) 2017-2020 Uber Technologies Inc.
//
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

package serialization

import (
	"fmt"

	"github.com/uber/cadence/common"
	"github.com/uber/cadence/common/persistence"
)

type (
	parser struct {
		encoder  encoder
		decoders map[common.EncodingType]decoder
	}
)

// NewParser constructs a new parser using encoder as specified by encodingType and using decoders specified by decodingTypes
func NewParser(encodingType common.EncodingType, decodingTypes ...common.EncodingType) (Parser, error) {
	encoder, err := getEncoder(encodingType)
	if err != nil {
		return nil, err
	}
	decoders := make(map[common.EncodingType]decoder)
	for _, dt := range decodingTypes {
		decoder, err := getDecoder(dt)
		if err != nil {
			return nil, err
		}
		decoders[dt] = decoder
	}
	return &parser{
		encoder:  encoder,
		decoders: decoders,
	}, nil
}

func (p *parser) ShardInfoToBlob(info *ShardInfo) (persistence.DataBlob, error) {
	db := persistence.DataBlob{}
	data, err := p.encoder.shardInfoToBlob(info)
	if err != nil {
		return db, err
	}
	db.Data = data
	db.Encoding = p.encoder.encodingType()
	return db, nil
}

func (p *parser) DomainInfoToBlob(info *DomainInfo) (persistence.DataBlob, error) {
	db := persistence.DataBlob{}
	data, err := p.encoder.domainInfoToBlob(info)
	if err != nil {
		return db, err
	}
	db.Data = data
	db.Encoding = p.encoder.encodingType()
	return db, nil
}

func (p *parser) HistoryTreeInfoToBlob(info *HistoryTreeInfo) (persistence.DataBlob, error) {
	db := persistence.DataBlob{}
	data, err := p.encoder.historyTreeInfoToBlob(info)
	if err != nil {
		return db, err
	}
	db.Data = data
	db.Encoding = p.encoder.encodingType()
	return db, nil
}

func (p *parser) WorkflowExecutionInfoToBlob(info *WorkflowExecutionInfo) (persistence.DataBlob, error) {
	db := persistence.DataBlob{}
	data, err := p.encoder.workflowExecutionInfoToBlob(info)
	if err != nil {
		return db, err
	}
	db.Data = data
	db.Encoding = p.encoder.encodingType()
	return db, nil
}

func (p *parser) ActivityInfoToBlob(info *ActivityInfo) (persistence.DataBlob, error) {
	db := persistence.DataBlob{}
	data, err := p.encoder.activityInfoToBlob(info)
	if err != nil {
		return db, err
	}
	db.Data = data
	db.Encoding = p.encoder.encodingType()
	return db, nil
}

func (p *parser) ChildExecutionInfoToBlob(info *ChildExecutionInfo) (persistence.DataBlob, error) {
	db := persistence.DataBlob{}
	data, err := p.encoder.childExecutionInfoToBlob(info)
	if err != nil {
		return db, err
	}
	db.Data = data
	db.Encoding = p.encoder.encodingType()
	return db, nil
}

func (p *parser) SignalInfoToBlob(info *SignalInfo) (persistence.DataBlob, error) {
	db := persistence.DataBlob{}
	data, err := p.encoder.signalInfoToBlob(info)
	if err != nil {
		return db, err
	}
	db.Data = data
	db.Encoding = p.encoder.encodingType()
	return db, nil
}

func (p *parser) RequestCancelInfoToBlob(info *RequestCancelInfo) (persistence.DataBlob, error) {
	db := persistence.DataBlob{}
	data, err := p.encoder.requestCancelInfoToBlob(info)
	if err != nil {
		return db, err
	}
	db.Data = data
	db.Encoding = p.encoder.encodingType()
	return db, nil
}

func (p *parser) TimerInfoToBlob(info *TimerInfo) (persistence.DataBlob, error) {
	db := persistence.DataBlob{}
	data, err := p.encoder.timerInfoToBlob(info)
	if err != nil {
		return db, err
	}
	db.Data = data
	db.Encoding = p.encoder.encodingType()
	return db, nil
}

func (p *parser) TaskInfoToBlob(info *TaskInfo) (persistence.DataBlob, error) {
	db := persistence.DataBlob{}
	data, err := p.encoder.taskInfoToBlob(info)
	if err != nil {
		return db, err
	}
	db.Data = data
	db.Encoding = p.encoder.encodingType()
	return db, nil
}

func (p *parser) TaskListInfoToBlob(info *TaskListInfo) (persistence.DataBlob, error) {
	db := persistence.DataBlob{}
	data, err := p.encoder.taskListInfoToBlob(info)
	if err != nil {
		return db, err
	}
	db.Data = data
	db.Encoding = p.encoder.encodingType()
	return db, nil
}

func (p *parser) TransferTaskInfoToBlob(info *TransferTaskInfo) (persistence.DataBlob, error) {
	db := persistence.DataBlob{}
	data, err := p.encoder.transferTaskInfoToBlob(info)
	if err != nil {
		return db, err
	}
	db.Data = data
	db.Encoding = p.encoder.encodingType()
	return db, nil
}

func (p *parser) TimerTaskInfoToBlob(info *TimerTaskInfo) (persistence.DataBlob, error) {
	db := persistence.DataBlob{}
	data, err := p.encoder.timerTaskInfoToBlob(info)
	if err != nil {
		return db, err
	}
	db.Data = data
	db.Encoding = p.encoder.encodingType()
	return db, nil
}

func (p *parser) ReplicationTaskInfoToBlob(info *ReplicationTaskInfo) (persistence.DataBlob, error) {
	db := persistence.DataBlob{}
	data, err := p.encoder.replicationTaskInfoToBlob(info)
	if err != nil {
		return db, err
	}
	db.Data = data
	db.Encoding = p.encoder.encodingType()
	return db, nil
}

func (p *parser) ShardInfoFromBlob(data []byte, encoding string) (*ShardInfo, error) {
	decoder, err := p.getCachedDecoder(common.EncodingType(encoding))
	if err != nil {
		return nil, err
	}
	return decoder.shardInfoFromBlob(data)
}

func (p *parser) DomainInfoFromBlob(data []byte, encoding string) (*DomainInfo, error) {
	decoder, err := p.getCachedDecoder(common.EncodingType(encoding))
	if err != nil {
		return nil, err
	}
	return decoder.domainInfoFromBlob(data)
}

func (p *parser) HistoryTreeInfoFromBlob(data []byte, encoding string) (*HistoryTreeInfo, error) {
	decoder, err := p.getCachedDecoder(common.EncodingType(encoding))
	if err != nil {
		return nil, err
	}
	return decoder.historyTreeInfoFromBlob(data)
}

func (p *parser) WorkflowExecutionInfoFromBlob(data []byte, encoding string) (*WorkflowExecutionInfo, error) {
	decoder, err := p.getCachedDecoder(common.EncodingType(encoding))
	if err != nil {
		return nil, err
	}
	return decoder.workflowExecutionInfoFromBlob(data)
}

func (p *parser) ActivityInfoFromBlob(data []byte, encoding string) (*ActivityInfo, error) {
	decoder, err := p.getCachedDecoder(common.EncodingType(encoding))
	if err != nil {
		return nil, err
	}
	return decoder.activityInfoFromBlob(data)
}

func (p *parser) ChildExecutionInfoFromBlob(data []byte, encoding string) (*ChildExecutionInfo, error) {
	decoder, err := p.getCachedDecoder(common.EncodingType(encoding))
	if err != nil {
		return nil, err
	}
	return decoder.childExecutionInfoFromBlob(data)
}

func (p *parser) SignalInfoFromBlob(data []byte, encoding string) (*SignalInfo, error) {
	decoder, err := p.getCachedDecoder(common.EncodingType(encoding))
	if err != nil {
		return nil, err
	}
	return decoder.signalInfoFromBlob(data)
}

func (p *parser) RequestCancelInfoFromBlob(data []byte, encoding string) (*RequestCancelInfo, error) {
	decoder, err := p.getCachedDecoder(common.EncodingType(encoding))
	if err != nil {
		return nil, err
	}
	return decoder.requestCancelInfoFromBlob(data)
}

func (p *parser) TimerInfoFromBlob(data []byte, encoding string) (*TimerInfo, error) {
	decoder, err := p.getCachedDecoder(common.EncodingType(encoding))
	if err != nil {
		return nil, err
	}
	return decoder.timerInfoFromBlob(data)
}

func (p *parser) TaskInfoFromBlob(data []byte, encoding string) (*TaskInfo, error) {
	decoder, err := p.getCachedDecoder(common.EncodingType(encoding))
	if err != nil {
		return nil, err
	}
	return decoder.taskInfoFromBlob(data)
}

func (p *parser) TaskListInfoFromBlob(data []byte, encoding string) (*TaskListInfo, error) {
	decoder, err := p.getCachedDecoder(common.EncodingType(encoding))
	if err != nil {
		return nil, err
	}
	return decoder.taskListInfoFromBlob(data)
}

func (p *parser) TransferTaskInfoFromBlob(data []byte, encoding string) (*TransferTaskInfo, error) {
	decoder, err := p.getCachedDecoder(common.EncodingType(encoding))
	if err != nil {
		return nil, err
	}
	return decoder.transferTaskInfoFromBlob(data)
}

func (p *parser) TimerTaskInfoFromBlob(data []byte, encoding string) (*TimerTaskInfo, error) {
	decoder, err := p.getCachedDecoder(common.EncodingType(encoding))
	if err != nil {
		return nil, err
	}
	return decoder.timerTaskInfoFromBlob(data)
}

func (p *parser) ReplicationTaskInfoFromBlob(data []byte, encoding string) (*ReplicationTaskInfo, error) {
	decoder, err := p.getCachedDecoder(common.EncodingType(encoding))
	if err != nil {
		return nil, err
	}
	return decoder.replicationTaskInfoFromBlob(data)
}

func (p *parser) getCachedDecoder(encoding common.EncodingType) (decoder, error) {
	decoder, ok := p.decoders[encoding]
	if !ok {
		return nil, unsupportedEncodingError(encoding)
	}
	return decoder, nil
}

func getDecoder(encoding common.EncodingType) (decoder, error) {
	switch encoding {
	case common.EncodingTypeThriftRW:
		return newThriftDecoder(), nil
	case common.EncodingTypeProto:
		return newProtoDecoder(), nil
	default:
		return nil, unsupportedEncodingError(encoding)
	}
}

func getEncoder(encoding common.EncodingType) (encoder, error) {
	switch encoding {
	case common.EncodingTypeThriftRW:
		return newThriftEncoder(), nil
	case common.EncodingTypeProto:
		return newProtoEncoder(), nil
	default:
		return nil, unsupportedEncodingError(encoding)
	}
}

func unsupportedEncodingError(encoding common.EncodingType) error {
	return fmt.Errorf("invalid encoding type: %v", encoding)
}
