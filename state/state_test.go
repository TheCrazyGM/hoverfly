package state

import (
	"testing"
)

func TestStateDefaults(t *testing.T) {
	// Create an in-memory State
	s, err := NewState("", false)
	if err != nil {
		t.Fatalf("failed to create state: %v", err)
	}
	defer s.Close()

	// Check default seeded account
	acc, err := s.GetAccount("alice")
	if err != nil {
		t.Fatalf("failed to get account alice: %v", err)
	}
	if acc.Name != "alice" {
		t.Errorf("expected account name 'alice', got '%s'", acc.Name)
	}

	// Check key references
	if acc.ActiveKey == "" {
		t.Fatalf("expected alice to have an ActiveKey")
	}
	refs, err := s.GetKeyReferences([]string{acc.ActiveKey})
	if err != nil {
		t.Fatalf("failed to get key references: %v", err)
	}
	if len(refs) != 1 || refs[0] != "alice" {
		t.Errorf("expected key reference 'alice', got %v", refs)
	}

	// Check dynamic global properties
	props, err := s.GetDynamicProperties()
	if err != nil {
		t.Fatalf("failed to get dynamic properties: %v", err)
	}
	if props.HeadBlockNumber != 100000000 {
		t.Errorf("expected head block 100000000, got %d", props.HeadBlockNumber)
	}
}

func TestStateAccountMutations(t *testing.T) {
	s, err := NewState("", false)
	if err != nil {
		t.Fatalf("failed to create state: %v", err)
	}
	defer s.Close()

	// Save custom account
	newAcc := &AccountData{
		Name:          "dave",
		VotingPower:   5000,
		Balance:       "50.000 HIVE",
		HbdBalance:    "5.000 HBD",
		VestingShares: "1000000.000000 VESTS",
	}

	if err := s.SaveAccount(newAcc); err != nil {
		t.Fatalf("failed to save account: %v", err)
	}

	acc, err := s.GetAccount("dave")
	if err != nil {
		t.Fatalf("failed to get account: %v", err)
	}
	if acc.Balance != "50.000 HIVE" {
		t.Errorf("expected balance '50.000 HIVE', got '%s'", acc.Balance)
	}
}

func TestStateContentMutations(t *testing.T) {
	s, err := NewState("", false)
	if err != nil {
		t.Fatalf("failed to create state: %v", err)
	}
	defer s.Close()

	// Save custom post
	newPost := &PostData{
		Author:   "alice",
		Permlink: "first-post",
		Title:    "Hello World",
		Body:     "This is my first post on Hoverfly!",
	}

	if err := s.SaveContent(newPost); err != nil {
		t.Fatalf("failed to save content: %v", err)
	}

	post, err := s.GetContent("alice", "first-post")
	if err != nil {
		t.Fatalf("failed to get content: %v", err)
	}
	if post.Title != "Hello World" {
		t.Errorf("expected post title 'Hello World', got '%s'", post.Title)
	}
}
