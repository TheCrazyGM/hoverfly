package state

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/dgraph-io/badger/v4"
	anthercrypto "github.com/srbde/anther/crypto"
)

type Manabar struct {
	CurrentMana    float64 `json:"current_mana"`
	LastUpdateTime int64   `json:"last_update_time"`
}

type AccountData struct {
	Name              string  `json:"name"`
	VotingPower       float64 `json:"voting_power"`
	VotingManabar     Manabar `json:"voting_manabar"`
	LastVoteTime      string  `json:"last_vote_time"`
	Balance           string  `json:"balance"`
	HbdBalance        string  `json:"hbd_balance"`
	VestingShares     string  `json:"vesting_shares"`
	Created           string  `json:"created"`
	SavingsBalance    string  `json:"savings_balance"`
	SavingsHbdBalance string  `json:"savings_hbd_balance"`

	// Key fields (dynamic / rotating)
	ActiveKey         string `json:"active_key,omitempty"`
	PostingKey        string `json:"posting_key,omitempty"`
	OwnerKey          string `json:"owner_key,omitempty"`
	MemoKey           string `json:"memo_key,omitempty"`
	ActivePrivateKey  string `json:"active_private_key,omitempty"`
	PostingPrivateKey string `json:"posting_private_key,omitempty"`
}

type PostData struct {
	Author         string   `json:"author"`
	Permlink       string   `json:"permlink"`
	ParentAuthor   string   `json:"parent_author"`
	ParentPermlink string   `json:"parent_permlink"`
	Category       string   `json:"category"`
	Title          string   `json:"title"`
	Body           string   `json:"body"`
	JSONMetadata   string   `json:"json_metadata"`
	Created        string   `json:"created"`
	ActiveVotes    []string `json:"active_votes"`
}

type DynamicProperties struct {
	HeadBlockNumber          uint32 `json:"head_block_number"`
	HeadBlockID              string `json:"head_block_id"`
	Time                     string `json:"time"`
	LastIrreversibleBlockNum uint32 `json:"last_irreversible_block_num"`
	TotalVestingFundHive     string `json:"total_vesting_fund_hive"`
	TotalVestingShares       string `json:"total_vesting_shares"`
}

type TransactionData struct {
	TransactionID  string   `json:"transaction_id"`
	BlockNum       uint32   `json:"block_num"`
	TransactionNum uint32   `json:"transaction_num"`
	RefBlockNum    uint16   `json:"ref_block_num"`
	RefBlockPrefix uint32   `json:"ref_block_prefix"`
	Expiration     string   `json:"expiration"`
	Operations     []any    `json:"operations"`
	Extensions     []any    `json:"extensions"`
	Signatures     []string `json:"signatures"`
}

type State struct {
	db *badger.DB
}

// NewState creates a State instance.
// If dbPath is empty, it runs in pure-Go in-memory mode.
// If reset is true and dbPath is specified, it cleans the directory.
func NewState(dbPath string, reset bool) (*State, error) {
	if dbPath != "" && reset {
		if err := os.RemoveAll(dbPath); err != nil {
			return nil, fmt.Errorf("failed to reset database path: %w", err)
		}
	}

	var opt badger.Options
	if dbPath == "" {
		opt = badger.DefaultOptions("").WithInMemory(true)
	} else {
		opt = badger.DefaultOptions(dbPath)
	}
	opt = opt.WithLogger(nil) // suppress Badger's verbose logs

	db, err := badger.Open(opt)
	if err != nil {
		return nil, fmt.Errorf("failed to open badger DB: %w", err)
	}

	state := &State{db: db}
	if err := state.seedDefaults(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to seed defaults: %w", err)
	}

	return state, nil
}

func (s *State) Close() error {
	return s.db.Close()
}

