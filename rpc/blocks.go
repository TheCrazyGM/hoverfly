package rpc

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/thecrazygm/hoverfly/state"
)

func (h *RPCHandler) handleGetBlock(params json.RawMessage, method string) (any, *rpcError) {
	var blockNum uint32
	var arrArgs []uint32
	if err := json.Unmarshal(params, &arrArgs); err == nil && len(arrArgs) > 0 {
		blockNum = arrArgs[0]
	} else {
		var objArgs struct {
			BlockNum uint32 `json:"block_num"`
		}
		if err := json.Unmarshal(params, &objArgs); err == nil && objArgs.BlockNum > 0 {
			blockNum = objArgs.BlockNum
		} else {
			var arrObjArgs []struct {
				BlockNum uint32 `json:"block_num"`
			}
			if err := json.Unmarshal(params, &arrObjArgs); err == nil && len(arrObjArgs) > 0 {
				blockNum = arrObjArgs[0].BlockNum
			} else {
				return nil, &rpcError{Code: -32602, Message: "Invalid parameters"}
			}
		}
	}

	blockObj := map[string]any{
		"block_id":                fmt.Sprintf("05f5e100f72d57fd5a542459a94f3a8153c68c%02d", blockNum%100),
		"previous":                "05f5e0fff72d57fd5a542459a94f3a8153c68c4a",
		"timestamp":               time.Now().UTC().Format("2006-01-02T15:04:05"),
		"witness":                 "blocktrades",
		"transaction_merkle_root": "0000000000000000000000000000000000000000",
		"extensions":              []any{},
		"witness_signature":       "207f...mock...sig",
		"transactions":            []any{},
		"transaction_ids":         []any{},
		"signing_key":             "STM6ipXFLZyBeJRLFkXNRzAeQDz5T9zawSzYUdMShPsBHqB9W4SaC",
	}

	if method == "block_api.get_block" {
		return map[string]any{
			"block": blockObj,
		}, nil
	}
	return blockObj, nil
}

func (h *RPCHandler) handleGetBlockHeader(params json.RawMessage, method string) (any, *rpcError) {
	var blockNum uint32
	var arrArgs []uint32
	if err := json.Unmarshal(params, &arrArgs); err == nil && len(arrArgs) > 0 {
		blockNum = arrArgs[0]
	} else {
		var objArgs struct {
			BlockNum uint32 `json:"block_num"`
		}
		if err := json.Unmarshal(params, &objArgs); err == nil && objArgs.BlockNum > 0 {
			blockNum = objArgs.BlockNum
		} else {
			var arrObjArgs []struct {
				BlockNum uint32 `json:"block_num"`
			}
			if err := json.Unmarshal(params, &arrObjArgs); err == nil && len(arrObjArgs) > 0 {
				blockNum = arrObjArgs[0].BlockNum
			} else {
				return nil, &rpcError{Code: -32602, Message: "Invalid parameters"}
			}
		}
	}

	headerObj := map[string]any{
		"previous":                fmt.Sprintf("05f5e100f72d57fd5a542459a94f3a8153c68c%02d", (blockNum-1)%100),
		"timestamp":               time.Now().UTC().Format("2006-01-02T15:04:05"),
		"witness":                 "blocktrades",
		"transaction_merkle_root": "0000000000000000000000000000000000000000",
		"extensions":              []any{},
	}

	if method == "block_api.get_block_header" {
		return map[string]any{
			"header": headerObj,
		}, nil
	}
	return headerObj, nil
}

func (h *RPCHandler) handleGetBlockRange(params json.RawMessage) (any, *rpcError) {
	var args struct {
		StartingBlockNum uint32 `json:"starting_block_num"`
		Count            uint32 `json:"count"`
	}
	if err := json.Unmarshal(params, &args); err != nil || args.Count == 0 {
		var arrArgs []struct {
			StartingBlockNum uint32 `json:"starting_block_num"`
			Count            uint32 `json:"count"`
		}
		if err := json.Unmarshal(params, &arrArgs); err == nil && len(arrArgs) > 0 {
			args = arrArgs[0]
		}
	}
	if args.Count == 0 {
		return nil, &rpcError{Code: -32602, Message: "Invalid parameters"}
	}
	if args.Count > 1000 {
		args.Count = 1000
	}

	blocks := make([]any, 0, args.Count)
	for i := uint32(0); i < args.Count; i++ {
		block, rpcErr := h.handleGetBlock(mustMarshal(map[string]uint32{"block_num": args.StartingBlockNum + i}), "block_api.get_block")
		if rpcErr != nil {
			return nil, rpcErr
		}
		if blockMap, ok := block.(map[string]any); ok {
			blocks = append(blocks, blockMap["block"])
		}
	}

	return map[string]any{"blocks": blocks}, nil
}

