package rpc

import (
	"encoding/json"
	"slices"
	"strings"
	"time"

	"github.com/srbde/hoverfly/state"
)

func (h *RPCHandler) handleFindRCAccounts(params json.RawMessage) (any, *rpcError) {
	var args struct {
		Accounts []string `json:"accounts"`
	}
	if err := json.Unmarshal(params, &args); err != nil || len(args.Accounts) == 0 {
		var arrArgs []any
		if err := json.Unmarshal(params, &arrArgs); err == nil && len(arrArgs) > 0 {
			if names, ok := arrArgs[0].([]any); ok {
				for _, n := range names {
					if s, ok := n.(string); ok {
						args.Accounts = append(args.Accounts, s)
					}
				}
			}
		}
	}

	if len(args.Accounts) == 0 {
		return nil, &rpcError{Code: -32602, Message: "Invalid parameters"}
	}

	type rcAccount struct {
		Account   string        `json:"account"`
		RcManabar state.Manabar `json:"rc_manabar"`
		MaxRC     string        `json:"max_rc"`
	}

	var results []rcAccount
	for _, name := range args.Accounts {
		results = append(results, rcAccount{
			Account: name,
			RcManabar: state.Manabar{
				CurrentMana:    16450459302631,
				LastUpdateTime: time.Now().Unix(),
			},
			MaxRC: "16450459302631",
		})
	}

	return map[string]any{
		"rc_accounts": results,
	}, nil
}

func (h *RPCHandler) handleListRCAccounts(method string, params json.RawMessage) (any, *rpcError) {
	var args struct {
		Start string `json:"start"`
		Limit int    `json:"limit"`
	}
	if err := json.Unmarshal(params, &args); err != nil {
		var arrArgs []struct {
			Start string `json:"start"`
			Limit int    `json:"limit"`
		}
		if err := json.Unmarshal(params, &arrArgs); err == nil && len(arrArgs) > 0 {
			args = arrArgs[0]
		}
	}
	limit := clampLimit(args.Limit, 1000)

	accounts, err := h.state.ListAccounts()
	if err != nil {
		return nil, &rpcError{Code: -32603, Message: err.Error()}
	}

	type rcAccount struct {
		Account                 string        `json:"account"`
		RcManabar               state.Manabar `json:"rc_manabar"`
		MaxRC                   string        `json:"max_rc"`
		DelegatedRC             int64         `json:"delegated_rc"`
		ReceivedDelegatedRC     int64         `json:"received_delegated_rc"`
		MaxRCCreationAdjustment any           `json:"max_rc_creation_adjustment"`
	}

	results := make([]rcAccount, 0, limit)
	for _, acc := range accounts {
		if args.Start != "" && acc.Name < args.Start {
			continue
		}
		results = append(results, rcAccount{
			Account: acc.Name,
			RcManabar: state.Manabar{
				CurrentMana:    16450459302631,
				LastUpdateTime: time.Now().Unix(),
			},
			MaxRC:               "16450459302631",
			DelegatedRC:         0,
			ReceivedDelegatedRC: 0,
			MaxRCCreationAdjustment: map[string]any{
				"amount":    "0",
				"precision": 6,
				"nai":       "@@000000037",
			},
		})
		if len(results) >= limit {
			break
		}
	}

	if method == "condenser_api.list_rc_accounts" {
		return results, nil
	}
	return map[string]any{"rc_accounts": results}, nil
}

func (h *RPCHandler) handleListRCDirectDelegations(method string) (any, *rpcError) {
	if method == "condenser_api.list_rc_direct_delegations" {
		return []any{}, nil
	}
	return map[string]any{"rc_direct_delegations": []any{}}, nil
}

func (h *RPCHandler) handleGetConfig() (any, *rpcError) {
	return map[string]any{
		"HIVE_BLOCKCHAIN_VERSION":  "1.28.6",
		"HIVE_BLOCKCHAIN_HARDFORK": 28,
	}, nil
}

func (h *RPCHandler) handleGetChainProperties() (any, *rpcError) {
	return map[string]any{
		"account_creation_fee":   "3.000 HIVE",
		"maximum_block_size":     65536,
		"hbd_interest_rate":      1500,
		"account_subsidy_budget": 797,
		"account_subsidy_decay":  347321,
	}, nil
}

func (h *RPCHandler) handleGetVersion() (any, *rpcError) {
	return map[string]any{
		"blockchain_version": "1.28.6",
		"hive_revision":      "hoverfly",
		"fc_revision":        "hoverfly",
		"haf_revision":       "hoverfly",
		"chain_id":           "beeab0de00000000000000000000000000000000000000000000000000000000",
		"node_type":          "testnet",
	}, nil
}

