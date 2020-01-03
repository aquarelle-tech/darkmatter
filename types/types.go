// Package types
package types

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"log"
)

const (
	// ServiceHash is a random number to be included in all the signs to enforce the hashes creation
	ServiceHash = "1d0684170dcf58ed2499d233be72b5dde48d8124cb617f1309bae85da2fe85cf"
	// BlockHashPrefix is the standard prefix used in DarkMatter protocol to recognize their blocks hashes
	BlockHashPrefix = "dd"
)

// KVStore defines a KV pair storage manager definition
type KVStore interface {
	StoreValue(key string, value []byte) error // StoreValue saves a unespecified instance using an string as key
	GetValue(key string) ([]byte, error)
	StoreBlock(block FullSignedBlock) error
	GetBlock(hash string) (*FullSignedBlock, error)
	GetLatestBlocks(timestamp uint64, n int) ([]FullSignedBlock, error)
	FindBlockByTimestamp(timestamp uint64) (*FullSignedBlock, error)
	FindBlockByHeight(Height uint64) (*FullSignedBlock, error)
}

// LiteIndexValueMessage is the message model used to be send to users and index the blocks
type LiteIndexValueMessage struct {
	Hash   string `json:"hash"`
	Height uint64 `json:"height"`
	// PriceIndex    float64 `json:"priceIndex"`
	// Quoted        string  `json:"quote"`
	NodeAddress   string `json:"nodeAddress"`
	Timestamp     uint64 `json:"timestamp"`
	Confirmations int    `json:"confirmations"`
}

// FullSignedBlock is the message to send to the connected clients through websocket
type FullSignedBlock struct {
	Hash      string      `json:"hash"`
	Height    uint64      `json:"height"`
	Timestamp uint64      `json:"timestamp"`
	Payload   interface{} `json:"data"`

	// AveragePrice  float64 `json:"avgPrice"`
	// AverageVolume float64 `json:"avgVolumen"`
	// Ticker        string  `json:"ticker"`

	PreviousHash    string   `json:"previousHash"`
	Address         string   `json:"address"`
	PreviousAddress string   `json:"previousAddress"`
	Memo            string   `json:"memo"`
	Evidence        []Result `json:"evidence"`
}

// CreateHash calculates the hash for a block
func (block *FullSignedBlock) CreateHash() error {

	// create a hash the result
	block.Hash = "" // To asure a clean hash
	hash, err := calculateHash(block)
	if err == nil {
		block.Hash = hash
	}

	// The hashes for the block has attached a prefix and the the number of seconds taken from the timestamp
	seconds := time.Unix(int64(block.Timestamp), 0).Second()
	block.Hash = fmt.Sprintf("%s%02d%s", BlockHashPrefix, seconds, block.Hash)

	return err // No error
}

// Implement the Stringer interface
func (block FullSignedBlock) String() string {
	bytes, err := json.Marshal(block)
	if err != nil {
		log.Fatal("Error deserializing a block", err)
	}

	return string(bytes)
}

// GetDataJob is the job message to insert in a queue to be processed as part of the the Mapping Stage
type GetDataJob struct {
	Quote       string
	DataCrawler interface{}
}

// Result is the message that will receive the results from the mapped nodes in the Reduce Stage
type Result struct {
	CrawlerName string `json:"name"`
	Data        []byte `json:"data"`
	HasError    bool   `json:"hasError"`
	Timestamp   int64  `json:"timestamp"`
	Ticker      string `json:"ticker"`
	Hash        string `json:"hash"`
}

// CreateHash creates a double hash (sha256(sha256)) for all the content
func (result *Result) CreateHash() error {
	// create a hash the result
	result.Hash = "" // To asure a clean hash
	hash, err := calculateHash(result)

	if err == nil {
		result.Hash = hash
	}

	return err // No error
}

// Generate a hash using a double operation over the serialized content of object
func calculateHash(obj interface{}) (string, error) {
	bytes, err := json.Marshal(obj)
	if err != nil {
		log.Println("Error serializing message", err)
		return "", err
	}
	// Sign the content of block including the hash of DarkMatter
	rawContent := fmt.Sprintf("%s:%s", ServiceHash, bytes)

	// Double hash for the content
	doubleHash := fmt.Sprintf("%x", sha256.Sum256([]byte(rawContent)))
	doubleHash = fmt.Sprintf("%x", sha256.Sum256([]byte(doubleHash)))

	return doubleHash, nil
}