func (h *RPCHandler) handleGetAccountHistory(method string, params json.RawMessage) (any, *rpcError) {
	account, limit := parseAccountLimit(params)
	if account == "" {
		var args []any
		if err := json.Unmarshal(params, &args); err == nil && len(args) > 0 {
			account, _ = args[0].(string)
			if len(args) > 2 {
				if rawLimit, ok := args[2].(float64); ok && rawLimit > 0 {
					limit = int(rawLimit)
				}
			}
		}
	}
	if account == "" {
		return nil, &rpcError{Code: -32602, Message: "Invalid parameters"}
	}
	limit = clampLimit(limit, 1000)

	transactions, err := h.state.ListTransactions()
	if err != nil {
		return nil, &rpcError{Code: -32603, Message: err.Error()}
	}

	history := make([]any, 0, limit)
	index := 0
	for _, txData := range transactions {
		for _, op := range txData.Operations {
			if !operationTouchesAccount(op, account) {
				continue
			}
			entry := map[string]any{
				"trx_id":       txData.TransactionID,
				"block":        txData.BlockNum,
				"trx_in_block": txData.TransactionNum,
				"op_in_trx":    0,
				"timestamp":    txData.Expiration,
				"op":           op,
				"virtual_op":   false,
			}
			if method == "condenser_api.get_account_history" {
				history = append(history, []any{index, entry})
			} else {
				history = append(history, entry)
			}
			index++
			if len(history) >= limit {
				break
			}
		}
		if len(history) >= limit {
			break
		}
	}

	if method == "condenser_api.get_account_history" {
		return history, nil
	}
	return map[string]any{"history": history}, nil
}

func operationTouchesAccount(op any, account string) bool {
	bytes, err := json.Marshal(op)
	if err != nil {
		return false
	}
	return strings.Contains(string(bytes), `"`+account+`"`)
}

func (h *RPCHandler) handleGetOpsInBlock(params json.RawMessage) (any, *rpcError) {
	blockNum, onlyVirtual := parseBlockNumAndVirtual(params)
	if blockNum == 0 {
		return nil, &rpcError{Code: -32602, Message: "Invalid parameters"}
	}

	transactions, err := h.state.ListTransactions()
	if err != nil {
		return nil, &rpcError{Code: -32603, Message: err.Error()}
	}

	ops := make([]any, 0)
	if !onlyVirtual {
		for _, txData := range transactions {
			if txData.BlockNum != blockNum {
				continue
			}
			for idx, op := range txData.Operations {
				ops = append(ops, map[string]any{
					"trx_id":       txData.TransactionID,
					"block":        txData.BlockNum,
					"trx_in_block": txData.TransactionNum,
					"op_in_trx":    idx,
					"timestamp":    txData.Expiration,
					"op":           op,
					"virtual_op":   false,
				})
			}
		}
	}
	return map[string]any{"ops": ops}, nil
}

func (h *RPCHandler) handleCondenserGetOpsInBlock(params json.RawMessage) (any, *rpcError) {
	result, rpcErr := h.handleGetOpsInBlock(params)
	if rpcErr != nil {
		return nil, rpcErr
	}
	resultMap, _ := result.(map[string]any)
	ops, _ := resultMap["ops"].([]any)
	return ops, nil
}

func parseBlockNumAndVirtual(params json.RawMessage) (uint32, bool) {
	var args []any
	if err := json.Unmarshal(params, &args); err == nil && len(args) > 0 {
		blockNum, _ := args[0].(float64)
		onlyVirtual := false
		if len(args) > 1 {
			onlyVirtual, _ = args[1].(bool)
		}
		return uint32(blockNum), onlyVirtual
	}

	var objectArgs struct {
		BlockNum    uint32 `json:"block_num"`
		OnlyVirtual bool   `json:"only_virtual"`
	}
	if err := json.Unmarshal(params, &objectArgs); err == nil {
		return objectArgs.BlockNum, objectArgs.OnlyVirtual
	}
	return 0, false
}

func (h *RPCHandler) handleEnumVirtualOps(params json.RawMessage) (any, *rpcError) {
	return map[string]any{
		"ops":                    []any{},
		"ops_by_block":           []any{},
		"next_block_range_begin": 0,
		"next_operation_begin":   "0",
	}, nil
}

func parseTransactionID(params json.RawMessage) string {
	var stringArgs []string
	if err := json.Unmarshal(params, &stringArgs); err == nil && len(stringArgs) > 0 {
		return stringArgs[0]
	}

	var objectArgs struct {
		ID            string `json:"id"`
		TransactionID string `json:"transaction_id"`
		TrxID         string `json:"trx_id"`
	}
	if err := json.Unmarshal(params, &objectArgs); err == nil {
		switch {
		case objectArgs.ID != "":
			return objectArgs.ID
		case objectArgs.TransactionID != "":
			return objectArgs.TransactionID
		case objectArgs.TrxID != "":
			return objectArgs.TrxID
		}
	}

	var wrappedObjectArgs []struct {
		ID            string `json:"id"`
		TransactionID string `json:"transaction_id"`
		TrxID         string `json:"trx_id"`
	}
	if err := json.Unmarshal(params, &wrappedObjectArgs); err == nil && len(wrappedObjectArgs) > 0 {
		switch {
		case wrappedObjectArgs[0].ID != "":
			return wrappedObjectArgs[0].ID
		case wrappedObjectArgs[0].TransactionID != "":
			return wrappedObjectArgs[0].TransactionID
		case wrappedObjectArgs[0].TrxID != "":
			return wrappedObjectArgs[0].TrxID
		}
	}

	return ""
}

