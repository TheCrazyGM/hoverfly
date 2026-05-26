package rpc

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/charmbracelet/log"
	"github.com/srbde/hoverfly/state"
)

type jsonRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
	ID      any             `json:"id"`
}

type jsonRPCResponse struct {
	JSONRPC string `json:"jsonrpc"`
	Result  any    `json:"result,omitempty"`
	Error   any    `json:"error,omitempty"`
	ID      any    `json:"id"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type rpcMethodFunc func(json.RawMessage) (any, *rpcError)

type RPCHandler struct {
	state  *state.State
	debug  bool
	routes map[string]rpcMethodFunc
}

func NewRPCHandler(s *state.State, debug bool) *RPCHandler {
	h := &RPCHandler{
		state:  s,
		debug:  debug,
		routes: make(map[string]rpcMethodFunc),
	}
	h.initRoutes()
	return h
}

func (h *RPCHandler) route(method string, params json.RawMessage) (any, *rpcError) {
	if fn, ok := h.routes[method]; ok {
		return fn(params)
	}

	if result, ok := h.handleKnownOpenAPIMethod(method, params); ok {
		return result, nil
	}

	return nil, &rpcError{Code: -32601, Message: fmt.Sprintf("Method not found: %s", method)}
}

func (h *RPCHandler) initRoutes() {
	// Accounts
	h.routes["condenser_api.get_accounts"] = h.handleGetAccounts
	h.routes["database_api.get_accounts"] = h.handleGetAccounts
	h.routes["database_api.find_accounts"] = h.handleFindAccounts
	h.routes["condenser_api.get_account_count"] = func(_ json.RawMessage) (any, *rpcError) { return h.handleGetAccountCount() }
	h.routes["condenser_api.lookup_accounts"] = h.handleLookupAccounts
	h.routes["condenser_api.lookup_account_names"] = h.handleLookupAccountNames
	h.routes["database_api.list_accounts"] = h.handleListAccounts
	h.routes["condenser_api.get_key_references"] = h.handleGetKeyReferences
	h.routes["account_by_key_api.get_key_references"] = h.handleGetKeyReferences

	// Content / Bridge
	h.routes["condenser_api.get_content"] = h.handleGetContent
	h.routes["tags_api.get_discussion"] = h.handleGetContent
	h.routes["condenser_api.get_content_replies"] = h.handleGetContentReplies
	h.routes["tags_api.get_content_replies"] = h.handleGetContentReplies
	h.routes["condenser_api.get_active_votes"] = h.handleGetActiveVotes
	h.routes["database_api.find_votes"] = h.handleDatabaseVotes
	h.routes["database_api.list_votes"] = h.handleDatabaseVotes
	h.routes["condenser_api.get_blog"] = h.handleGetBlog
	h.routes["follow_api.get_blog"] = h.handleGetBlog
	h.routes["condenser_api.get_blog_entries"] = h.handleGetBlogEntries
	h.routes["follow_api.get_blog_entries"] = h.handleGetBlogEntries
	h.routes["database_api.find_comments"] = h.handleFindComments
	h.routes["database_api.list_comments"] = h.handleListComments
	h.routes["rewards_api.simulate_curve_payouts"] = h.handleSimulateCurvePayouts
	h.routes["database_api.get_comment_pending_payouts"] = h.handleCommentPendingPayouts
	h.routes["search_api.find_text"] = h.handleSearchText

	// Discussions - multiple tags_api and condenser_api endpoints
	discussionsMethods := []string{
		"condenser_api.get_discussions_by_author_before_date", "condenser_api.get_discussions_by_blog",
		"condenser_api.get_discussions_by_comments", "condenser_api.get_discussions_by_created",
		"condenser_api.get_discussions_by_feed", "condenser_api.get_discussions_by_hot",
		"condenser_api.get_discussions_by_trending", "condenser_api.get_comment_discussions_by_payout",
		"condenser_api.get_post_discussions_by_payout", "tags_api.get_discussions_by_author_before_date",
		"tags_api.get_discussions_by_blog", "tags_api.get_discussions_by_comments",
		"tags_api.get_discussions_by_created", "tags_api.get_discussions_by_hot",
		"tags_api.get_discussions_by_trending", "tags_api.get_comment_discussions_by_payout",
		"tags_api.get_post_discussions_by_payout",
	}
	for _, m := range discussionsMethods {
		h.routes[m] = h.handleGetDiscussions
	}

	h.routes["condenser_api.get_account_reputations"] = h.handleGetAccountReputations
	h.routes["follow_api.get_account_reputations"] = h.handleGetAccountReputations
	h.routes["reputation_api.get_account_reputations"] = h.handleGetAccountReputations
	h.routes["condenser_api.get_followers"] = h.handleGetFollowers
	h.routes["follow_api.get_followers"] = h.handleGetFollowers
	h.routes["condenser_api.get_following"] = h.handleGetFollowing
	h.routes["follow_api.get_following"] = h.handleGetFollowing

	emptyListStrSlice := func(_ json.RawMessage) (any, *rpcError) { return []string{}, nil }
	h.routes["condenser_api.get_reblogged_by"] = emptyListStrSlice
	h.routes["follow_api.get_reblogged_by"] = emptyListStrSlice

	// Bridge APIs
	h.routes["bridge.get_post"] = h.handleBridgePost
	h.routes["bridge.normalize_post"] = h.handleBridgePost
	h.routes["bridge.get_post_header"] = h.handleBridgePostHeader
	h.routes["bridge.get_discussion"] = h.handleBridgeDiscussion
	h.routes["bridge.get_account_posts"] = h.handleBridgeAccountPosts
	h.routes["bridge.get_ranked_posts"] = h.handleBridgeRankedPosts
	h.routes["bridge.get_profile"] = h.handleBridgeProfile
	h.routes["bridge.get_profiles"] = h.handleBridgeProfiles
	h.routes["bridge.get_community"] = h.handleBridgeCommunity
	h.routes["bridge.list_communities"] = h.handleBridgeListCommunities
	h.routes["bridge.list_pop_communities"] = h.handleBridgeListPopCommunities
	h.routes["bridge.get_trending_topics"] = h.handleBridgeTrendingTopics
	h.routes["bridge.get_payout_stats"] = func(_ json.RawMessage) (any, *rpcError) { return h.handleBridgePayoutStats() }

	h.routes["bridge.get_community_context"] = func(_ json.RawMessage) (any, *rpcError) {
		return map[string]any{"role": "guest", "subscribed": false, "title": ""}, nil
	}
	emptyListAny := func(_ json.RawMessage) (any, *rpcError) { return []any{}, nil }
	h.routes["bridge.list_subscribers"] = emptyListAny
	h.routes["bridge.list_community_roles"] = emptyListAny
	h.routes["bridge.list_all_subscriptions"] = emptyListAny
	h.routes["bridge.get_follow_list"] = emptyListAny
	h.routes["bridge.does_user_follow_any_lists"] = func(_ json.RawMessage) (any, *rpcError) { return false, nil }
	h.routes["bridge.get_relationship_between_accounts"] = func(_ json.RawMessage) (any, *rpcError) {
		return map[string]any{"blacklists": false, "follows": false, "follows_blacklists": false, "follows_muted": false, "ignores": false}, nil
	}
	h.routes["bridge.account_notifications"] = emptyListAny
	h.routes["bridge.post_notifications"] = emptyListAny
	h.routes["bridge.unread_notifications"] = func(_ json.RawMessage) (any, *rpcError) {
		return map[string]any{"lastread": time.Now().UTC().Format("2006-01-02 15:04:05"), "unread": 0}, nil
	}
	h.routes["bridge.list_muted_reasons_enum"] = func(_ json.RawMessage) (any, *rpcError) {
		return map[string]any{
			"MUTED_COMMUNITY_MODERATION": 0,
			"MUTED_COMMUNITY_TYPE":       1,
			"MUTED_PARENT":               2,
			"MUTED_REPUTATION":           3,
			"MUTED_ROLE_COMMUNITY":       4,
		}, nil
	}

	// Blocks & History
	h.routes["condenser_api.get_block"] = func(p json.RawMessage) (any, *rpcError) { return h.handleGetBlock(p, "condenser_api.get_block") }
	h.routes["database_api.get_block"] = func(p json.RawMessage) (any, *rpcError) { return h.handleGetBlock(p, "database_api.get_block") }
	h.routes["block_api.get_block"] = func(p json.RawMessage) (any, *rpcError) { return h.handleGetBlock(p, "block_api.get_block") }

	h.routes["condenser_api.get_block_header"] = func(p json.RawMessage) (any, *rpcError) { return h.handleGetBlockHeader(p, "condenser_api.get_block_header") }
	h.routes["database_api.get_block_header"] = func(p json.RawMessage) (any, *rpcError) { return h.handleGetBlockHeader(p, "database_api.get_block_header") }
	h.routes["block_api.get_block_header"] = func(p json.RawMessage) (any, *rpcError) { return h.handleGetBlockHeader(p, "block_api.get_block_header") }

	h.routes["block_api.get_block_range"] = h.handleGetBlockRange
	h.routes["condenser_api.get_ops_in_block"] = h.handleCondenserGetOpsInBlock
	h.routes["account_history_api.get_ops_in_block"] = h.handleGetOpsInBlock
	h.routes["account_history_api.enum_virtual_ops"] = h.handleEnumVirtualOps
	h.routes["condenser_api.get_follow_count"] = h.handleGetFollowCount
	h.routes["follow_api.get_follow_count"] = h.handleGetFollowCount

	h.routes["condenser_api.get_transaction"] = h.handleGetTransaction
	h.routes["account_history_api.get_transaction"] = h.handleGetTransaction

	h.routes["condenser_api.is_known_transaction"] = func(p json.RawMessage) (any, *rpcError) { return h.handleIsKnownTransaction("condenser_api.is_known_transaction", p) }
	h.routes["database_api.is_known_transaction"] = func(p json.RawMessage) (any, *rpcError) { return h.handleIsKnownTransaction("database_api.is_known_transaction", p) }

	h.routes["transaction_status_api.find_transaction"] = h.handleFindTransaction
	h.routes["jsonrpc.get_methods"] = func(_ json.RawMessage) (any, *rpcError) { return h.handleGetMethods() }
	h.routes["jsonrpc.get_signature"] = h.handleGetSignature
	h.routes["hive.get_info"] = func(_ json.RawMessage) (any, *rpcError) { return h.handleHiveInfo() }

	h.routes["account_history_api.get_account_history"] = func(p json.RawMessage) (any, *rpcError) { return h.handleGetAccountHistory("account_history_api.get_account_history", p) }
	h.routes["condenser_api.get_account_history"] = func(p json.RawMessage) (any, *rpcError) { return h.handleGetAccountHistory("condenser_api.get_account_history", p) }

	// Broadcast & Hex & Signatures
	broadcastFunc := h.handleBroadcastTransaction
	h.routes["condenser_api.broadcast_transaction"] = broadcastFunc
	h.routes["condenser_api.broadcast_transaction_synchronous"] = broadcastFunc
	h.routes["network_broadcast_api.broadcast_transaction"] = broadcastFunc
	h.routes["network_broadcast_api.broadcast_transaction_synchronous"] = broadcastFunc

	h.routes["condenser_api.get_transaction_hex"] = func(p json.RawMessage) (any, *rpcError) { return h.handleGetTransactionHex("condenser_api.get_transaction_hex", p) }
	h.routes["database_api.get_transaction_hex"] = func(p json.RawMessage) (any, *rpcError) { return h.handleGetTransactionHex("database_api.get_transaction_hex", p) }

	h.routes["condenser_api.get_potential_signatures"] = func(p json.RawMessage) (any, *rpcError) { return h.handlePotentialSignatures("condenser_api.get_potential_signatures", p) }
	h.routes["database_api.get_potential_signatures"] = func(p json.RawMessage) (any, *rpcError) { return h.handlePotentialSignatures("database_api.get_potential_signatures", p) }

	h.routes["condenser_api.get_required_signatures"] = func(p json.RawMessage) (any, *rpcError) { return h.handleRequiredSignatures("condenser_api.get_required_signatures", p) }
	h.routes["database_api.get_required_signatures"] = func(p json.RawMessage) (any, *rpcError) { return h.handleRequiredSignatures("database_api.get_required_signatures", p) }

	h.routes["condenser_api.verify_authority"] = func(p json.RawMessage) (any, *rpcError) { return h.handleVerifyAuthority("condenser_api.verify_authority", p) }
	h.routes["database_api.verify_authority"] = func(p json.RawMessage) (any, *rpcError) { return h.handleVerifyAuthority("database_api.verify_authority", p) }
	h.routes["database_api.verify_account_authority"] = func(p json.RawMessage) (any, *rpcError) { return h.handleVerifyAuthority("database_api.verify_account_authority", p) }
	h.routes["database_api.verify_signatures"] = func(p json.RawMessage) (any, *rpcError) { return h.handleVerifyAuthority("database_api.verify_signatures", p) }

	// Debug Node
	h.routes["debug_node_api.debug_get_head_block"] = func(_ json.RawMessage) (any, *rpcError) { return h.handleDebugHeadBlock() }
	h.routes["debug_node_api.debug_generate_blocks"] = h.handleDebugGenerateBlocks
	h.routes["debug_node_api.debug_generate_blocks_until"] = h.handleDebugGenerateBlocksUntil
	h.routes["hive.db_head_state"] = func(_ json.RawMessage) (any, *rpcError) { return h.handleDBHeadState() }

	h.routes["debug_node_api.debug_get_json_schema"] = func(p json.RawMessage) (any, *rpcError) { return h.handleOpenAPIExample("debug_node_api.debug_get_json_schema") }
	h.routes["debug_node_api.debug_has_hardfork"] = func(_ json.RawMessage) (any, *rpcError) { return true, nil }
	okTrueResponse := func(_ json.RawMessage) (any, *rpcError) { return map[string]any{"ok": true}, nil }
	h.routes["debug_node_api.debug_set_hardfork"] = okTrueResponse
	h.routes["debug_node_api.debug_set_vest_price"] = okTrueResponse
	h.routes["debug_node_api.debug_throw_exception"] = func(_ json.RawMessage) (any, *rpcError) {
		return nil, &rpcError{Code: -32000, Message: "debug exception requested"}
	}

	// Governance / RC / Config
	h.routes["condenser_api.find_rc_accounts"] = h.handleFindRCAccounts
	h.routes["rc_api.find_rc_accounts"] = h.handleFindRCAccounts

	h.routes["condenser_api.list_rc_accounts"] = func(p json.RawMessage) (any, *rpcError) { return h.handleListRCAccounts("condenser_api.list_rc_accounts", p) }
	h.routes["rc_api.list_rc_accounts"] = func(p json.RawMessage) (any, *rpcError) { return h.handleListRCAccounts("rc_api.list_rc_accounts", p) }

	h.routes["condenser_api.list_rc_direct_delegations"] = func(_ json.RawMessage) (any, *rpcError) { return h.handleListRCDirectDelegations("condenser_api.list_rc_direct_delegations") }
	h.routes["rc_api.list_rc_direct_delegations"] = func(_ json.RawMessage) (any, *rpcError) { return h.handleListRCDirectDelegations("rc_api.list_rc_direct_delegations") }

	rcStatsFunc := func(m string) rpcMethodFunc {
		return func(_ json.RawMessage) (any, *rpcError) { return h.handleOpenAPIExample(m) }
	}
	h.routes["rc_api.get_rc_operation_stats"] = rcStatsFunc("rc_api.get_rc_operation_stats")
	h.routes["rc_api.get_rc_stats"] = rcStatsFunc("rc_api.get_rc_stats")
	h.routes["rc_api.get_resource_params"] = rcStatsFunc("rc_api.get_resource_params")
	h.routes["rc_api.get_resource_pool"] = rcStatsFunc("rc_api.get_resource_pool")

	h.routes["database_api.get_config"] = func(_ json.RawMessage) (any, *rpcError) { return h.handleGetConfig() }
	h.routes["condenser_api.get_config"] = func(_ json.RawMessage) (any, *rpcError) { return h.handleGetConfig() }
	h.routes["condenser_api.get_chain_properties"] = func(_ json.RawMessage) (any, *rpcError) { return h.handleGetChainProperties() }
	h.routes["condenser_api.get_version"] = func(_ json.RawMessage) (any, *rpcError) { return h.handleGetVersion() }
	h.routes["database_api.get_version"] = func(_ json.RawMessage) (any, *rpcError) { return h.handleGetVersion() }
	h.routes["condenser_api.get_hardfork_version"] = func(_ json.RawMessage) (any, *rpcError) { return "1.28.6", nil }

	h.routes["condenser_api.get_witness_count"] = func(_ json.RawMessage) (any, *rpcError) { return 21, nil }
	h.routes["condenser_api.lookup_witness_accounts"] = h.handleLookupWitnessAccounts
	h.routes["condenser_api.get_witness_by_account"] = h.handleGetWitnessByAccount

	witnessFunc := func(m string) rpcMethodFunc {
		return func(_ json.RawMessage) (any, *rpcError) { return h.handleOpenAPIExample(m) }
	}
	h.routes["condenser_api.get_witnesses"] = witnessFunc("condenser_api.get_witnesses")
	h.routes["condenser_api.get_witnesses_by_vote"] = witnessFunc("condenser_api.get_witnesses_by_vote")
	h.routes["database_api.find_witnesses"] = witnessFunc("database_api.find_witnesses")
	h.routes["database_api.list_witnesses"] = witnessFunc("database_api.list_witnesses")

	h.routes["condenser_api.find_proposals"] = func(_ json.RawMessage) (any, *rpcError) { return h.handleProposalList("condenser_api.find_proposals") }
	h.routes["database_api.find_proposals"] = func(_ json.RawMessage) (any, *rpcError) { return h.handleProposalList("database_api.find_proposals") }
	h.routes["condenser_api.list_proposals"] = func(_ json.RawMessage) (any, *rpcError) { return h.handleProposalList("condenser_api.list_proposals") }
	h.routes["database_api.list_proposals"] = func(_ json.RawMessage) (any, *rpcError) { return h.handleProposalList("database_api.list_proposals") }

	h.routes["condenser_api.list_proposal_votes"] = func(_ json.RawMessage) (any, *rpcError) { return h.handleProposalVoteList("condenser_api.list_proposal_votes") }
	h.routes["database_api.list_proposal_votes"] = func(_ json.RawMessage) (any, *rpcError) { return h.handleProposalVoteList("database_api.list_proposal_votes") }

	h.routes["database_api.list_witness_votes"] = func(_ json.RawMessage) (any, *rpcError) { return map[string]any{"votes": []any{}}, nil }

	emptyListWitnesses := func(_ json.RawMessage) (any, *rpcError) { return []any{}, nil }
	h.routes["condenser_api.find_recurrent_transfers"] = emptyListWitnesses
	h.routes["condenser_api.get_collateralized_conversion_requests"] = emptyListWitnesses
	h.routes["condenser_api.get_conversion_requests"] = emptyListWitnesses
	h.routes["condenser_api.get_expiring_vesting_delegations"] = emptyListWitnesses
	h.routes["condenser_api.get_open_orders"] = emptyListWitnesses
	h.routes["condenser_api.get_owner_history"] = emptyListWitnesses
	h.routes["condenser_api.get_savings_withdraw_from"] = emptyListWitnesses
	h.routes["condenser_api.get_savings_withdraw_to"] = emptyListWitnesses
	h.routes["condenser_api.get_vesting_delegations"] = emptyListWitnesses
	h.routes["condenser_api.get_withdraw_routes"] = emptyListWitnesses
	h.routes["condenser_api.get_replies_by_last_update"] = emptyListWitnesses
	h.routes["condenser_api.get_trending_tags"] = emptyListWitnesses
	h.routes["condenser_api.get_blog_authors"] = emptyListWitnesses

	h.routes["condenser_api.get_escrow"] = func(_ json.RawMessage) (any, *rpcError) { return json.RawMessage("null"), nil }
	h.routes["condenser_api.get_recovery_request"] = func(_ json.RawMessage) (any, *rpcError) { return json.RawMessage("null"), nil }

	emptyStateFunc := func(m string) rpcMethodFunc {
		return func(_ json.RawMessage) (any, *rpcError) { return h.handleEmptyDatabaseState(m) }
	}
	h.routes["database_api.find_account_recovery_requests"] = emptyStateFunc("database_api.find_account_recovery_requests")
	h.routes["database_api.find_change_recovery_account_requests"] = emptyStateFunc("database_api.find_change_recovery_account_requests")
	h.routes["database_api.find_collateralized_conversion_requests"] = emptyStateFunc("database_api.find_collateralized_conversion_requests")
	h.routes["database_api.find_decline_voting_rights_requests"] = emptyStateFunc("database_api.find_decline_voting_rights_requests")
	h.routes["database_api.find_escrows"] = emptyStateFunc("database_api.find_escrows")
	h.routes["database_api.find_hbd_conversion_requests"] = emptyStateFunc("database_api.find_hbd_conversion_requests")
	h.routes["database_api.find_limit_orders"] = emptyStateFunc("database_api.find_limit_orders")
	h.routes["database_api.find_owner_histories"] = emptyStateFunc("database_api.find_owner_histories")
	h.routes["database_api.find_recurrent_transfers"] = emptyStateFunc("database_api.find_recurrent_transfers")
	h.routes["database_api.find_savings_withdrawals"] = emptyStateFunc("database_api.find_savings_withdrawals")
	h.routes["database_api.find_vesting_delegation_expirations"] = emptyStateFunc("database_api.find_vesting_delegation_expirations")
	h.routes["database_api.find_vesting_delegations"] = emptyStateFunc("database_api.find_vesting_delegations")
	h.routes["database_api.find_withdraw_vesting_routes"] = emptyStateFunc("database_api.find_withdraw_vesting_routes")
	h.routes["database_api.list_account_recovery_requests"] = emptyStateFunc("database_api.list_account_recovery_requests")
	h.routes["database_api.list_change_recovery_account_requests"] = emptyStateFunc("database_api.list_change_recovery_account_requests")
	h.routes["database_api.list_collateralized_conversion_requests"] = emptyStateFunc("database_api.list_collateralized_conversion_requests")
	h.routes["database_api.list_decline_voting_rights_requests"] = emptyStateFunc("database_api.list_decline_voting_rights_requests")
	h.routes["database_api.list_escrows"] = emptyStateFunc("database_api.list_escrows")
	h.routes["database_api.list_hbd_conversion_requests"] = emptyStateFunc("database_api.list_hbd_conversion_requests")
	h.routes["database_api.list_limit_orders"] = emptyStateFunc("database_api.list_limit_orders")
	h.routes["database_api.list_owner_histories"] = emptyStateFunc("database_api.list_owner_histories")
	h.routes["database_api.list_savings_withdrawals"] = emptyStateFunc("database_api.list_savings_withdrawals")
	h.routes["database_api.list_vesting_delegation_expirations"] = emptyStateFunc("database_api.list_vesting_delegation_expirations")
	h.routes["database_api.list_vesting_delegations"] = emptyStateFunc("database_api.list_vesting_delegations")
	h.routes["database_api.list_withdraw_vesting_routes"] = emptyStateFunc("database_api.list_withdraw_vesting_routes")

	witnessBandwidthFunc := func(_ json.RawMessage) (any, *rpcError) { return map[string]any{}, nil }
	h.routes["witness_api.get_account_bandwidth"] = witnessBandwidthFunc
	h.routes["network_broadcast_api.broadcast_block"] = witnessBandwidthFunc
}

func (h *RPCHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		writeError(w, nil, -32600, "Invalid Request Method (POST required)")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, nil, -32700, "Parse error reading request body")
		return
	}

	var req jsonRPCRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, nil, -32700, "Parse error invalid JSON-RPC payload")
		return
	}

	log.Debugf("RPC Call: %s | Params: %s", req.Method, string(req.Params))

	result, rpcErr := h.route(req.Method, req.Params)
	if rpcErr != nil {
		log.Warnf("RPC Error: %d - %s", rpcErr.Code, rpcErr.Message)
		writeError(w, req.ID, rpcErr.Code, rpcErr.Message)
		return
	}

	resp := jsonRPCResponse{
		JSONRPC: "2.0",
		Result:  result,
		ID:      req.ID,
	}
	respBytes, _ := json.Marshal(resp)
	w.Write(respBytes)
}

func writeError(w http.ResponseWriter, id any, code int, msg string) {
	resp := jsonRPCResponse{
		JSONRPC: "2.0",
		Error: rpcError{
			Code:    code,
			Message: msg,
		},
		ID: id,
	}
	respBytes, _ := json.Marshal(resp)
	w.Write(respBytes)
}

type Authority struct {
	WeightThreshold uint32  `json:"weight_threshold"`
	AccountAuths    [][]any `json:"account_auths"`
	KeyAuths        [][]any `json:"key_auths"`
}

type EnrichedAccountData struct {
	Name                   string        `json:"name"`
	VotingPower            float64       `json:"voting_power"`
	VotingManabar          state.Manabar `json:"voting_manabar"`
	LastVoteTime           string        `json:"last_vote_time"`
	Balance                string        `json:"balance"`
	HbdBalance             string        `json:"hbd_balance"`
	VestingShares          string        `json:"vesting_shares"`
	Created                string        `json:"created"`
	SavingsBalance         string        `json:"savings_balance"`
	SavingsHbdBalance      string        `json:"savings_hbd_balance"`
	PostCount              uint32        `json:"post_count"`
	Reputation             any           `json:"reputation"`
	RewardHiveBalance      string        `json:"reward_hive_balance"`
	RewardHbdBalance       string        `json:"reward_hbd_balance"`
	RewardVestingBalance   string        `json:"reward_vesting_balance"`
	RewardVestingHive      string        `json:"reward_vesting_hive"`
	DelegatedVestingShares string        `json:"delegated_vesting_shares"`
	ReceivedVestingShares  string        `json:"received_vesting_shares"`
	VestingWithdrawRate    string        `json:"vesting_withdraw_rate"`
	NextVestingWithdrawal  string        `json:"next_vesting_withdrawal"`
	Owner                  Authority     `json:"owner"`
	Active                 Authority     `json:"active"`
	Posting                Authority     `json:"posting"`
	MemoKey                string        `json:"memo_key"`
}

func enrichAccount(acc state.AccountData) EnrichedAccountData {
	activeKey := "STM5kQ1uy2CGNSwibSeYyLELWFng3HTyYVSsQd4Bjd4sWfqgKgtgJ"
	postingKey := "STM8Ep2rQp1wPzBPE2tS7tfcvU2JpbnkeyhfsYB1Jcnz7S2w8H9Q3"
	ownerKey := "STM5kQ1uy2CGNSwibSeYyLELWFng3HTyYVSsQd4Bjd4sWfqgKgtgJ"
	memoKey := "STM5kQ1uy2CGNSwibSeYyLELWFng3HTyYVSsQd4Bjd4sWfqgKgtgJ"

	savingsBalance := acc.SavingsBalance
	if savingsBalance == "" {
		savingsBalance = "0.000 HIVE"
	}
	savingsHbdBalance := acc.SavingsHbdBalance
	if savingsHbdBalance == "" {
		savingsHbdBalance = "0.000 HBD"
	}

	return EnrichedAccountData{
		Name:                   acc.Name,
		VotingPower:            acc.VotingPower,
		VotingManabar:          acc.VotingManabar,
		LastVoteTime:           acc.LastVoteTime,
		Balance:                acc.Balance,
		HbdBalance:             acc.HbdBalance,
		VestingShares:          acc.VestingShares,
		Created:                acc.Created,
		SavingsBalance:         savingsBalance,
		SavingsHbdBalance:      savingsHbdBalance,
		PostCount:              0,
		Reputation:             "1000000000000",
		RewardHiveBalance:      "0.000 HIVE",
		RewardHbdBalance:       "0.000 HBD",
		RewardVestingBalance:   "0.000000 VESTS",
		RewardVestingHive:      "0.000 HIVE",
		DelegatedVestingShares: "0.000000 VESTS",
		ReceivedVestingShares:  "0.000000 VESTS",
		VestingWithdrawRate:    "0.000000 VESTS",
		NextVestingWithdrawal:  "1970-01-01T00:00:00",
		Owner: Authority{
			WeightThreshold: 1,
			AccountAuths:    [][]any{},
			KeyAuths:        [][]any{{ownerKey, 1}},
		},
		Active: Authority{
			WeightThreshold: 1,
			AccountAuths:    [][]any{},
			KeyAuths:        [][]any{{activeKey, 1}},
		},
		Posting: Authority{
			WeightThreshold: 1,
			AccountAuths:    [][]any{},
			KeyAuths:        [][]any{{postingKey, 1}},
		},
		MemoKey: memoKey,
	}
}

func (h *RPCHandler) handleGetDynamicGlobalProperties() (any, *rpcError) {
	props, err := h.state.GetDynamicProperties()
	if err != nil {
		return nil, &rpcError{Code: -32603, Message: err.Error()}
	}
	return props, nil
}

func clampLimit(limit, max int) int {
	if limit <= 0 {
		return max
	}
	if limit > max {
		return max
	}
	return limit
}

func (h *RPCHandler) handleKnownOpenAPIMethod(method string, params json.RawMessage) (any, bool) {
	if _, ok := knownHiveAPIMethods[method]; !ok {
		return nil, false
	}

	log.Debugf("Using generic OpenAPI mock response for %s", method)

	result, rpcErr := h.handleOpenAPIExample(method)
	if rpcErr == nil {
		return result, true
	}

	return map[string]any{}, true
}

func (h *RPCHandler) handleOpenAPIExample(method string) (any, *rpcError) {
	if raw, ok := openAPIMockResponses[method]; ok {
		var result any
		if err := json.Unmarshal([]byte(raw), &result); err == nil {
			return result, nil
		}
	}

	return nil, &rpcError{Code: -32603, Message: fmt.Sprintf("invalid OpenAPI mock response for %s", method)}
}

func mustMarshal(v any) json.RawMessage {
	bytes, _ := json.Marshal(v)
	return bytes
}
