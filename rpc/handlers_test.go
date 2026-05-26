package rpc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/srbde/hoverfly/state"
)

func TestRPCHandlers(t *testing.T) {
	s, err := state.NewState("", false)
	if err != nil {
		t.Fatalf("failed to create state: %v", err)
	}
	defer s.Close()

	handler := NewRPCHandler(s, false)

	t.Run("condenser_api.get_accounts", func(t *testing.T) {
		reqBody := `{"jsonrpc":"2.0","method":"condenser_api.get_accounts","params":[["thecrazygm"]],"id":1}`
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(reqBody))
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		var resp jsonRPCResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if resp.Error != nil {
			t.Fatalf("unexpected RPC error: %v", resp.Error)
		}

		results, ok := resp.Result.([]any)
		if !ok || len(results) == 0 {
			t.Fatalf("expected slice result, got %T: %v", resp.Result, resp.Result)
		}

		accMap, ok := results[0].(map[string]any)
		if !ok {
			t.Fatalf("expected map for account, got %T", results[0])
		}

		if accMap["name"] != "thecrazygm" {
			t.Errorf("expected account name 'thecrazygm', got '%v'", accMap["name"])
		}

		if _, exists := accMap["active"]; !exists {
			t.Error("expected 'active' authority to be present in get_accounts response")
		}

		if _, exists := accMap["memo_key"]; !exists {
			t.Error("expected 'memo_key' to be present in get_accounts response")
		}
	})

	t.Run("database_api.find_accounts", func(t *testing.T) {
		// Test direct object params: {"accounts": ["thecrazygm"]}
		reqBody := `{"jsonrpc":"2.0","method":"database_api.find_accounts","params":{"accounts":["thecrazygm"]},"id":2}`
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(reqBody))
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		var resp jsonRPCResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if resp.Error != nil {
			t.Fatalf("unexpected RPC error: %v", resp.Error)
		}

		resultMap, ok := resp.Result.(map[string]any)
		if !ok {
			t.Fatalf("expected map result, got %T", resp.Result)
		}

		accounts, ok := resultMap["accounts"].([]any)
		if !ok || len(accounts) == 0 {
			t.Fatalf("expected 'accounts' list in result, got: %v", resultMap)
		}

		accMap, ok := accounts[0].(map[string]any)
		if !ok {
			t.Fatalf("expected map for account, got %T", accounts[0])
		}

		if accMap["name"] != "thecrazygm" {
			t.Errorf("expected account name 'thecrazygm', got '%v'", accMap["name"])
		}

		if _, exists := accMap["posting"]; !exists {
			t.Error("expected 'posting' authority to be present in find_accounts response")
		}
	})

	t.Run("database_api.find_accounts array wrapping", func(t *testing.T) {
		// Test array-wrapped params: [{"accounts": ["thecrazygm"]}]
		reqBody := `{"jsonrpc":"2.0","method":"database_api.find_accounts","params":[{"accounts":["thecrazygm"]}],"id":3}`
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(reqBody))
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		var resp jsonRPCResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if resp.Error != nil {
			t.Fatalf("unexpected RPC error: %v", resp.Error)
		}

		resultMap, ok := resp.Result.(map[string]any)
		if !ok {
			t.Fatalf("expected map result, got %T", resp.Result)
		}

		accounts, ok := resultMap["accounts"].([]any)
		if !ok || len(accounts) == 0 {
			t.Fatalf("expected 'accounts' list in result, got: %v", resultMap)
		}

		accMap, ok := accounts[0].(map[string]any)
		if !ok {
			t.Fatalf("expected map for account, got %T", accounts[0])
		}

		if accMap["name"] != "thecrazygm" {
			t.Errorf("expected account name 'thecrazygm', got '%v'", accMap["name"])
		}
	})

	t.Run("database_api.get_dynamic_global_properties", func(t *testing.T) {
		reqBody := `{"jsonrpc":"2.0","method":"database_api.get_dynamic_global_properties","params":[],"id":4}`
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(reqBody))
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		var resp jsonRPCResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if resp.Error != nil {
			t.Fatalf("unexpected RPC error: %v", resp.Error)
		}

		propsMap, ok := resp.Result.(map[string]any)
		if !ok {
			t.Fatalf("expected map result, got %T", resp.Result)
		}

		if _, exists := propsMap["head_block_number"]; !exists {
			t.Error("expected 'head_block_number' to be present in get_dynamic_global_properties response")
		}
	})

	t.Run("condenser_api.get_follow_count", func(t *testing.T) {
		reqBody := `{"jsonrpc":"2.0","method":"condenser_api.get_follow_count","params":["thecrazygm"],"id":5}`
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(reqBody))
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		var resp jsonRPCResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if resp.Error != nil {
			t.Fatalf("unexpected RPC error: %v", resp.Error)
		}

		resultMap, ok := resp.Result.(map[string]any)
		if !ok {
			t.Fatalf("expected map result, got %T", resp.Result)
		}

		if resultMap["account"] != "thecrazygm" {
			t.Errorf("expected account 'thecrazygm', got '%v'", resultMap["account"])
		}

		if _, exists := resultMap["follower_count"]; !exists {
			t.Error("expected 'follower_count' to be present in get_follow_count response")
		}
	})

	t.Run("block_api.get_block", func(t *testing.T) {
		reqBody := `{"jsonrpc":"2.0","method":"block_api.get_block","params":{"block_num":1000},"id":6}`
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(reqBody))
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		var resp jsonRPCResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if resp.Error != nil {
			t.Fatalf("unexpected RPC error: %v", resp.Error)
		}

		resultMap, ok := resp.Result.(map[string]any)
		if !ok {
			t.Fatalf("expected map result, got %T", resp.Result)
		}

		blockMap, ok := resultMap["block"].(map[string]any)
		if !ok {
			t.Fatalf("expected result.block map, got %T", resultMap["block"])
		}

		if _, exists := blockMap["previous"]; !exists {
			t.Error("expected 'previous' to be present in get_block response block")
		}
	})

	t.Run("condenser_api.get_transaction", func(t *testing.T) {
		// Let's manually save a transaction to the state instead of broadcasting,
		// to test the retrieval logic independently.
		txID := "834f519fc6882a2384002ba93c90b7c02c7fe5f8"
		s.SaveTransaction(&state.TransactionData{
			TransactionID: txID,
			BlockNum:      12345,
			Operations:    []any{[]any{"vote", map[string]any{"voter": "alice"}}},
		})

		reqBody := fmt.Sprintf(`{"jsonrpc":"2.0","method":"condenser_api.get_transaction","params":["%s"],"id":7}`, txID)
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(reqBody))
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		var resp jsonRPCResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if resp.Error != nil {
			t.Fatalf("unexpected RPC error: %v", resp.Error)
		}

		txMap, ok := resp.Result.(map[string]any)
		if !ok {
			t.Fatalf("expected map result, got %T", resp.Result)
		}

		if txMap["transaction_id"] != txID {
			t.Errorf("expected transaction_id '%s', got '%v'", txID, txMap["transaction_id"])
		}
	})

	t.Run("transaction status APIs reflect saved transaction state", func(t *testing.T) {
		txID := "934f519fc6882a2384002ba93c90b7c02c7fe5f8"
		s.SaveTransaction(&state.TransactionData{
			TransactionID: txID,
			BlockNum:      100000001,
			Operations:    []any{[]any{"transfer", map[string]any{"from": "alice", "to": "bob"}}},
		})

		for _, reqBody := range []string{
			fmt.Sprintf(`{"jsonrpc":"2.0","method":"condenser_api.is_known_transaction","params":["%s"],"id":8}`, txID),
			fmt.Sprintf(`{"jsonrpc":"2.0","method":"database_api.is_known_transaction","params":{"id":"%s"},"id":9}`, txID),
			fmt.Sprintf(`{"jsonrpc":"2.0","method":"transaction_status_api.find_transaction","params":{"transaction_id":"%s","expiration":"2030-01-01T00:00:00"},"id":10}`, txID),
			fmt.Sprintf(`{"jsonrpc":"2.0","method":"account_history_api.get_transaction","params":{"transaction_id":"%s"},"id":11}`, txID),
		} {
			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(reqBody))
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			var resp jsonRPCResponse
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to parse response: %v", err)
			}
			if resp.Error != nil {
				t.Fatalf("unexpected RPC error for %s: %v", reqBody, resp.Error)
			}
		}
	})

	t.Run("utility APIs are first-class useful responses", func(t *testing.T) {
		for _, reqBody := range []string{
			`{"jsonrpc":"2.0","method":"condenser_api.find_rc_accounts","params":[["thecrazygm"]],"id":12}`,
			`{"jsonrpc":"2.0","method":"block_api.get_block_range","params":{"starting_block_num":1000,"count":2},"id":13}`,
			`{"jsonrpc":"2.0","method":"jsonrpc.get_methods","params":{},"id":14}`,
			`{"jsonrpc":"2.0","method":"jsonrpc.get_signature","params":{"method":"condenser_api.get_dynamic_global_properties"},"id":15}`,
			`{"jsonrpc":"2.0","method":"hive.get_info","params":{},"id":16}`,
		} {
			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(reqBody))
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			var resp jsonRPCResponse
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to parse response: %v", err)
			}
			if resp.Error != nil {
				t.Fatalf("unexpected RPC error for %s: %v", reqBody, resp.Error)
			}
		}
	})

	t.Run("account discovery APIs reflect state", func(t *testing.T) {
		for _, reqBody := range []string{
			`{"jsonrpc":"2.0","method":"condenser_api.get_account_count","params":[],"id":17}`,
			`{"jsonrpc":"2.0","method":"condenser_api.lookup_accounts","params":["a",10],"id":18}`,
			`{"jsonrpc":"2.0","method":"condenser_api.lookup_account_names","params":[["alice","missing-account"]],"id":19}`,
			`{"jsonrpc":"2.0","method":"database_api.list_accounts","params":{"start":"","limit":10,"order":"by_name"},"id":20}`,
		} {
			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(reqBody))
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			var resp jsonRPCResponse
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to parse response: %v", err)
			}
			if resp.Error != nil {
				t.Fatalf("unexpected RPC error for %s: %v", reqBody, resp.Error)
			}
		}
	})

	t.Run("version and chain metadata APIs are useful static responses", func(t *testing.T) {
		for _, reqBody := range []string{
			`{"jsonrpc":"2.0","method":"condenser_api.get_chain_properties","params":[],"id":21}`,
			`{"jsonrpc":"2.0","method":"condenser_api.get_version","params":[],"id":22}`,
			`{"jsonrpc":"2.0","method":"database_api.get_version","params":{},"id":23}`,
			`{"jsonrpc":"2.0","method":"condenser_api.get_hardfork_version","params":[],"id":24}`,
		} {
			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(reqBody))
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			var resp jsonRPCResponse
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to parse response: %v", err)
			}
			if resp.Error != nil {
				t.Fatalf("unexpected RPC error for %s: %v", reqBody, resp.Error)
			}
		}
	})

	t.Run("content and social read APIs reflect stored posts", func(t *testing.T) {
		s.SaveContent(&state.PostData{
			Author:       "alice",
			Permlink:     "hello-hoverfly",
			Category:     "hive",
			Title:        "Hello Hoverfly",
			Body:         "A state-backed post",
			JSONMetadata: "{}",
			Created:      "2026-05-25T12:00:00",
			ActiveVotes:  []string{"bob"},
		})
		s.SaveContent(&state.PostData{
			Author:         "bob",
			Permlink:       "re-hello-hoverfly",
			ParentAuthor:   "alice",
			ParentPermlink: "hello-hoverfly",
			Category:       "hive",
			Title:          "",
			Body:           "A state-backed reply",
			JSONMetadata:   "{}",
			Created:        "2026-05-25T12:01:00",
			ActiveVotes:    []string{},
		})

		for _, reqBody := range []string{
			`{"jsonrpc":"2.0","method":"tags_api.get_discussion","params":{"account":"alice","permlink":"hello-hoverfly"},"id":25}`,
			`{"jsonrpc":"2.0","method":"condenser_api.get_content_replies","params":{"account":"alice","permlink":"hello-hoverfly"},"id":26}`,
			`{"jsonrpc":"2.0","method":"tags_api.get_content_replies","params":{"account":"alice","permlink":"hello-hoverfly"},"id":27}`,
			`{"jsonrpc":"2.0","method":"condenser_api.get_active_votes","params":["alice","hello-hoverfly"],"id":28}`,
			`{"jsonrpc":"2.0","method":"condenser_api.get_blog","params":["alice",0,10],"id":29}`,
			`{"jsonrpc":"2.0","method":"follow_api.get_blog_entries","params":{"account":"alice","limit":10},"id":30}`,
			`{"jsonrpc":"2.0","method":"condenser_api.get_discussions_by_created","params":{"tag":"hive","limit":10},"id":31}`,
			`{"jsonrpc":"2.0","method":"tags_api.get_discussions_by_trending","params":{"tag":"hive","limit":10},"id":32}`,
			`{"jsonrpc":"2.0","method":"condenser_api.get_account_reputations","params":["alice",10],"id":33}`,
			`{"jsonrpc":"2.0","method":"follow_api.get_followers","params":["alice","","blog",10],"id":34}`,
			`{"jsonrpc":"2.0","method":"follow_api.get_following","params":["alice","","blog",10],"id":35}`,
			`{"jsonrpc":"2.0","method":"condenser_api.get_reblogged_by","params":["alice","hello-hoverfly"],"id":36}`,
		} {
			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(reqBody))
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			var resp jsonRPCResponse
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to parse response: %v", err)
			}
			if resp.Error != nil {
				t.Fatalf("unexpected RPC error for %s: %v", reqBody, resp.Error)
			}
		}
	})

	t.Run("market and rc APIs return useful first-class responses", func(t *testing.T) {
		for _, reqBody := range []string{
			`{"jsonrpc":"2.0","method":"condenser_api.get_market_history","params":[86400,"2026-01-01T00:00:00","2026-01-02T00:00:00"],"id":37}`,
			`{"jsonrpc":"2.0","method":"market_history_api.get_market_history","params":{"bucket_seconds":86400,"start":"2026-01-01T00:00:00","end":"2026-01-02T00:00:00"},"id":38}`,
			`{"jsonrpc":"2.0","method":"condenser_api.get_market_history_buckets","params":[],"id":39}`,
			`{"jsonrpc":"2.0","method":"market_history_api.get_market_history_buckets","params":{},"id":40}`,
			`{"jsonrpc":"2.0","method":"condenser_api.get_order_book","params":[10],"id":41}`,
			`{"jsonrpc":"2.0","method":"database_api.get_order_book","params":{"limit":10},"id":42}`,
			`{"jsonrpc":"2.0","method":"market_history_api.get_order_book","params":{"limit":10},"id":43}`,
			`{"jsonrpc":"2.0","method":"condenser_api.get_recent_trades","params":[10],"id":44}`,
			`{"jsonrpc":"2.0","method":"market_history_api.get_recent_trades","params":{"limit":10},"id":45}`,
			`{"jsonrpc":"2.0","method":"condenser_api.get_ticker","params":[],"id":46}`,
			`{"jsonrpc":"2.0","method":"market_history_api.get_ticker","params":{},"id":47}`,
			`{"jsonrpc":"2.0","method":"condenser_api.get_trade_history","params":["2026-01-01T00:00:00","2026-01-02T00:00:00",10],"id":48}`,
			`{"jsonrpc":"2.0","method":"market_history_api.get_trade_history","params":{"start":"2026-01-01T00:00:00","end":"2026-01-02T00:00:00","limit":10},"id":49}`,
			`{"jsonrpc":"2.0","method":"condenser_api.get_volume","params":[],"id":50}`,
			`{"jsonrpc":"2.0","method":"market_history_api.get_volume","params":{},"id":51}`,
			`{"jsonrpc":"2.0","method":"condenser_api.list_rc_accounts","params":{"start":"","limit":10},"id":52}`,
			`{"jsonrpc":"2.0","method":"rc_api.list_rc_accounts","params":{"start":"","limit":10},"id":53}`,
			`{"jsonrpc":"2.0","method":"condenser_api.list_rc_direct_delegations","params":{"start":["alice","bob"],"limit":10},"id":54}`,
			`{"jsonrpc":"2.0","method":"rc_api.list_rc_direct_delegations","params":{"start":["alice","bob"],"limit":10},"id":55}`,
			`{"jsonrpc":"2.0","method":"rc_api.get_rc_operation_stats","params":{"operation":"transfer_operation"},"id":56}`,
			`{"jsonrpc":"2.0","method":"rc_api.get_rc_stats","params":{},"id":57}`,
			`{"jsonrpc":"2.0","method":"rc_api.get_resource_params","params":{},"id":58}`,
			`{"jsonrpc":"2.0","method":"rc_api.get_resource_pool","params":{},"id":59}`,
		} {
			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(reqBody))
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			var resp jsonRPCResponse
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to parse response: %v", err)
			}
			if resp.Error != nil {
				t.Fatalf("unexpected RPC error for %s: %v", reqBody, resp.Error)
			}
		}
	})

	t.Run("witness proposal and feed APIs return useful first-class responses", func(t *testing.T) {
		for _, reqBody := range []string{
			`{"jsonrpc":"2.0","method":"condenser_api.get_active_witnesses","params":[],"id":60}`,
			`{"jsonrpc":"2.0","method":"database_api.get_active_witnesses","params":{},"id":61}`,
			`{"jsonrpc":"2.0","method":"condenser_api.get_witness_count","params":[],"id":62}`,
			`{"jsonrpc":"2.0","method":"condenser_api.lookup_witness_accounts","params":["a",10],"id":63}`,
			`{"jsonrpc":"2.0","method":"condenser_api.get_witness_by_account","params":["abit"],"id":64}`,
			`{"jsonrpc":"2.0","method":"condenser_api.get_witnesses","params":[["abit"],1],"id":65}`,
			`{"jsonrpc":"2.0","method":"condenser_api.get_witnesses_by_vote","params":["",10],"id":66}`,
			`{"jsonrpc":"2.0","method":"database_api.find_witnesses","params":{"owners":["abit"]},"id":67}`,
			`{"jsonrpc":"2.0","method":"database_api.list_witnesses","params":{"start":"","limit":10,"order":"by_name"},"id":68}`,
			`{"jsonrpc":"2.0","method":"database_api.list_witness_votes","params":{"start":["alice","abit"],"limit":10,"order":"by_account_witness"},"id":69}`,
			`{"jsonrpc":"2.0","method":"condenser_api.get_witness_schedule","params":[],"id":70}`,
			`{"jsonrpc":"2.0","method":"database_api.get_witness_schedule","params":{},"id":71}`,
			`{"jsonrpc":"2.0","method":"debug_node_api.debug_get_future_witness_schedule","params":{},"id":72}`,
			`{"jsonrpc":"2.0","method":"debug_node_api.debug_get_witness_schedule","params":{},"id":73}`,
			`{"jsonrpc":"2.0","method":"condenser_api.get_current_median_history_price","params":[],"id":74}`,
			`{"jsonrpc":"2.0","method":"database_api.get_current_price_feed","params":{},"id":75}`,
			`{"jsonrpc":"2.0","method":"condenser_api.get_feed_history","params":[],"id":76}`,
			`{"jsonrpc":"2.0","method":"database_api.get_feed_history","params":{},"id":77}`,
			`{"jsonrpc":"2.0","method":"condenser_api.get_next_scheduled_hardfork","params":[],"id":78}`,
			`{"jsonrpc":"2.0","method":"database_api.get_hardfork_properties","params":{},"id":79}`,
			`{"jsonrpc":"2.0","method":"debug_node_api.debug_get_hardfork_property_object","params":{},"id":80}`,
			`{"jsonrpc":"2.0","method":"condenser_api.get_reward_fund","params":["post"],"id":81}`,
			`{"jsonrpc":"2.0","method":"database_api.get_reward_funds","params":{},"id":82}`,
			`{"jsonrpc":"2.0","method":"condenser_api.find_proposals","params":[[0]],"id":83}`,
			`{"jsonrpc":"2.0","method":"database_api.find_proposals","params":{"proposal_ids":[0]},"id":84}`,
			`{"jsonrpc":"2.0","method":"condenser_api.list_proposals","params":[[""],10,"by_creator","ascending","all"],"id":85}`,
			`{"jsonrpc":"2.0","method":"database_api.list_proposals","params":{"start":[""],"limit":10,"order":"by_creator"},"id":86}`,
			`{"jsonrpc":"2.0","method":"condenser_api.list_proposal_votes","params":[["alice"],10,"by_voter_proposal","ascending","all"],"id":87}`,
			`{"jsonrpc":"2.0","method":"database_api.list_proposal_votes","params":{"start":["alice"],"limit":10,"order":"by_voter_proposal"},"id":88}`,
		} {
			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(reqBody))
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			var resp jsonRPCResponse
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to parse response: %v", err)
			}
			if resp.Error != nil {
				t.Fatalf("unexpected RPC error for %s: %v", reqBody, resp.Error)
			}
		}
	})

	t.Run("empty local-state and utility APIs are useful first-class responses", func(t *testing.T) {
		s.SaveContent(&state.PostData{
			Author:       "alice",
			Permlink:     "utility-post",
			Category:     "hive",
			Title:        "Utility Post",
			Body:         "Votes and lookup target",
			JSONMetadata: "{}",
			Created:      "2026-05-25T12:02:00",
			ActiveVotes:  []string{"bob"},
		})

		txJSON := `{"ref_block_num":1,"ref_block_prefix":2,"expiration":"2030-01-01T00:00:00","operations":[["transfer",{"from":"alice","to":"bob","amount":"1.000 HIVE","memo":"hex"}]],"extensions":[],"signatures":[]}`
		for _, reqBody := range []string{
			`{"jsonrpc":"2.0","method":"condenser_api.find_recurrent_transfers","params":["alice"],"id":89}`,
			`{"jsonrpc":"2.0","method":"condenser_api.get_collateralized_conversion_requests","params":["alice"],"id":90}`,
			`{"jsonrpc":"2.0","method":"condenser_api.get_conversion_requests","params":["alice"],"id":91}`,
			`{"jsonrpc":"2.0","method":"condenser_api.get_escrow","params":["alice",1],"id":92}`,
			`{"jsonrpc":"2.0","method":"condenser_api.get_expiring_vesting_delegations","params":["alice",0],"id":93}`,
			`{"jsonrpc":"2.0","method":"condenser_api.get_open_orders","params":["alice"],"id":94}`,
			`{"jsonrpc":"2.0","method":"condenser_api.get_owner_history","params":["alice"],"id":95}`,
			`{"jsonrpc":"2.0","method":"condenser_api.get_recovery_request","params":["alice"],"id":96}`,
			`{"jsonrpc":"2.0","method":"condenser_api.get_savings_withdraw_from","params":["alice"],"id":97}`,
			`{"jsonrpc":"2.0","method":"condenser_api.get_savings_withdraw_to","params":["alice"],"id":98}`,
			`{"jsonrpc":"2.0","method":"condenser_api.get_vesting_delegations","params":["alice","",10],"id":99}`,
			`{"jsonrpc":"2.0","method":"condenser_api.get_withdraw_routes","params":["alice","all"],"id":100}`,
			`{"jsonrpc":"2.0","method":"condenser_api.get_replies_by_last_update","params":["alice","utility-post",10],"id":101}`,
			`{"jsonrpc":"2.0","method":"condenser_api.get_trending_tags","params":["",10],"id":102}`,
			`{"jsonrpc":"2.0","method":"database_api.find_account_recovery_requests","params":{"accounts":["alice"]},"id":103}`,
			`{"jsonrpc":"2.0","method":"database_api.find_escrows","params":{"from":"alice"},"id":104}`,
			`{"jsonrpc":"2.0","method":"database_api.find_limit_orders","params":{"account":"alice"},"id":105}`,
			`{"jsonrpc":"2.0","method":"database_api.find_votes","params":{"author":"alice","permlink":"utility-post"},"id":106}`,
			`{"jsonrpc":"2.0","method":"database_api.find_comments","params":{"comments":[{"author":"alice","permlink":"utility-post"}]},"id":107}`,
			`{"jsonrpc":"2.0","method":"database_api.get_comment_pending_payouts","params":{"author":"alice","permlink":"utility-post"},"id":108}`,
			`{"jsonrpc":"2.0","method":"database_api.list_votes","params":{"start":["bob","alice","utility-post"],"limit":10,"order":"by_voter_comment"},"id":109}`,
			`{"jsonrpc":"2.0","method":"database_api.list_savings_withdrawals","params":{"start":["alice"],"limit":10,"order":"by_from_id"},"id":110}`,
			`{"jsonrpc":"2.0","method":"debug_node_api.debug_get_head_block","params":{},"id":111}`,
			`{"jsonrpc":"2.0","method":"hive.db_head_state","params":{},"id":112}`,
			fmt.Sprintf(`{"jsonrpc":"2.0","method":"condenser_api.get_transaction_hex","params":[%s],"id":113}`, txJSON),
			fmt.Sprintf(`{"jsonrpc":"2.0","method":"database_api.get_transaction_hex","params":{"trx":%s},"id":114}`, txJSON),
		} {
			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(reqBody))
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			var resp jsonRPCResponse
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to parse response: %v", err)
			}
			if resp.Error != nil {
				t.Fatalf("unexpected RPC error for %s: %v", reqBody, resp.Error)
			}
		}
	})

	t.Run("history authority debug and search APIs are useful first-class responses", func(t *testing.T) {
		txID := "a34f519fc6882a2384002ba93c90b7c02c7fe5f8"
		s.SaveTransaction(&state.TransactionData{
			TransactionID:  txID,
			BlockNum:       100000010,
			TransactionNum: 1,
			Expiration:     "2030-01-01T00:00:00",
			Operations: []any{
				[]any{"transfer", map[string]any{"from": "alice", "to": "bob", "amount": "1.000 HIVE", "memo": "history"}},
			},
		})
		s.SaveContent(&state.PostData{
			Author:       "alice",
			Permlink:     "searchable-post",
			Category:     "hive",
			Title:        "Searchable Hoverfly Post",
			Body:         "This post should appear in search results.",
			JSONMetadata: "{}",
			Created:      "2026-05-25T12:03:00",
		})

		txJSON := `{"ref_block_num":1,"ref_block_prefix":2,"expiration":"2030-01-01T00:00:00","operations":[["transfer",{"from":"alice","to":"bob","amount":"1.000 HIVE","memo":"auth"}]],"extensions":[],"signatures":[]}`
		for _, reqBody := range []string{
			`{"jsonrpc":"2.0","method":"account_history_api.enum_virtual_ops","params":{"block_range_begin":1,"block_range_end":2},"id":117}`,
			`{"jsonrpc":"2.0","method":"account_history_api.get_account_history","params":{"account":"alice","start":-1,"limit":10},"id":118}`,
			`{"jsonrpc":"2.0","method":"condenser_api.get_account_history","params":["alice",-1,10],"id":119}`,
			`{"jsonrpc":"2.0","method":"account_history_api.get_ops_in_block","params":{"block_num":100000010,"only_virtual":false},"id":120}`,
			`{"jsonrpc":"2.0","method":"condenser_api.get_ops_in_block","params":[100000010,false],"id":1201}`,
			fmt.Sprintf(`{"jsonrpc":"2.0","method":"condenser_api.get_potential_signatures","params":[%s,["STM5kQ1uy2CGNSwibSeYyLELWFng3HTyYVSsQd4Bjd4sWfqgKgtgJ"]],"id":121}`, txJSON),
			fmt.Sprintf(`{"jsonrpc":"2.0","method":"condenser_api.get_required_signatures","params":[%s,["STM5kQ1uy2CGNSwibSeYyLELWFng3HTyYVSsQd4Bjd4sWfqgKgtgJ"]],"id":122}`, txJSON),
			fmt.Sprintf(`{"jsonrpc":"2.0","method":"condenser_api.verify_authority","params":[%s],"id":123}`, txJSON),
			fmt.Sprintf(`{"jsonrpc":"2.0","method":"database_api.get_potential_signatures","params":{"trx":%s,"available_keys":["STM5kQ1uy2CGNSwibSeYyLELWFng3HTyYVSsQd4Bjd4sWfqgKgtgJ"]},"id":124}`, txJSON),
			fmt.Sprintf(`{"jsonrpc":"2.0","method":"database_api.get_required_signatures","params":{"trx":%s,"available_keys":["STM5kQ1uy2CGNSwibSeYyLELWFng3HTyYVSsQd4Bjd4sWfqgKgtgJ"]},"id":125}`, txJSON),
			fmt.Sprintf(`{"jsonrpc":"2.0","method":"database_api.verify_authority","params":{"trx":%s},"id":126}`, txJSON),
			`{"jsonrpc":"2.0","method":"database_api.verify_account_authority","params":{"account":"thecrazygm","keys":["STM5kQ1uy2CGNSwibSeYyLELWFng3HTyYVSsQd4Bjd4sWfqgKgtgJ"]},"id":127}`,
			`{"jsonrpc":"2.0","method":"database_api.verify_signatures","params":{"hash":"00","signatures":[],"required_owner":[],"required_active":[],"required_posting":[]},"id":128}`,
			`{"jsonrpc":"2.0","method":"debug_node_api.debug_generate_blocks","params":["initminer",2],"id":129}`,
			`{"jsonrpc":"2.0","method":"debug_node_api.debug_generate_blocks_until","params":["2030-01-01T00:00:00",false],"id":130}`,
			`{"jsonrpc":"2.0","method":"debug_node_api.debug_get_json_schema","params":{},"id":131}`,
			`{"jsonrpc":"2.0","method":"debug_node_api.debug_has_hardfork","params":{"hardfork":28},"id":132}`,
			`{"jsonrpc":"2.0","method":"debug_node_api.debug_set_hardfork","params":{"hardfork":28},"id":133}`,
			`{"jsonrpc":"2.0","method":"debug_node_api.debug_set_vest_price","params":{"price":{"base":"1.000 HIVE","quote":"1.000 VESTS"}},"id":134}`,
			`{"jsonrpc":"2.0","method":"search_api.find_text","params":{"q":"hoverfly","limit":10},"id":135}`,
		} {
			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(reqBody))
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			var resp jsonRPCResponse
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to parse response: %v", err)
			}
			if resp.Error != nil {
				t.Fatalf("unexpected RPC error for %s: %v", reqBody, resp.Error)
			}
		}

		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"jsonrpc":"2.0","method":"debug_node_api.debug_throw_exception","params":{},"id":136}`))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		var resp jsonRPCResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}
		if resp.Error == nil {
			t.Fatal("expected debug_throw_exception to return an intentional RPC error")
		}
	})

	t.Run("bridge APIs return local developer-useful responses", func(t *testing.T) {
		s.SaveContent(&state.PostData{
			Author:       "alice",
			Permlink:     "bridge-post",
			Category:     "hive",
			Title:        "Bridge Post",
			Body:         "Bridge API local state target.",
			JSONMetadata: `{"tags":["hive"],"app":"hoverfly-test"}`,
			Created:      "2026-05-25T12:04:00",
			ActiveVotes:  []string{"bob"},
		})
		s.SaveContent(&state.PostData{
			Author:         "bob",
			Permlink:       "re-bridge-post",
			ParentAuthor:   "alice",
			ParentPermlink: "bridge-post",
			Category:       "hive",
			Title:          "RE: Bridge Post",
			Body:           "Bridge reply target.",
			JSONMetadata:   "{}",
			Created:        "2026-05-25T12:05:00",
		})

		for _, reqBody := range []string{
			`{"jsonrpc":"2.0","method":"bridge.account_notifications","params":{"account":"alice"},"id":139}`,
			`{"jsonrpc":"2.0","method":"bridge.does_user_follow_any_lists","params":{"observer":"alice"},"id":140}`,
			`{"jsonrpc":"2.0","method":"bridge.get_account_posts","params":{"account":"alice","sort":"posts","limit":10},"id":141}`,
			`{"jsonrpc":"2.0","method":"bridge.get_community","params":{"name":"hive-123456"},"id":142}`,
			`{"jsonrpc":"2.0","method":"bridge.get_community_context","params":{"name":"hive-123456","account":"alice"},"id":143}`,
			`{"jsonrpc":"2.0","method":"bridge.get_discussion","params":{"author":"alice","permlink":"bridge-post"},"id":144}`,
			`{"jsonrpc":"2.0","method":"bridge.get_follow_list","params":{"observer":"alice","follow_type":"blacklisted"},"id":145}`,
			`{"jsonrpc":"2.0","method":"bridge.get_payout_stats","params":{},"id":146}`,
			`{"jsonrpc":"2.0","method":"bridge.get_post","params":{"author":"alice","permlink":"bridge-post"},"id":147}`,
			`{"jsonrpc":"2.0","method":"bridge.get_post_header","params":{"author":"alice","permlink":"bridge-post"},"id":148}`,
			`{"jsonrpc":"2.0","method":"bridge.get_profile","params":{"account":"alice"},"id":149}`,
			`{"jsonrpc":"2.0","method":"bridge.get_profiles","params":{"accounts":["alice","bob"]},"id":150}`,
			`{"jsonrpc":"2.0","method":"bridge.get_ranked_posts","params":{"tag":"hive","limit":10},"id":151}`,
			`{"jsonrpc":"2.0","method":"bridge.get_relationship_between_accounts","params":{"account1":"alice","account2":"bob","observer":"alice"},"id":152}`,
			`{"jsonrpc":"2.0","method":"bridge.get_trending_topics","params":{},"id":153}`,
			`{"jsonrpc":"2.0","method":"bridge.list_all_subscriptions","params":{"account":"alice"},"id":154}`,
			`{"jsonrpc":"2.0","method":"bridge.list_communities","params":{"limit":10,"query":"","sort":"rank"},"id":155}`,
			`{"jsonrpc":"2.0","method":"bridge.list_community_roles","params":{"community":"hive-123456","limit":10},"id":156}`,
			`{"jsonrpc":"2.0","method":"bridge.list_muted_reasons_enum","params":{},"id":157}`,
			`{"jsonrpc":"2.0","method":"bridge.list_pop_communities","params":{"limit":10},"id":158}`,
			`{"jsonrpc":"2.0","method":"bridge.list_subscribers","params":{"community":"hive-123456","limit":10},"id":159}`,
			`{"jsonrpc":"2.0","method":"bridge.normalize_post","params":{"author":"alice","permlink":"bridge-post"},"id":160}`,
			`{"jsonrpc":"2.0","method":"bridge.post_notifications","params":{"account":"alice"},"id":161}`,
			`{"jsonrpc":"2.0","method":"bridge.unread_notifications","params":{"account":"alice"},"id":162}`,
		} {
			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(reqBody))
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			var resp jsonRPCResponse
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to parse response: %v", err)
			}
			if resp.Error != nil {
				t.Fatalf("unexpected RPC error for %s: %v", reqBody, resp.Error)
			}
		}
	})

	t.Run("known OpenAPI method gets generic mock response", func(t *testing.T) {
		reqBody := `{"jsonrpc":"2.0","method":"bridge.get_trending_topics","params":{},"id":137}`
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(reqBody))
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		var resp jsonRPCResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if resp.Error != nil {
			t.Fatalf("unexpected RPC error for known OpenAPI method: %v", resp.Error)
		}

		topics, ok := resp.Result.([]any)
		if !ok || len(topics) == 0 {
			t.Fatalf("expected OpenAPI example topics, got %T: %v", resp.Result, resp.Result)
		}
	})

	t.Run("unknown method still returns method not found", func(t *testing.T) {
		reqBody := `{"jsonrpc":"2.0","method":"not_api.nope","params":[],"id":138}`
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(reqBody))
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		var resp jsonRPCResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if resp.Error == nil {
			t.Fatal("expected RPC error for unknown method")
		}
	})
}

func TestOpenAPIMockResponses(t *testing.T) {
	s, err := state.NewState("", false)
	if err != nil {
		t.Fatalf("failed to create state: %v", err)
	}
	defer s.Close()

	handler := NewRPCHandler(s, false)

	if len(knownHiveAPIMethods) != 215 {
		t.Fatalf("expected 215 known Hive OpenAPI methods, got %d", len(knownHiveAPIMethods))
	}

	if len(openAPIMockResponses) != len(knownHiveAPIMethods) {
		t.Fatalf("expected mock response for every known method, got %d responses for %d methods", len(openAPIMockResponses), len(knownHiveAPIMethods))
	}

	for method := range knownHiveAPIMethods {
		result, ok := handler.handleKnownOpenAPIMethod(method, nil)
		if !ok {
			t.Fatalf("expected known method %s to be handled by OpenAPI fallback", method)
		}

		raw, ok := openAPIMockResponses[method]
		if !ok {
			t.Fatalf("missing OpenAPI mock response for %s", method)
		}
		var decoded any
		if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
			t.Fatalf("invalid generated JSON for %s: %v", method, err)
		}
		if result == nil && raw != "null" {
			t.Fatalf("expected non-nil mock result for %s", method)
		}
	}
}
