package store

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type NodeType string

const (
	NodeSS       NodeType = "ss"
	NodeVMess    NodeType = "vmess"
	NodeTrojan   NodeType = "trojan"
	NodeVLESS    NodeType = "vless"
	NodeHysteria NodeType = "hysteria2"
	NodeUnknown  NodeType = "unknown"
)

type Node struct {
	ID      string            `json:"id"`
	Name    string            `json:"name"`
	Type    NodeType          `json:"type"`
	Server  string            `json:"server"`
	Port    int               `json:"port"`
	Country string            `json:"country"`
	RawURI  string            `json:"raw_uri"`
	Params  map[string]string `json:"params"`
}

type SourceType string

const (
	SourceURL   SourceType = "url"
	SourceFile  SourceType = "file"
	SourceLocal SourceType = "local"
)

type Subscription struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	URL          string                 `json:"url"`
	SourceType   SourceType             `json:"source_type"`
	LocalContent string                 `json:"local_content,omitempty"`
	Enabled      bool                   `json:"enabled"`
	Nodes        []Node                 `json:"nodes"`
	UpdatedAt    time.Time              `json:"updated_at"`
	NodeCount    int                    `json:"node_count"`
	Params       map[string]string      `json:"params,omitempty"`
	Extra        map[string]interface{} `json:"extra,omitempty"` // stores Sub-Store fields like displayName, tag, etc.
}

type ExpireUnit string

const (
	ExpireDay     ExpireUnit = "day"
	ExpireMonth   ExpireUnit = "month"
	ExpireQuarter ExpireUnit = "quarter"
	ExpireYear    ExpireUnit = "year"
)

