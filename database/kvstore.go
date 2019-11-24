package database

import (
	"encoding/binary"
	"encoding/json"

	"github.com/aquarelle-tech/darkmatter/types"
	"github.com/dgraph-io/badger"
)

const (
	// Prefixes indentify each key in the datastore
	HashKeyPrefix      = 0x1
	TimestampKeyPrefix = 0x2
	HeightKeyPrefix    = 0x3
	FixedKeyPrefix     = 0xFF // Any other key
)

// Implements the KVStore interface
type Store struct {
	StorFileLocation string
	storHandler      *badger.DB
}

// Creates a new store for key-value pairs
func NewKVStore(locationDirectory string) types.KVStore {

	// Open badger
	options := badger.DefaultOptions(locationDirectory)
	options.Truncate = true // To avoid problems with Windows. WARNING

	stor, err := badger.Open(options)
	if err != nil {
		panic(err)
	}

	kvs := &Store{
		StorFileLocation: locationDirectory,
		storHandler:      stor,
	}

	return kvs
}

// Store a value in the database indexed by an uint64
func storeUIntIndex(txn *badger.Txn, key uint64, value []byte, prefix byte) error {

	index := make([]byte, 8)
	binary.LittleEndian.PutUint64(index, key)
	index = append([]byte{prefix}, index...)

	return txn.Set(index, value)
}

// Read a value from the database indexed by an uint64
func readUIntIndex(txn *badger.Txn, key uint64, prefix byte) ([]byte, error) {

	index := make([]byte, 8)
	binary.LittleEndian.PutUint64(index, key)
	index = append([]byte{prefix}, index...)

	item, err := txn.Get(index)
	if err != nil {
		return nil, err
	}

	return item.ValueCopy(nil)
}

// Store a value in the database indexed by an uint64
func storeStringIndex(txn *badger.Txn, key string, value []byte, prefix byte) error {

	index := append([]byte{prefix}, []byte(key)...)
	return txn.Set(index, value)
}

// Read a value from the database indexed by an uint64
func readStringIndex(txn *badger.Txn, key string, prefix byte) ([]byte, error) {

	index := append([]byte{prefix}, []byte(key)...)
	item, err := txn.Get(index)
	if err != nil {
		return nil, err
	}

	return item.ValueCopy(nil)
}

// Store a full block in the database. The block will be indexed by their timestamp and Height
func (s Store) StoreBlock(block types.FullSignedBlock) error {

	// Serialize all the parts: block in json
	bytes, err := json.Marshal(block)

	err = s.storHandler.Update(func(txn *badger.Txn) error {

		var txErr error
		// Store the hash as a key. This is the main register
		if txErr = storeStringIndex(txn, block.Hash, bytes, HashKeyPrefix); txErr == nil {
			// And now store the indexes. Using this indexes it is possible to retrieve the hash, and next the block
			if txErr = storeUIntIndex(txn, block.Timestamp, []byte(block.Hash), TimestampKeyPrefix); txErr != nil { // By timestamp
				return txErr
			}

			if txErr = storeUIntIndex(txn, block.Height, []byte(block.Hash), HeightKeyPrefix); txErr != nil { // By block Height
				return txErr
			}
		}

		return txErr
	})

	return err
}

// Read a block from the database using their hash
func (s Store) GetBlock(hash string) (*types.FullSignedBlock, error) {

	var block types.FullSignedBlock
	err := s.storHandler.View(func(txn *badger.Txn) error {
		bytes, err := readStringIndex(txn, hash, HashKeyPrefix)
		if err != nil {
			return err
		}
		err = json.Unmarshal(bytes, &block)

		return err
	})

	return &block, err
}

// Read a block from the database using their timestamp as index
func (s Store) FindBlockByTimestamp(timestamp uint64) (*types.FullSignedBlock, error) {

	var block types.FullSignedBlock
	err := s.storHandler.View(func(txn *badger.Txn) error {
		// retrieve the the indexed hash
		hashBytes, err := readUIntIndex(txn, timestamp, TimestampKeyPrefix)
		if err != nil {
			return err
		}
		// Get the hash from the result and look for the block
		bytes, err := readStringIndex(txn, string(hashBytes), HashKeyPrefix)
		if err != nil {
			return err
		}
		err = json.Unmarshal(bytes, &block)

		return err
	})

	return &block, err
}

// Read a block from the database using their timestamp as index
func (s Store) FindBlockByHeight(height uint64) (*types.FullSignedBlock, error) {

	var block types.FullSignedBlock
	err := s.storHandler.View(func(txn *badger.Txn) error {
		// retrieve the the indexed hash
		hashBytes, err := readUIntIndex(txn, height, HeightKeyPrefix)
		if err != nil {
			return err
		}
		// Get the hash from the result and look for the block
		bytes, err := readStringIndex(txn, string(hashBytes), HashKeyPrefix)
		if err != nil {
			return err
		}
		err = json.Unmarshal(bytes, &block)

		return err
	})

	return &block, err
}

// StoreValue stores an abritrary value in the database, indexed by a string
func (s Store) StoreValue(key string, value []byte) error {

	err := s.storHandler.Update(func(txn *badger.Txn) error {
		return storeStringIndex(txn, key, value, FixedKeyPrefix)
	})

	return err
}

// GetValue returns a value stored in the database indexed by an string
func (s *Store) GetValue(key string) ([]byte, error) {

	var bytes []byte
	var err error

	err = s.storHandler.Update(func(txn *badger.Txn) error {
		bytes, err = readStringIndex(txn, key, FixedKeyPrefix)

		return err
	})

	return bytes, err
}
