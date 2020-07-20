/*
 * Copyright 2020, Offchain Labs, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cmachine

/*
#cgo CFLAGS: -I.
#cgo LDFLAGS: -L. -L../build/rocksdb -lcavm -lavm -ldata_storage -lavm_values -lavm_utils -lstdc++ -lm -lrocksdb
#include "../cavm/caggregator.h"
#include <stdio.h>
#include <stdlib.h>
*/
import "C"
import (
	"bytes"
	"errors"
	"github.com/offchainlabs/arbitrum/packages/arb-util/machine"
	"math/big"
	"runtime"
	"unsafe"

	"github.com/offchainlabs/arbitrum/packages/arb-util/common"
	"github.com/offchainlabs/arbitrum/packages/arb-util/value"
)

type AggregatorStore struct {
	c unsafe.Pointer
}

func deleteAggregatorStore(bs *BlockStore) {
	C.deleteAggregatorStore(bs.c)
}

func NewAggregatorStore(c unsafe.Pointer) *AggregatorStore {
	as := &AggregatorStore{c: c}
	runtime.SetFinalizer(as, deleteAggregatorStore)
	return as
}

func (as *AggregatorStore) LogCount() (uint64, error) {
	result := C.aggregatorLogCount(as.c)
	if result.found == 0 {
		return 0, errors.New("failed to load log count")
	}
	return uint64(result.value), nil
}

func (as *AggregatorStore) SaveLog(val value.Value) error {
	var buf bytes.Buffer
	if err := value.MarshalValue(val, &buf); err != nil {
		return err
	}

	cData := C.CBytes(buf.Bytes())
	defer C.free(cData)
	if C.aggregatorSaveLog(as.c, cData, C.uint64_t(buf.Len())) == 0 {
		return errors.New("failed to save block")
	}
	return nil
}

func (as *AggregatorStore) GetLog(index uint64) (value.Value, error) {
	result := C.aggregatorGetLog(as.c, C.uint64_t(index))
	if result.found == 0 {
		return nil, errors.New("failed to get log")
	}
	logBytes := toByteSlice(result.slice)
	return value.UnmarshalValue(bytes.NewBuffer(logBytes))
}

func (as *AggregatorStore) MessageCount() (uint64, error) {
	result := C.aggregatorMessageCount(as.c)
	if result.found == 0 {
		return 0, errors.New("failed to load message count")
	}
	return uint64(result.value), nil
}

func (as *AggregatorStore) SaveMessage(val value.Value) error {
	var buf bytes.Buffer
	if err := value.MarshalValue(val, &buf); err != nil {
		return err
	}

	cData := C.CBytes(buf.Bytes())
	defer C.free(cData)
	if C.aggregatorSaveMessage(as.c, cData, C.uint64_t(buf.Len())) == 0 {
		return errors.New("failed to save block")
	}

	return nil
}

func (as *AggregatorStore) GetMessage(index uint64) (value.Value, error) {
	result := C.aggregatorGetMessage(as.c, C.uint64_t(index))
	if result.found == 0 {
		return nil, errors.New("failed to get message")
	}
	logBytes := toByteSlice(result.slice)
	return value.UnmarshalValue(bytes.NewBuffer(logBytes))
}

func (as *AggregatorStore) LatestBlock() (*common.BlockId, error) {
	result := C.aggregatorLatestBlock(as.c)
	if result.found == 0 {
		return nil, errors.New("failed to load block count")
	}
	return &common.BlockId{
		Height:     common.NewTimeBlocks(new(big.Int).SetUint64(uint64(result.height))),
		HeaderHash: dataToHash(result.hash),
	}, nil
}

func (as *AggregatorStore) SaveBlock(id *common.BlockId, logBloom common.Hash) error {
	cHeaderHash := hashToData(id.HeaderHash)
	defer C.free(cHeaderHash)

	cLogBloom := hashToData(logBloom)
	defer C.free(cLogBloom)

	if C.aggregatorSaveBlock(as.c, C.uint64_t(id.Height.AsInt().Uint64()), cHeaderHash, cLogBloom) == 0 {
		return errors.New("failed to save block")
	}
	return nil
}

func (as *AggregatorStore) GetBlock(height uint64) (machine.BlockInfo, error) {
	blockData := C.aggregatorGetBlock(as.c, C.uint64_t(height))
	if blockData.found == 0 {
		return machine.BlockInfo{}, errors.New("failed to get block")
	}
	return machine.BlockInfo{
		Hash:         dataToHash(blockData.hash),
		StartLog:     uint64(blockData.start_log),
		LogCount:     uint64(blockData.log_count),
		StartMessage: uint64(blockData.start_message),
		MessageCount: uint64(blockData.message_count),
		Bloom:        dataToHash(blockData.hash),
	}, nil
}

func (as *AggregatorStore) RestoreBlock(height uint64) error {
	if C.aggregatorRestoreBlock(as.c, C.uint64_t(height)) == 0 {
		return errors.New("failed to restore block")
	}
	return nil
}

func (as *AggregatorStore) GetPossibleRequestInfo(requestId common.Hash) (uint64, uint64, error) {
	cHash := hashToData(requestId)
	defer C.free(cHash)

	result := C.aggregatorGetPossibleRequestInfo(as.c, cHash)
	if result.found == 0 {
		return 0, 0, errors.New("failed to get request")
	}
	return uint64(result.log_index), uint64(result.evm_start_log_index), nil
}

func (as *AggregatorStore) SaveRequest(requestId common.Hash, logIndex uint64, evmLogStartIndex uint64) error {
	cHash := hashToData(requestId)
	defer C.free(cHash)

	if C.aggregatorSaveRequest(as.c, cHash, C.uint64_t(logIndex), C.uint64_t(evmLogStartIndex)) == 0 {
		return errors.New("failed to save request")
	}
	return nil
}