type Collection struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	SubIDs      []string               `json:"sub_ids"`
	Token       string                 `json:"token"`
	Enabled     bool                   `json:"enabled"`
	ExpireValue int                    `json:"expire_value,omitempty"`
	ExpireUnit  ExpireUnit             `json:"expire_unit,omitempty"`
	ExpireAt    *time.Time             `json:"expire_at,omitempty"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Extra       map[string]interface{} `json:"extra,omitempty"`
}

func (c *Collection) IsExpired() bool {
	if c.ExpireAt == nil {
		return false
	}
	return time.Now().After(*c.ExpireAt)
}

// Token represents a share token (Sub-Store compat)
type Token struct {
	ID          string  `json:"id"`
	Type        string  `json:"type"`   // "sub" | "col" | "file"
	Name        string  `json:"name"`   // subscription/collection name
	DisplayName string  `json:"displayName,omitempty"`
	Remark      string  `json:"remark,omitempty"`
	Token       string  `json:"token"`
	CreatedAt   int64   `json:"createdAt"`
	Exp         *int64  `json:"exp,omitempty"`
}

type Data struct {
	Subscriptions []Subscription `json:"subscriptions"`
	Collections   []Collection   `json:"collections"`
	Tokens        []Token        `json:"tokens"`
}

type Store struct {
	mu       sync.RWMutex
	data     Data
	filePath string
	dataDir  string
}

func New(dataDir string) (*Store, error) {
	if dataDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = "."
		}
		dataDir = filepath.Join(home, ".sub-store")
	}
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}
	s := &Store{
		filePath: filepath.Join(dataDir, "data.json"),
		dataDir:  dataDir,
	}
	if err := s.load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return s, nil
}

func (s *Store) DataDir() string { return s.dataDir }

func (s *Store) load() error {
	b, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, &s.data)
}

func (s *Store) save() error {
	b, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.filePath, b, 0644)
}

// ── Subscriptions ─────────────────────────────────────────────────────────────

func (s *Store) GetSubscriptions() []Subscription {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Subscription, len(s.data.Subscriptions))
	for i, sub := range s.data.Subscriptions {
		sub.NodeCount = len(sub.Nodes)
		result[i] = sub
	}
	return result
}

func (s *Store) GetSubscription(id string) (*Subscription, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, sub := range s.data.Subscriptions {
		if sub.ID == id {
			cp := sub
			return &cp, true
		}
	}
	return nil, false
}

func (s *Store) GetSubscriptionByName(name string) (*Subscription, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, sub := range s.data.Subscriptions {
		if sub.Name == name {
			cp := sub
			return &cp, true
		}
	}
	return nil, false
}

func (s *Store) AddSubscription(sub Subscription) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Subscriptions = append(s.data.Subscriptions, sub)
	return s.save()
}

func (s *Store) UpdateSubscription(sub Subscription) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, existing := range s.data.Subscriptions {
		if existing.ID == sub.ID {
			s.data.Subscriptions[i] = sub
			return s.save()
		}
	}
	return nil
}

func (s *Store) DeleteSubscription(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, sub := range s.data.Subscriptions {
		if sub.ID == id {
			s.data.Subscriptions = append(s.data.Subscriptions[:i], s.data.Subscriptions[i+1:]...)
			return s.save()
		}
	}
	return nil
}

func (s *Store) ReorderSubscriptions(names []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	nameIdx := make(map[string]int, len(names))
	for i, n := range names {
		nameIdx[n] = i
	}
	ordered := make([]Subscription, 0, len(s.data.Subscriptions))
	// add in order
	for _, n := range names {
		for _, sub := range s.data.Subscriptions {
			if sub.Name == n {
				ordered = append(ordered, sub)
				break
			}
		}
	}
	// add any not in list
	for _, sub := range s.data.Subscriptions {
		if _, ok := nameIdx[sub.Name]; !ok {
			ordered = append(ordered, sub)
		}
	}
	s.data.Subscriptions = ordered
	s.save()
}

func (s *Store) GetAllNodes(subIDs []string) []Node {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.getNodesLocked(subIDs)
}

func (s *Store) getNodesLocked(subIDs []string) []Node {
	var nodes []Node
	subSet := make(map[string]bool, len(subIDs))
	for _, id := range subIDs {
		subSet[id] = true
	}
	for _, sub := range s.data.Subscriptions {
		if !sub.Enabled {
			continue
		}
		if len(subIDs) > 0 && !subSet[sub.ID] {
			continue
		}
		nodes = append(nodes, sub.Nodes...)
	}
	return nodes
}

// ── Collections ────────────────────────────────────────────────────────────────

func (s *Store) GetCollections() []Collection {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Collection, len(s.data.Collections))
	copy(result, s.data.Collections)
	return result
}

func (s *Store) GetCollection(id string) (*Collection, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, c := range s.data.Collections {
		if c.ID == id {
			cp := c
			return &cp, true
		}
	}
	return nil, false
}

func (s *Store) GetCollectionByName(name string) (*Collection, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, c := range s.data.Collections {
		if c.Name == name {
			cp := c
			return &cp, true
		}
	}
	return nil, false
}

func (s *Store) GetCollectionByToken(token string) (*Collection, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, c := range s.data.Collections {
		if c.Token == token && c.Enabled {
			cp := c
			return &cp, true
		}
	}
	return nil, false
}

func (s *Store) AddCollection(c Collection) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Collections = append(s.data.Collections, c)
	return s.save()
}

func (s *Store) UpdateCollection(c Collection) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, existing := range s.data.Collections {
		if existing.ID == c.ID {
			s.data.Collections[i] = c
			return s.save()
		}
	}
	return nil
}

func (s *Store) DeleteCollection(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, c := range s.data.Collections {
		if c.ID == id {
			s.data.Collections = append(s.data.Collections[:i], s.data.Collections[i+1:]...)
			return s.save()
		}
	}
	return nil
}

func (s *Store) ReorderCollections(names []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	nameIdx := make(map[string]int, len(names))
	for i, n := range names {
		nameIdx[n] = i
	}
	ordered := make([]Collection, 0, len(s.data.Collections))
	for _, n := range names {
		for _, col := range s.data.Collections {
			if col.Name == n {
				ordered = append(ordered, col)
				break
			}
		}
	}
	for _, col := range s.data.Collections {
		if _, ok := nameIdx[col.Name]; !ok {
			ordered = append(ordered, col)
		}
	}
	s.data.Collections = ordered
	s.save()
}

func (s *Store) GetCollectionNodes(col *Collection) []Node {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.getNodesLocked(col.SubIDs)
}

// ── Tokens ────────────────────────────────────────────────────────────────────

func (s *Store) GetTokens(typef, name string) []Token {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []Token
	for _, t := range s.data.Tokens {
		if typef != "" && t.Type != typef {
			continue
		}
		if name != "" && t.Name != name {
			continue
		}
		result = append(result, t)
	}
	if result == nil {
		result = []Token{}
	}
	return result
}

func (s *Store) GetTokenByValue(token string) (*Token, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, t := range s.data.Tokens {
		if t.Token == token {
			cp := t
			return &cp, true
		}
	}
	return nil, false
}

func (s *Store) AddToken(t Token) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Tokens = append(s.data.Tokens, t)
	return s.save()
}

func (s *Store) DeleteToken(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, t := range s.data.Tokens {
		if t.Token == token {
			s.data.Tokens = append(s.data.Tokens[:i], s.data.Tokens[i+1:]...)
			s.save()
			return
		}
	}
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func NewID() string {
	b := make([]byte, 6)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func NewToken() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func CalcExpireAt(value int, unit ExpireUnit) *time.Time {
	if value <= 0 {
		return nil
	}
	now := time.Now()
	var t time.Time
	switch unit {
	case ExpireDay:
		t = now.AddDate(0, 0, value)
	case ExpireMonth:
		t = now.AddDate(0, value, 0)
	case ExpireQuarter:
		t = now.AddDate(0, value*3, 0)
	case ExpireYear:
		t = now.AddDate(value, 0, 0)
	default:
		t = now.AddDate(0, 0, value)
	}
	return &t
}
