package rpc

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/thecrazygm/hoverfly/crypto"
	"github.com/thecrazygm/hoverfly/state"
)

func (h *RPCHandler) handleBroadcastTransaction(params json.RawMessage) (any, *rpcError) {
	var tx crypto.Transaction
	var parsed bool

	// 1. Try array format: [tx]
	var arrayParams []crypto.Transaction
	if err := json.Unmarshal(params, &arrayParams); err == nil && len(arrayParams) > 0 {
		tx = arrayParams[0]
		parsed = true
	}

	// 2. Try object format: {"trx": tx}
	if !parsed {
		var objectParams struct {
			Trx crypto.Transaction `json:"trx"`
		}
		if err := json.Unmarshal(params, &objectParams); err == nil && objectParams.Trx.RefBlockNum != 0 {
			tx = objectParams.Trx
			parsed = true
		}
	}

	if !parsed {
		return nil, &rpcError{Code: -32602, Message: "Invalid parameters"}
	}

	if len(tx.Signatures) > 0 {
		chainID := "0000000000000000000000000000000000000000000000000000000000000000"
		recoveredKeys, err := crypto.VerifySignatures(&tx, chainID)
		if err != nil {
			testnetChainID := "beeab30de373dca1e2f036c30d4970470d0d57d055748a30de53070470d0d57d"
			recoveredKeys, err = crypto.VerifySignatures(&tx, testnetChainID)
			if err != nil {
				log.Warnf("Transaction signature verification FAILED: %v", err)
				return nil, &rpcError{Code: -32000, Message: fmt.Sprintf("signature verification failed: %v", err)}
			}
		}
		log.Infof("Transaction verified successfully. Recovered signing key(s): %v", recoveredKeys)
	} else {
		log.Warn("Transaction has no signatures; skipping verification in mock server (permissive mode)")
	}

	for _, rawOp := range tx.Operations {
		var tuple []json.RawMessage
		if err := json.Unmarshal(rawOp, &tuple); err == nil && len(tuple) == 2 {
			var opName string
			json.Unmarshal(tuple[0], &opName)

			switch opName {
			case "transfer":
				var op struct {
					From   string `json:"from"`
					To     string `json:"to"`
					Amount string `json:"amount"`
					Memo   string `json:"memo"`
				}
				if err := json.Unmarshal(tuple[1], &op); err == nil {
					h.mutateTransfer(op.From, op.To, op.Amount)
				}

			case "transfer_to_savings":
				var op struct {
					From   string `json:"from"`
					To     string `json:"to"`
					Amount string `json:"amount"`
					Memo   string `json:"memo"`
				}
				if err := json.Unmarshal(tuple[1], &op); err == nil {
					h.mutateTransferToSavings(op.From, op.To, op.Amount)
				}

			case "comment":
				var op struct {
					Author         string `json:"author"`
					Permlink       string `json:"permlink"`
					ParentAuthor   string `json:"parent_author"`
					ParentPermlink string `json:"parent_permlink"`
					Category       string `json:"category"`
					Title          string `json:"title"`
					Body           string `json:"body"`
					JSONMetadata   string `json:"json_metadata"`
				}
				if err := json.Unmarshal(tuple[1], &op); err == nil {
					h.mutateComment(op.Author, op.Permlink, op.ParentAuthor, op.ParentPermlink, op.Category, op.Title, op.Body, op.JSONMetadata)
				}
			}
		}
	}

	txBytes, _ := tx.Serialize()
	hash := sha256.Sum256(txBytes)
	txID := hex.EncodeToString(hash[:20])

	props, _ := h.state.GetDynamicProperties()
	blockNum := uint32(100000001)
	if props != nil {
		blockNum = props.HeadBlockNumber + 1
	}

	// Save transaction to state for later polling (get_transaction)
	var ops []any
	for _, rawOp := range tx.Operations {
		var op []any
		if err := json.Unmarshal(rawOp, &op); err == nil {
			ops = append(ops, op)
		}
	}

	h.state.SaveTransaction(&state.TransactionData{
		TransactionID:  txID,
		BlockNum:       blockNum,
		TransactionNum: 1,
		RefBlockNum:    tx.RefBlockNum,
		RefBlockPrefix: tx.RefBlockPrefix,
		Expiration:     tx.Expiration,
		Operations:     ops,
		Extensions:     tx.Extensions,
		Signatures:     tx.Signatures,
	})

	return map[string]any{
		"id":        txID,
		"block_num": blockNum,
		"trx_num":   1,
		"expired":   false,
	}, nil
}