func (h *RPCHandler) handleGetFollowCount(params json.RawMessage) (any, *rpcError) {
	var names []string
	if err := json.Unmarshal(params, &names); err == nil && len(names) > 0 {
		return map[string]any{
			"account":         names[0],
			"follower_count":  10,
			"following_count": 5,
		}, nil
	}

	var args struct {
		Account string `json:"account"`
	}
	if err := json.Unmarshal(params, &args); err == nil && args.Account != "" {
		return map[string]any{
			"account":         args.Account,
			"follower_count":  10,
			"following_count": 5,
		}, nil
	}

	return nil, &rpcError{Code: -32602, Message: "Invalid parameters"}
}

func (h *RPCHandler) handleIsKnownTransaction(method string, params json.RawMessage) (any, *rpcError) {
	txID := parseTransactionID(params)
	if txID == "" {
		return nil, &rpcError{Code: -32602, Message: "Invalid parameters"}
	}

	_, err := h.state.GetTransaction(txID)
	isKnown := err == nil
	if method == "database_api.is_known_transaction" {
		return map[string]any{"is_known": isKnown}, nil
	}
	return isKnown, nil
}

func (h *RPCHandler) handleFindTransaction(params json.RawMessage) (any, *rpcError) {
	txID := parseTransactionID(params)
	if txID == "" {
		return nil, &rpcError{Code: -32602, Message: "Invalid parameters"}
	}

	tx, err := h.state.GetTransaction(txID)
	if err != nil {
		return map[string]any{"status": "unknown"}, nil
	}

	props, err := h.state.GetDynamicProperties()
	if err == nil && props != nil && tx.BlockNum <= props.LastIrreversibleBlockNum {
		return map[string]any{"status": "within_irreversible_block"}, nil
	}
	return map[string]any{"status": "within_reversible_block"}, nil
}

func (h *RPCHandler) handleGetMethods() (any, *rpcError) {
	methods := make([]string, 0, len(knownHiveAPIMethods))
	for method := range knownHiveAPIMethods {
		methods = append(methods, method)
	}
	sort.Strings(methods)
	return methods, nil
}

func (h *RPCHandler) handleGetSignature(params json.RawMessage) (any, *rpcError) {
	var args struct {
		Method string `json:"method"`
	}
	if err := json.Unmarshal(params, &args); err != nil || args.Method == "" {
		var arrArgs []struct {
			Method string `json:"method"`
		}
		if err := json.Unmarshal(params, &arrArgs); err == nil && len(arrArgs) > 0 {
			args.Method = arrArgs[0].Method
		}
	}
	if args.Method == "" {
		return nil, &rpcError{Code: -32602, Message: "Invalid parameters"}
	}
	if _, ok := knownHiveAPIMethods[args.Method]; !ok {
		return nil, &rpcError{Code: -32601, Message: fmt.Sprintf("Method not found: %s", args.Method)}
	}

	var responseShape any
	if raw, ok := openAPIMockResponses[args.Method]; ok {
		_ = json.Unmarshal([]byte(raw), &responseShape)
	}

	return map[string]any{
		"args": map[string]any{
			"method": args.Method,
			"params": "see https://api.syncad.com/hived-api/",
		},
		"ret": responseShape,
	}, nil
}

func (h *RPCHandler) handleHiveInfo() (any, *rpcError) {
	props, err := h.state.GetDynamicProperties()
	if err != nil {
		return nil, &rpcError{Code: -32603, Message: err.Error()}
	}

	return map[string]any{
		"database_head_block":          props.HeadBlockNumber,
		"database_schema_version":      2,
		"database_patch_date":          props.Time,
		"database_patched_to_revision": "hoverfly",
		"hivemind_version":             "hoverfly-0.1.0",
		"hivemind_git_rev":             "hoverfly",
		"hivemind_git_date":            props.Time,
	}, nil
}

func (h *RPCHandler) handleGetTransaction(params json.RawMessage) (any, *rpcError) {
	txID := parseTransactionID(params)
	if txID == "" {
		return nil, &rpcError{Code: -32602, Message: "Invalid parameters"}
	}

	tx, err := h.state.GetTransaction(txID)
	if err != nil {
		// Return a schema-compliant mock transaction for unknown tx IDs
		fallback := state.TransactionData{
			TransactionID:  txID,
			BlockNum:       100000,
			TransactionNum: 0,
			RefBlockNum:    1097,
			RefBlockPrefix: 2181793527,
			Expiration:     "2026-05-26T18:00:00",
			Operations:     []any{json.RawMessage(`["transfer",{"from":"initminer","to":"thecrazygm","amount":"100.000 HIVE","memo":"fallback mock transaction"}]`)},
			Signatures:     []string{},
		}
		return &fallback, nil
	}

	return tx, nil
}