// seedDefaults pre-populates default accounts and global properties.
func (s *State) seedDefaults() error {
	return s.db.Update(func(txn *badger.Txn) error {
		// 1. Seed Dynamic Global Properties if not present
		_, err := txn.Get([]byte("props"))
		if errors.Is(err, badger.ErrKeyNotFound) {
			props := DynamicProperties{
				HeadBlockNumber:          100000000,
				HeadBlockID:              "05f5e100f72d57fd5a542459a94f3a8153c68c4a",
				Time:                     time.Now().UTC().Format("2006-01-02T15:04:05"),
				LastIrreversibleBlockNum: 99999990,
				TotalVestingFundHive:     "200000000.000 HIVE",
				TotalVestingShares:       "400000000000.000000 VESTS",
			}
			bytes, _ := json.Marshal(props)
			if err := txn.Set([]byte("props"), bytes); err != nil {
				return err
			}
		}

		// Helper to seed an account if not present
		seedAcc := func(name, balance, hbd, vesting string) error {
			key := []byte("acc:" + name)
			_, err := txn.Get(key)
			if errors.Is(err, badger.ErrKeyNotFound) {
				activeWif, activePub, err := GenerateRandomKeyPair()
				if err != nil {
					return err
				}
				postingWif, postingPub, err := GenerateRandomKeyPair()
				if err != nil {
					return err
				}

				acc := AccountData{
					Name:        name,
					VotingPower: 10000,
					VotingManabar: Manabar{
						CurrentMana:    10000,
						LastUpdateTime: time.Now().Unix(),
					},
					LastVoteTime:      "1970-01-01T00:00:00",
					Balance:           balance,
					HbdBalance:        hbd,
					VestingShares:     vesting,
					Created:           "2016-03-24T16:00:00",
					ActiveKey:         activePub,
					ActivePrivateKey:  activeWif,
					PostingKey:        postingPub,
					PostingPrivateKey: postingWif,
					OwnerKey:          activePub,
					MemoKey:           activePub,
				}
				bytes, _ := json.Marshal(acc)
				if err := txn.Set(key, bytes); err != nil {
					return err
				}

				if err := txn.Set([]byte("key:"+activePub), []byte(name)); err != nil {
					return err
				}
				if err := txn.Set([]byte("key:"+postingPub), []byte(name)); err != nil {
					return err
				}
			}
			return nil
		}

		// Seed standard test accounts
		if err := seedAcc("alice", "500.000 HIVE", "100.000 HBD", "5000000.000000 VESTS"); err != nil {
			return err
		}

		if err := seedAcc("bob", "250.000 HIVE", "50.000 HBD", "2500000.000000 VESTS"); err != nil {
			return err
		}

		return nil
	})
}

func (s *State) GetAccount(name string) (*AccountData, error) {
	var acc AccountData
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("acc:" + name))
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &acc)
		})
	})
	if err != nil {
		return nil, err
	}
	return &acc, nil
}

func (s *State) SaveAccount(acc *AccountData) error {
	return s.db.Update(func(txn *badger.Txn) error {
		bytes, err := json.Marshal(acc)
		if err != nil {
			return err
		}
		return txn.Set([]byte("acc:"+acc.Name), bytes)
	})
}

func (s *State) ListAccounts() ([]AccountData, error) {
	var accounts []AccountData
	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = true
		it := txn.NewIterator(opts)
		defer it.Close()

		prefix := []byte("acc:")
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			var acc AccountData
			if err := item.Value(func(val []byte) error {
				return json.Unmarshal(val, &acc)
			}); err != nil {
				return err
			}
			accounts = append(accounts, acc)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(accounts, func(i, j int) bool {
		return accounts[i].Name < accounts[j].Name
	})
	return accounts, nil
}

func (s *State) GetKeyReferences(keys []string) ([]string, error) {
	var refs []string
	err := s.db.View(func(txn *badger.Txn) error {
		for _, k := range keys {
			item, err := txn.Get([]byte("key:" + k))
			if errors.Is(err, badger.ErrKeyNotFound) {
				continue
			}
			if err != nil {
				return err
			}
			err = item.Value(func(val []byte) error {
				refs = append(refs, string(val))
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return refs, nil
}

func (s *State) RegisterKey(key string, account string) error {
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte("key:"+key), []byte(account))
	})
}

func (s *State) GetContent(author, permlink string) (*PostData, error) {
	var post PostData
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("post:" + author + ":" + permlink))
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &post)
		})
	})
	if err != nil {
		return nil, err
	}
	return &post, nil
}