func (h *RPCHandler) handleGetTransactionHex(method string, params json.RawMessage) (any, *rpcError) {
	var tx crypto.Transaction
	if method == "condenser_api.get_transaction_hex" {
		var args []crypto.Transaction
		if err := json.Unmarshal(params, &args); err != nil || len(args) == 0 {
			return nil, &rpcError{Code: -32602, Message: "Invalid parameters"}
		}
		tx = args[0]
	} else {
		var args struct {
			Trx crypto.Transaction `json:"trx"`
		}
		if err := json.Unmarshal(params, &args); err != nil {
			var wrapped []struct {
				Trx crypto.Transaction `json:"trx"`
			}
			if err := json.Unmarshal(params, &wrapped); err != nil || len(wrapped) == 0 {
				return nil, &rpcError{Code: -32602, Message: "Invalid parameters"}
			}
			args = wrapped[0]
		}
		tx = args.Trx
	}

	bytes, err := tx.Serialize()
	if err != nil {
		return nil, &rpcError{Code: -32602, Message: err.Error()}
	}

	hexValue := hex.EncodeToString(bytes)
	if method == "database_api.get_transaction_hex" {
		return map[string]any{"hex": hexValue}, nil
	}
	return hexValue, nil
}

func (h *RPCHandler) handlePotentialSignatures(method string, params json.RawMessage) (any, *rpcError) {
	keys := extractAvailableKeys(params)
	if method == "database_api.get_potential_signatures" {
		return map[string]any{"keys": keys}, nil
	}
	return keys, nil
}

func (h *RPCHandler) handleRequiredSignatures(method string, params json.RawMessage) (any, *rpcError) {
	keys := extractAvailableKeys(params)
	if method == "database_api.get_required_signatures" {
		return map[string]any{"keys": keys}, nil
	}
	return keys, nil
}

func extractAvailableKeys(params json.RawMessage) []string {
	var args struct {
		AvailableKeys []string `json:"available_keys"`
		Keys          []string `json:"keys"`
	}
	if err := json.Unmarshal(params, &args); err == nil {
		if len(args.AvailableKeys) > 0 {
			return args.AvailableKeys
		}
		return args.Keys
	}

	var arrArgs []any
	if err := json.Unmarshal(params, &arrArgs); err == nil {
		for _, arg := range arrArgs {
			if keyList, ok := arg.([]any); ok {
				keys := make([]string, 0, len(keyList))
				for _, rawKey := range keyList {
					if key, ok := rawKey.(string); ok {
						keys = append(keys, key)
					}
				}
				if len(keys) > 0 {
					return keys
				}
			}
		}
	}
	return []string{}
}

func (h *RPCHandler) handleVerifyAuthority(method string, params json.RawMessage) (any, *rpcError) {
	valid := true
	if method == "database_api.verify_account_authority" {
		var args struct {
			Account string   `json:"account"`
			Keys    []string `json:"keys"`
		}
		if err := json.Unmarshal(params, &args); err == nil && args.Account != "" {
			valid = len(args.Keys) > 0
		}
	}
	if strings.HasPrefix(method, "database_api.") {
		return map[string]any{"valid": valid}, nil
	}
	return valid, nil
}

