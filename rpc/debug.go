package rpc

import (
	"encoding/json"
	"fmt"
	"time"
)

func (h *RPCHandler) handleDebugHeadBlock() (any, *rpcError) {
	props, err := h.state.GetDynamicProperties()
	if err != nil {
		return nil, &rpcError{Code: -32603, Message: err.Error()}
	}
	return h.handleGetBlock(mustMarshal(map[string]uint32{"block_num": props.HeadBlockNumber}), "block_api.get_block")
}

func (h *RPCHandler) handleDBHeadState() (any, *rpcError) {
	props, err := h.state.GetDynamicProperties()
	if err != nil {
		return nil, &rpcError{Code: -32603, Message: err.Error()}
	}
	return map[string]any{
		"head_block": map[string]any{
			"block_num": props.HeadBlockNumber,
			"block_id":  props.HeadBlockID,
			"time":      props.Time,
		},
		"last_irreversible_block_num": props.LastIrreversibleBlockNum,
	}, nil
}

func (h *RPCHandler) handleDebugGenerateBlocks(params json.RawMessage) (any, *rpcError) {
	count := uint32(1)
	var args []any
	if err := json.Unmarshal(params, &args); err == nil {
		for _, arg := range args {
			if rawCount, ok := arg.(float64); ok && rawCount > 0 {
				count = uint32(rawCount)
				break
			}
		}
	}
	if count > 10000 {
		count = 10000
	}
	return h.advanceBlocks(count)
}

func (h *RPCHandler) handleDebugGenerateBlocksUntil(params json.RawMessage) (any, *rpcError) {
	target := ""
	var args []any
	if err := json.Unmarshal(params, &args); err == nil && len(args) > 0 {
		target, _ = args[0].(string)
	}
	props, err := h.state.GetDynamicProperties()
	if err != nil {
		return nil, &rpcError{Code: -32603, Message: err.Error()}
	}
	if target == "" {
		return props.HeadBlockNumber, nil
	}
	targetTime, err := time.Parse("2006-01-02T15:04:05", target)
	if err != nil {
		return nil, &rpcError{Code: -32602, Message: err.Error()}
	}
	currentTime, err := time.Parse("2006-01-02T15:04:05", props.Time)
	if err != nil {
		currentTime = time.Now().UTC()
	}
	if !targetTime.After(currentTime) {
		return props.HeadBlockNumber, nil
	}
	blocks := uint32(targetTime.Sub(currentTime).Seconds() / 3)
	if blocks == 0 {
		blocks = 1
	}
	if blocks > 10000 {
		blocks = 10000
	}
	return h.advanceBlocks(blocks)
}

func (h *RPCHandler) advanceBlocks(count uint32) (any, *rpcError) {
	props, err := h.state.GetDynamicProperties()
	if err != nil {
		return nil, &rpcError{Code: -32603, Message: err.Error()}
	}
	props.HeadBlockNumber += count
	if props.HeadBlockNumber > 10 {
		props.LastIrreversibleBlockNum = props.HeadBlockNumber - 10
	}
	props.Time = time.Now().UTC().Format("2006-01-02T15:04:05")
	props.HeadBlockID = fmt.Sprintf("05f5e100f72d57fd5a542459a94f3a8153c68c%02d", props.HeadBlockNumber%100)
	if err := h.state.SaveDynamicProperties(props); err != nil {
		return nil, &rpcError{Code: -32603, Message: err.Error()}
	}
	return props.HeadBlockNumber, nil
}
