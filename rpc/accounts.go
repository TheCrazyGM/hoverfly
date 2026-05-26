package rpc

import (
	"encoding/json"
	"time"

	"github.com/srbde/hoverfly/state"
)

func (h *RPCHandler) handleGetAccounts(params json.RawMessage) (any, *rpcError) {
	var rawList []json.RawMessage
	if err := json.Unmarshal(params, &rawList); err != nil || len(rawList) == 0 {
		return nil, &rpcError{Code: -32602, Message: "Invalid parameters"}
	}
	var names []string
	if err := json.Unmarshal(rawList[0], &names); err != nil {
		return nil, &rpcError{Code: -32602, Message: "Invalid parameters"}
	}

	var results []EnrichedAccountData

	for _, name := range names {
		var baseAcc state.AccountData
		acc, err := h.state.GetAccount(name)
		if err == nil && acc != nil {
			baseAcc = *acc
		} else {
			baseAcc = state.AccountData{
				Name:        name,
				VotingPower: 10000,
				VotingManabar: state.Manabar{
					CurrentMana:    10000,
					LastUpdateTime: time.Now().Unix(),
				},
				LastVoteTime:  "1970-01-01T00:00:00",
				Balance:       "100.000 HIVE",
				HbdBalance:    "10.000 HBD",
				VestingShares: "5000000.000000 VESTS",
				Created:       "2018-01-01T00:00:00",
			}
		}
		results = append(results, enrichAccount(baseAcc))
	}

	return results, nil
}

func (h *RPCHandler) handleFindAccounts(params json.RawMessage) (any, *rpcError) {
	var args struct {
		Accounts []string `json:"accounts"`
	}
	if err := json.Unmarshal(params, &args); err != nil || len(args.Accounts) == 0 {
		var arrArgs []struct {
			Accounts []string `json:"accounts"`
		}
		if err := json.Unmarshal(params, &arrArgs); err == nil && len(arrArgs) > 0 {
			args.Accounts = arrArgs[0].Accounts
		}
	}

	if len(args.Accounts) == 0 {
		return nil, &rpcError{Code: -32602, Message: "Invalid parameters"}
	}

	var results []EnrichedAccountData

	for _, name := range args.Accounts {
		var baseAcc state.AccountData
		acc, err := h.state.GetAccount(name)
		if err == nil && acc != nil {
			baseAcc = *acc
		} else {
			baseAcc = state.AccountData{
				Name:        name,
				VotingPower: 10000,
				VotingManabar: state.Manabar{
					CurrentMana:    10000,
					LastUpdateTime: time.Now().Unix(),
				},
				LastVoteTime:  "1970-01-01T00:00:00",
				Balance:       "100.000 HIVE",
				HbdBalance:    "10.000 HBD",
				VestingShares: "5000000.000000 VESTS",
				Created:       "2018-01-01T00:00:00",
			}
		}
		results = append(results, enrichAccount(baseAcc))
	}

	return map[string]any{
		"accounts": results,
	}, nil
}

func (h *RPCHandler) handleGetAccountCount() (any, *rpcError) {
	accounts, err := h.state.ListAccounts()
	if err != nil {
		return nil, &rpcError{Code: -32603, Message: err.Error()}
	}
	return len(accounts), nil
}

func (h *RPCHandler) handleLookupAccounts(params json.RawMessage) (any, *rpcError) {
	var args []any
	if err := json.Unmarshal(params, &args); err != nil || len(args) == 0 {
		return nil, &rpcError{Code: -32602, Message: "Invalid parameters"}
	}

	lowerBound, _ := args[0].(string)
	limit := 1000
	if len(args) > 1 {
		if rawLimit, ok := args[1].(float64); ok && rawLimit > 0 {
			limit = int(rawLimit)
		}
	}
	if limit > 1000 {
		limit = 1000
	}

	accounts, err := h.state.ListAccounts()
	if err != nil {
		return nil, &rpcError{Code: -32603, Message: err.Error()}
	}

	results := make([]string, 0, limit)
	for _, acc := range accounts {
		if acc.Name >= lowerBound {
			results = append(results, acc.Name)
			if len(results) >= limit {
				break
			}
		}
	}
	return results, nil
}

func (h *RPCHandler) handleLookupAccountNames(params json.RawMessage) (any, *rpcError) {
	var rawList []json.RawMessage
	if err := json.Unmarshal(params, &rawList); err != nil || len(rawList) == 0 {
		return nil, &rpcError{Code: -32602, Message: "Invalid parameters"}
	}
	var names []string
	if err := json.Unmarshal(rawList[0], &names); err != nil {
		return nil, &rpcError{Code: -32602, Message: "Invalid parameters"}
	}

	results := make([]any, 0, len(names))
	for _, name := range names {
		acc, err := h.state.GetAccount(name)
		if err != nil || acc == nil {
			results = append(results, nil)
			continue
		}
		results = append(results, enrichAccount(*acc))
	}
	return results, nil
}

func (h *RPCHandler) handleListAccounts(params json.RawMessage) (any, *rpcError) {
	var args struct {
		Start any    `json:"start"`
		Limit uint32 `json:"limit"`
		Order string `json:"order"`
	}
	if err := json.Unmarshal(params, &args); err != nil {
		var arrArgs []struct {
			Start any    `json:"start"`
			Limit uint32 `json:"limit"`
			Order string `json:"order"`
		}
		if err := json.Unmarshal(params, &arrArgs); err == nil && len(arrArgs) > 0 {
			args = arrArgs[0]
		}
	}
	if args.Limit == 0 {
		args.Limit = 1000
	}
	if args.Limit > 1000 {
		args.Limit = 1000
	}

	startName := ""
	switch start := args.Start.(type) {
	case string:
		startName = start
	case map[string]any:
		if name, ok := start["name"].(string); ok {
			startName = name
		}
	}

	accounts, err := h.state.ListAccounts()
	if err != nil {
		return nil, &rpcError{Code: -32603, Message: err.Error()}
	}

	results := make([]EnrichedAccountData, 0, args.Limit)
	for _, acc := range accounts {
		if acc.Name >= startName {
			results = append(results, enrichAccount(acc))
			if len(results) >= int(args.Limit) {
				break
			}
		}
	}
	return map[string]any{"accounts": results}, nil
}

func (h *RPCHandler) handleGetKeyReferences(params json.RawMessage) (any, *rpcError) {
	var objParams struct {
		Keys []string `json:"keys"`
	}
	var keys []string
	if err := json.Unmarshal(params, &objParams); err == nil && len(objParams.Keys) > 0 {
		keys = objParams.Keys
	} else {
		var outerParams [][]string
		if err := json.Unmarshal(params, &outerParams); err == nil && len(outerParams) > 0 {
			keys = outerParams[0]
		}
	}
	if len(keys) == 0 {
		return nil, &rpcError{Code: -32602, Message: "Invalid parameters"}
	}
	refs, err := h.state.GetKeyReferences(keys)
	if err != nil {
		return nil, &rpcError{Code: -32603, Message: err.Error()}
	}

	var results [][]string
	for _, ref := range refs {
		results = append(results, []string{ref})
	}
	for len(results) < len(keys) {
		results = append(results, []string{})
	}

	return results, nil
}