func (h *RPCHandler) handleLookupWitnessAccounts(params json.RawMessage) (any, *rpcError) {
	lowerBound := ""
	limit := 1000
	var args []any
	if err := json.Unmarshal(params, &args); err == nil && len(args) > 0 {
		lowerBound, _ = args[0].(string)
		if len(args) > 1 {
			if rawLimit, ok := args[1].(float64); ok && rawLimit > 0 {
				limit = int(rawLimit)
			}
		}
	}
	limit = clampLimit(limit, 1000)

	witnesses := activeWitnessNames()
	results := make([]string, 0, limit)
	for _, witness := range witnesses {
		if witness >= lowerBound {
			results = append(results, witness)
			if len(results) >= limit {
				break
			}
		}
	}
	return results, nil
}

func (h *RPCHandler) handleGetWitnessByAccount(params json.RawMessage) (any, *rpcError) {
	account := ""
	var args []string
	if err := json.Unmarshal(params, &args); err == nil && len(args) > 0 {
		account = args[0]
	}
	if account == "" {
		var objectArgs struct {
			Account string `json:"account"`
		}
		if err := json.Unmarshal(params, &objectArgs); err == nil {
			account = objectArgs.Account
		}
	}
	if account == "" {
		return nil, &rpcError{Code: -32602, Message: "Invalid parameters"}
	}

	if slices.Contains(activeWitnessNames(), account) {
		return mockWitness(account), nil
	}
	return json.RawMessage("null"), nil
}

func activeWitnessNames() []string {
	return []string{
		"abit", "arcange", "ausbitbank", "blocktrades", "deathwing", "emrebeyler",
		"good-karma", "gtg", "guiltyparties", "ocd-witness", "pharesim", "quochuy",
		"roelandp", "smooth.witness", "steempeak", "stoodkev", "themarkymark",
		"therealwolf", "threespeak", "yabapmatt", "anyx",
	}
}

func mockWitness(owner string) map[string]any {
	return map[string]any{
		"id":                       0,
		"owner":                    owner,
		"created":                  "2016-03-24T16:00:00",
		"url":                      "https://hoverfly.local/witness",
		"votes":                    "0",
		"virtual_last_update":      "0",
		"virtual_position":         "0",
		"virtual_scheduled_time":   "0",
		"total_missed":             0,
		"last_aslot":               0,
		"last_confirmed_block_num": 0,
		"signing_key":              "STM6ipXFLZyBeJRLFkXNRzAeQDz5T9zawSzYUdMShPsBHqB9W4SaC",
		"props": map[string]any{
			"account_creation_fee":   "3.000 HIVE",
			"maximum_block_size":     65536,
			"hbd_interest_rate":      1500,
			"account_subsidy_budget": 797,
			"account_subsidy_decay":  347321,
		},
		"hbd_exchange_rate": map[string]any{
			"base":  "0.200 HBD",
			"quote": "1.000 HIVE",
		},
	}
}

func (h *RPCHandler) handleProposalList(method string) (any, *rpcError) {
	if strings.HasPrefix(method, "database_api.") {
		return map[string]any{"proposals": []any{}}, nil
	}
	return []any{}, nil
}

func (h *RPCHandler) handleProposalVoteList(method string) (any, *rpcError) {
	if strings.HasPrefix(method, "database_api.") {
		return map[string]any{"proposal_votes": []any{}}, nil
	}
	return []any{}, nil
}

func (h *RPCHandler) handleEmptyDatabaseState(method string) (any, *rpcError) {
	field := "items"
	switch {
	case strings.Contains(method, "recovery"):
		field = "requests"
	case strings.Contains(method, "conversion"):
		field = "requests"
	case strings.Contains(method, "decline_voting"):
		field = "requests"
	case strings.Contains(method, "escrow"):
		field = "escrows"
	case strings.Contains(method, "limit_order"):
		field = "orders"
	case strings.Contains(method, "owner_histor"):
		field = "owner_auths"
	case strings.Contains(method, "recurrent_transfer"):
		field = "recurrent_transfers"
	case strings.Contains(method, "savings_withdraw"):
		field = "withdrawals"
	case strings.Contains(method, "vesting_delegation"):
		field = "delegations"
	case strings.Contains(method, "withdraw_vesting"):
		field = "routes"
	}
	return map[string]any{field: []any{}}, nil
}

func (h *RPCHandler) handleSimulateCurvePayouts(params json.RawMessage) (any, *rpcError) {
	var args struct {
		Curve string `json:"curve"`
		Var1  string `json:"var1"`
	}
	if err := json.Unmarshal(params, &args); err != nil {
		// Ignore parsing error
	}
	recentClaims := args.Var1
	if recentClaims == "" {
		recentClaims = "2000000000000"
	}
	return map[string]any{
		"recent_claims": recentClaims,
		"payouts":       []any{},
	}, nil
}