func (h *RPCHandler) mutateTransfer(from, to, amountStr string) {
	parts := strings.Fields(amountStr)
	if len(parts) != 2 {
		return
	}
	var val float64
	fmt.Sscanf(parts[0], "%f", &val)
	symbol := parts[1]

	updateBal := func(name string, add float64) {
		acc, err := h.state.GetAccount(name)
		if err != nil {
			acc = &state.AccountData{
				Name:        name,
				VotingPower: 10000,
				VotingManabar: state.Manabar{
					CurrentMana:    10000,
					LastUpdateTime: time.Now().Unix(),
				},
				LastVoteTime:  "1970-01-01T00:00:00",
				Balance:       "0.000 HIVE",
				HbdBalance:    "0.000 HBD",
				VestingShares: "0.000000 VESTS",
				Created:       time.Now().UTC().Format("2006-01-02T15:04:05"),
			}
		}

		if symbol == "HIVE" {
			var current float64
			fmt.Sscanf(acc.Balance, "%f", &current)
			acc.Balance = fmt.Sprintf("%.3f HIVE", current+add)
		} else if symbol == "HBD" {
			var current float64
			fmt.Sscanf(acc.HbdBalance, "%f", &current)
			acc.HbdBalance = fmt.Sprintf("%.3f HBD", current+add)
		}

		h.state.SaveAccount(acc)
	}

	updateBal(from, -val)
	updateBal(to, val)
	log.Infof("State Mutated (Transfer): %s -> %s (%s)", from, to, amountStr)
}

func (h *RPCHandler) mutateTransferToSavings(from, to, amountStr string) {
	parts := strings.Fields(amountStr)
	if len(parts) != 2 {
		return
	}
	var val float64
	fmt.Sscanf(parts[0], "%f", &val)
	symbol := parts[1]

	// Deduct from sender's liquid balance
	accFrom, err := h.state.GetAccount(from)
	if err == nil && accFrom != nil {
		if symbol == "HIVE" {
			var current float64
			fmt.Sscanf(accFrom.Balance, "%f", &current)
			accFrom.Balance = fmt.Sprintf("%.3f HIVE", current-val)
		} else if symbol == "HBD" {
			var current float64
			fmt.Sscanf(accFrom.HbdBalance, "%f", &current)
			accFrom.HbdBalance = fmt.Sprintf("%.3f HBD", current-val)
		}
		h.state.SaveAccount(accFrom)
	}

	// Add to receiver's savings balance
	accTo, err := h.state.GetAccount(to)
	if err != nil {
		accTo = &state.AccountData{
			Name:        to,
			VotingPower: 10000,
			VotingManabar: state.Manabar{
				CurrentMana:    10000,
				LastUpdateTime: time.Now().Unix(),
			},
			LastVoteTime:  "1970-01-01T00:00:00",
			Balance:       "0.000 HIVE",
			HbdBalance:    "0.000 HBD",
			VestingShares: "0.000000 VESTS",
			Created:       time.Now().UTC().Format("2006-01-02T15:04:05"),
		}
	}

	if accTo.SavingsBalance == "" {
		accTo.SavingsBalance = "0.000 HIVE"
	}
	if accTo.SavingsHbdBalance == "" {
		accTo.SavingsHbdBalance = "0.000 HBD"
	}

	if symbol == "HIVE" {
		var current float64
		fmt.Sscanf(accTo.SavingsBalance, "%f", &current)
		accTo.SavingsBalance = fmt.Sprintf("%.3f HIVE", current+val)
	} else if symbol == "HBD" {
		var current float64
		fmt.Sscanf(accTo.SavingsHbdBalance, "%f", &current)
		accTo.SavingsHbdBalance = fmt.Sprintf("%.3f HBD", current+val)
	}
	h.state.SaveAccount(accTo)

	log.Infof("State Mutated (Transfer to Savings): %s -> %s (%s)", from, to, amountStr)
}

func (h *RPCHandler) mutateComment(author, permlink, parentAuthor, parentPermlink, category, title, body, jsonMeta string) {
	post := &state.PostData{
		Author:         author,
		Permlink:       permlink,
		ParentAuthor:   parentAuthor,
		ParentPermlink: parentPermlink,
		Category:       category,
		Title:          title,
		Body:           body,
		JSONMetadata:   jsonMeta,
		Created:        time.Now().UTC().Format("2006-01-02T15:04:05"),
		ActiveVotes:    []string{},
	}
	h.state.SaveContent(post)
	log.Infof("State Mutated (Comment): @%s/%s", author, permlink)
}
