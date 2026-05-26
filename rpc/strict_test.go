package rpc

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/srbde/hoverfly/state"
)

func TestStrictModeFailures(t *testing.T) {
	s, err := state.NewState("", false)
	if err != nil {
		t.Fatalf("failed to create state: %v", err)
	}
	defer s.Close()

	// Seed one active account "alice" with some initial funds
	alice := &state.AccountData{
		Name:          "alice",
		Balance:       "10.000 HIVE",
		HbdBalance:    "5.000 HBD",
		VestingShares: "1000000.000000 VESTS",
		VotingPower:   10000,
	}
	s.SaveAccount(alice)

	// Create strict handler
	handler := NewRPCHandler(s, false, true)

	t.Run("get_accounts strict - nonexistent account returns null", func(t *testing.T) {
		reqBody := `{"jsonrpc":"2.0","method":"condenser_api.get_accounts","params":[["alice","nonexistent"]],"id":1}`
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(reqBody))
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		var resp jsonRPCResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		results, ok := resp.Result.([]any)
		if !ok || len(results) != 2 {
			t.Fatalf("expected list of 2 results, got %v", resp.Result)
		}

		if results[0] == nil {
			t.Errorf("expected first account (alice) to not be nil")
		}
		if results[1] != nil {
			t.Errorf("expected second account (nonexistent) to be null in strict mode")
		}
	})

	t.Run("find_accounts strict - nonexistent account omitted", func(t *testing.T) {
		reqBody := `{"jsonrpc":"2.0","method":"database_api.find_accounts","params":{"accounts":["alice","nonexistent"]},"id":2}`
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(reqBody))
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		var resp jsonRPCResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		resMap, ok := resp.Result.(map[string]any)
		if !ok {
			t.Fatalf("expected map result, got %v", resp.Result)
		}

		accounts, ok := resMap["accounts"].([]any)
		if !ok || len(accounts) != 1 {
			t.Fatalf("expected 1 account returned, got %v", resMap["accounts"])
		}

		acc0 := accounts[0].(map[string]any)
		if acc0["name"] != "alice" {
			t.Errorf("expected account 'alice', got '%v'", acc0["name"])
		}
	})

	t.Run("get_content strict - nonexistent post returns zero post", func(t *testing.T) {
		reqBody := `{"jsonrpc":"2.0","method":"condenser_api.get_content","params":["alice","nonexistent-post"],"id":3}`
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(reqBody))
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		var resp jsonRPCResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		postMap, ok := resp.Result.(map[string]any)
		if !ok {
			t.Fatalf("expected map result, got %v", resp.Result)
		}

		if postMap["author"] != "" || postMap["permlink"] != "" {
			t.Errorf("expected empty post fields for nonexistent post in strict mode, got %v", postMap)
		}
	})

	t.Run("broadcast strict - nonexistent sender fails validation", func(t *testing.T) {
		reqBody := `{"jsonrpc":"2.0","method":"condenser_api.broadcast_transaction","params":[{"ref_block_num":100,"ref_block_prefix":123,"expiration":"2026-05-26T18:00:00","operations":[["transfer",{"from":"nonexistent","to":"alice","amount":"1.000 HIVE","memo":"fail test"}]]}],"id":4}`
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(reqBody))
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		var resp jsonRPCResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if resp.Error == nil {
			t.Fatalf("expected error response, got success: %v", resp.Result)
		}

		errMap := resp.Error.(map[string]any)
		msg := errMap["message"].(string)
		if msg == "" || !bytes.Contains([]byte(msg), []byte("sender account nonexistent does not exist")) {
			t.Errorf("expected nonexistent sender message, got '%s'", msg)
		}
	})

	t.Run("broadcast strict - nonexistent recipient fails validation", func(t *testing.T) {
		reqBody := `{"jsonrpc":"2.0","method":"condenser_api.broadcast_transaction","params":[{"ref_block_num":100,"ref_block_prefix":123,"expiration":"2026-05-26T18:00:00","operations":[["transfer",{"from":"alice","to":"nonexistent","amount":"1.000 HIVE","memo":"fail test"}]]}],"id":5}`
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(reqBody))
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		var resp jsonRPCResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if resp.Error == nil {
			t.Fatalf("expected error response, got success: %v", resp.Result)
		}

		errMap := resp.Error.(map[string]any)
		msg := errMap["message"].(string)
		if msg == "" || !bytes.Contains([]byte(msg), []byte("recipient account nonexistent does not exist")) {
			t.Errorf("expected nonexistent recipient message, got '%s'", msg)
		}
	})

	t.Run("broadcast strict - insufficient funds fails validation", func(t *testing.T) {
		// Create recipient "bob"
		bob := &state.AccountData{Name: "bob", Balance: "0.000 HIVE"}
		s.SaveAccount(bob)

		// alice has 10.000 HIVE, try to transfer 11.000 HIVE
		reqBody := `{"jsonrpc":"2.0","method":"condenser_api.broadcast_transaction","params":[{"ref_block_num":100,"ref_block_prefix":123,"expiration":"2026-05-26T18:00:00","operations":[["transfer",{"from":"alice","to":"bob","amount":"11.000 HIVE","memo":"fail test"}]]}],"id":6}`
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(reqBody))
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		var resp jsonRPCResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if resp.Error == nil {
			t.Fatalf("expected error response, got success: %v", resp.Result)
		}

		errMap := resp.Error.(map[string]any)
		msg := errMap["message"].(string)
		if msg == "" || !bytes.Contains([]byte(msg), []byte("insufficient funds")) {
			t.Errorf("expected insufficient funds message, got '%s'", msg)
		}
	})

	t.Run("broadcast strict - account_create creator nonexistent fails", func(t *testing.T) {
		reqBody := `{"jsonrpc":"2.0","method":"condenser_api.broadcast_transaction","params":[{"ref_block_num":100,"ref_block_prefix":123,"expiration":"2026-05-26T18:00:00","operations":[["account_create",{"fee":"3.000 HIVE","creator":"nonexistent","new_account_name":"carol","owner":{"weight_threshold":1,"account_auths":[],"key_auths":[["STM5Key1",1]]},"active":{"weight_threshold":1,"account_auths":[],"key_auths":[["STM5Key2",1]]},"posting":{"weight_threshold":1,"account_auths":[],"key_auths":[["STM5Key3",1]]},"memo_key":"STM5Key4","json_metadata":""}]]}],"id":7}`
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(reqBody))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		var resp jsonRPCResponse
		json.Unmarshal(w.Body.Bytes(), &resp)
		if resp.Error == nil {
			t.Fatalf("expected error, got success")
		}
		errMap := resp.Error.(map[string]any)
		msg := errMap["message"].(string)
		if !bytes.Contains([]byte(msg), []byte("creator account nonexistent does not exist")) {
			t.Errorf("expected creator nonexistent error, got: %s", msg)
		}
	})

	t.Run("broadcast strict - account_create account already exists fails", func(t *testing.T) {
		reqBody := `{"jsonrpc":"2.0","method":"condenser_api.broadcast_transaction","params":[{"ref_block_num":100,"ref_block_prefix":123,"expiration":"2026-05-26T18:00:00","operations":[["account_create",{"fee":"3.000 HIVE","creator":"alice","new_account_name":"bob","owner":{"weight_threshold":1,"account_auths":[],"key_auths":[["STM5Key1",1]]},"active":{"weight_threshold":1,"account_auths":[],"key_auths":[["STM5Key2",1]]},"posting":{"weight_threshold":1,"account_auths":[],"key_auths":[["STM5Key3",1]]},"memo_key":"STM5Key4","json_metadata":""}]]}],"id":8}`
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(reqBody))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		var resp jsonRPCResponse
		json.Unmarshal(w.Body.Bytes(), &resp)
		if resp.Error == nil {
			t.Fatalf("expected error, got success")
		}
		errMap := resp.Error.(map[string]any)
		msg := errMap["message"].(string)
		if !bytes.Contains([]byte(msg), []byte("account bob already exists")) {
			t.Errorf("expected already exists error, got: %s", msg)
		}
	})

	t.Run("broadcast strict - account_create insufficient funds fails", func(t *testing.T) {
		reqBody := `{"jsonrpc":"2.0","method":"condenser_api.broadcast_transaction","params":[{"ref_block_num":100,"ref_block_prefix":123,"expiration":"2026-05-26T18:00:00","operations":[["account_create",{"fee":"15.000 HIVE","creator":"alice","new_account_name":"carol","owner":{"weight_threshold":1,"account_auths":[],"key_auths":[["STM5Key1",1]]},"active":{"weight_threshold":1,"account_auths":[],"key_auths":[["STM5Key2",1]]},"posting":{"weight_threshold":1,"account_auths":[],"key_auths":[["STM5Key3",1]]},"memo_key":"STM5Key4","json_metadata":""}]]}],"id":9}`
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(reqBody))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		var resp jsonRPCResponse
		json.Unmarshal(w.Body.Bytes(), &resp)
		if resp.Error == nil {
			t.Fatalf("expected error, got success")
		}
		errMap := resp.Error.(map[string]any)
		msg := errMap["message"].(string)
		if !bytes.Contains([]byte(msg), []byte("insufficient funds for account creation fee")) {
			t.Errorf("expected insufficient funds error, got: %s", msg)
		}
	})

	t.Run("broadcast - account_create mutates state successfully", func(t *testing.T) {
		reqBody := `{"jsonrpc":"2.0","method":"condenser_api.broadcast_transaction","params":[{"ref_block_num":100,"ref_block_prefix":123,"expiration":"2026-05-26T18:00:00","operations":[["account_create",{"fee":"3.000 HIVE","creator":"alice","new_account_name":"carol","owner":{"weight_threshold":1,"account_auths":[],"key_auths":[["STM5OwnerKey",1]]},"active":{"weight_threshold":1,"account_auths":[],"key_auths":[["STM5ActiveKey",1]]},"posting":{"weight_threshold":1,"account_auths":[],"key_auths":[["STM5PostingKey",1]]},"memo_key":"STM5MemoKey","json_metadata":"{}"}]]}],"id":10}`
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(reqBody))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		var resp jsonRPCResponse
		json.Unmarshal(w.Body.Bytes(), &resp)
		if resp.Error != nil {
			t.Fatalf("unexpected error: %v", resp.Error)
		}

		// Verify carol exists in state
		carol, err := s.GetAccount("carol")
		if err != nil || carol == nil {
			t.Fatalf("expected carol to exist in database, got error: %v", err)
		}
		if carol.ActiveKey != "STM5ActiveKey" {
			t.Errorf("expected active key STM5ActiveKey, got %s", carol.ActiveKey)
		}
		if carol.PostingKey != "STM5PostingKey" {
			t.Errorf("expected posting key STM5PostingKey, got %s", carol.PostingKey)
		}

		// Verify alice's balance was deducted (10.000 HIVE -> 7.000 HIVE)
		aliceAcc, _ := s.GetAccount("alice")
		if aliceAcc.Balance != "7.000 HIVE" {
			t.Errorf("expected alice balance to be 7.000 HIVE, got %s", aliceAcc.Balance)
		}

		// Verify key references resolved
		refs, err := s.GetKeyReferences([]string{"STM5ActiveKey"})
		if err != nil {
			t.Fatalf("failed to get key references: %v", err)
		}
		if len(refs) != 1 || refs[0] != "carol" {
			t.Errorf("expected key references for STM5ActiveKey to be ['carol'], got %v", refs)
		}
	})
}