func (s *State) SaveContent(post *PostData) error {
	return s.db.Update(func(txn *badger.Txn) error {
		bytes, err := json.Marshal(post)
		if err != nil {
			return err
		}
		return txn.Set([]byte("post:"+post.Author+":"+post.Permlink), bytes)
	})
}

func (s *State) ListContent() ([]PostData, error) {
	var posts []PostData
	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = true
		it := txn.NewIterator(opts)
		defer it.Close()

		prefix := []byte("post:")
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			var post PostData
			if err := item.Value(func(val []byte) error {
				return json.Unmarshal(val, &post)
			}); err != nil {
				return err
			}
			posts = append(posts, post)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(posts, func(i, j int) bool {
		if posts[i].Created == posts[j].Created {
			if posts[i].Author == posts[j].Author {
				return posts[i].Permlink > posts[j].Permlink
			}
			return posts[i].Author > posts[j].Author
		}
		return posts[i].Created > posts[j].Created
	})
	return posts, nil
}

func (s *State) GetDynamicProperties() (*DynamicProperties, error) {
	var props DynamicProperties
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("props"))
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &props)
		})
	})
	if err != nil {
		return nil, err
	}
	if props.TotalVestingFundHive == "" {
		props.TotalVestingFundHive = "200000000.000 HIVE"
	}
	if props.TotalVestingShares == "" {
		props.TotalVestingShares = "400000000000.000000 VESTS"
	}
	return &props, nil
}

func (s *State) SaveDynamicProperties(props *DynamicProperties) error {
	return s.db.Update(func(txn *badger.Txn) error {
		bytes, err := json.Marshal(props)
		if err != nil {
			return err
		}
		return txn.Set([]byte("props"), bytes)
	})
}

func (s *State) GetTransaction(id string) (*TransactionData, error) {
	var tx TransactionData
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("tx:" + id))
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &tx)
		})
	})
	if err != nil {
		return nil, err
	}
	return &tx, nil
}

func (s *State) ListTransactions() ([]TransactionData, error) {
	var transactions []TransactionData
	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = true
		it := txn.NewIterator(opts)
		defer it.Close()

		prefix := []byte("tx:")
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			var txData TransactionData
			if err := item.Value(func(val []byte) error {
				return json.Unmarshal(val, &txData)
			}); err != nil {
				return err
			}
			transactions = append(transactions, txData)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(transactions, func(i, j int) bool {
		if transactions[i].BlockNum == transactions[j].BlockNum {
			return transactions[i].TransactionNum < transactions[j].TransactionNum
		}
		return transactions[i].BlockNum < transactions[j].BlockNum
	})
	return transactions, nil
}

func (s *State) SaveTransaction(tx *TransactionData) error {
	return s.db.Update(func(txn *badger.Txn) error {
		bytes, err := json.Marshal(tx)
		if err != nil {
			return err
		}
		return txn.Set([]byte("tx:"+tx.TransactionID), bytes)
	})
}

// GenerateRandomKeyPair generates a random private key (WIF) and its corresponding public key (STM...).
func GenerateRandomKeyPair() (string, string, error) {
	privKeyBytes := make([]byte, 32)
	_, err := rand.Read(privKeyBytes)
	if err != nil {
		return "", "", err
	}

	// Construct WIF: version 0x80 + private key (32 bytes) + compressed flag 0x01 (1 byte)
	data := make([]byte, 0, 34)
	data = append(data, 0x80)
	data = append(data, privKeyBytes...)
	data = append(data, 0x01)

	// Double SHA256 for checksum
	h1 := sha256.Sum256(data)
	h2 := sha256.Sum256(h1[:])
	checksum := h2[:4]

	wifBytes := append(data, checksum...)
	wifStr := anthercrypto.Base58Encode(wifBytes)

	// Derive public key from WIF
	pubKey, err := anthercrypto.WIFToPublicKey(wifStr)
	if err != nil {
		return "", "", err
	}

	return wifStr, pubKey, nil
}
