package raft

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/boltdb/bolt"
	"github.com/hashicorp/raft"
)
const (
	DirPath = "assets/db"
	BUCKET_NAME = "kvstore"
)

const (
	OpSet = "SET"
	OpGet = "GET"
	OpUpdate = "UPDATE"
	OpDelete = "DELETE"
)

type RaftFSM struct {
	DB       *bolt.DB
	ServerId raft.ServerID
}

type FSMSnapshot struct {
	Tx *bolt.Tx
}

// type KV struct {
// 	Key []byte `json:"key"`
// 	Value []byte `json:"value"`
// }

type KVOp struct {
	*KV
	Op    string `json:"op"`
}

type FSMResponse struct {
	Kv KV
	Error error
}

func NewRaftFSM(id string) (*RaftFSM, error) {
	dirPath, err := MakeDirectory(id)
	if err != nil {
		return nil, fmt.Errorf("error making directory for fsm storage: %w", err)
	}
	path := filepath.Join(dirPath, "store.db")
	db, err := bolt.Open(path, 0600, nil)
	if err != nil {
		return nil, fmt.Errorf("error opening up db: %w", err)
	}
	fsm := &RaftFSM{ServerId: raft.ServerID(id), DB: db}

	err = fsm.CreateBucket()
	if err != nil {
		return nil, fmt.Errorf("errror creating bucket: %w", err)
	}

	return fsm, nil
}

func (r *RaftFSM) Apply(log *raft.Log) interface{} {
	
	kvOp, err := ParseData(log.Data)
	if err != nil {
		return FSMResponse{
			Kv: KV{},
			Error: fmt.Errorf("error parsing the log: %w", err),
		}
	}

	switch kvOp.Op{
	case OpSet, OpUpdate:
		kv, err := r.Update(kvOp)
		if err != nil {
			return FSMResponse{Kv: KV{}, Error: fmt.Errorf("error setting key:value: %w", err)}
		}
		return FSMResponse{Kv: kv, Error: nil}
	case OpDelete:
		err := r.Delete(kvOp.Key)
		if err != nil {
			return FSMResponse{Kv: KV{}, Error: fmt.Errorf("error deleting key: value: %w", err)}
		}
		return FSMResponse{Kv: KV{}, Error: nil}
	default:
		return FSMResponse{Kv: KV{}, Error: errors.New("invalid operation")}
	}
}


func (r *RaftFSM) Update(k *KVOp) (KV, error) {
	err := r.DB.Update(func(tx *bolt.Tx) error {
		err := tx.Bucket([]byte(BUCKET_NAME)).Put(k.Key, k.Value)
		if err != nil {
			return err
		}
		err = tx.Commit()
		if err != nil {
			return fmt.Errorf("error commititng update to db: %w", err)
		}
		return nil
	})

	if err != nil {
		return KV{}, fmt.Errorf("error updating key, value: %w", err)
	}
	return *k.KV, nil
}

func (r *RaftFSM) Get(key []byte) (KV, error) {
	var kv = KV{}
	err := r.DB.View(func(tx *bolt.Tx) error {
		value := tx.Bucket([]byte(BUCKET_NAME)).Get(key)
		if value == nil {
			return errors.New("error finding value")
		}
		kv.Key = key
		kv.Value = make([]byte, len(value))
		copy(kv.Value, value)
		err := tx.Rollback()
		if err != nil {
			return fmt.Errorf("error roll back get from db: %w", err)
		}
		return nil
	})
	return kv, err
}

func (r *RaftFSM) Delete(key []byte) error {
	err := r.DB.Update(func(tx *bolt.Tx) error {
		err := tx.Bucket([]byte(BUCKET_NAME)).Delete(key)
		if err != nil {
			return err
		}
		err = tx.Commit()
		if err != nil {
			return fmt.Errorf("error commititng update to db: %w", err)
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (r *RaftFSM) Snapshot() (raft.FSMSnapshot, error) {
	tx, err := r.DB.Begin(false)
	if err != nil {
		return nil, fmt.Errorf("error starting a transaction for the fsm snapshot: %w", err)
	}
	fsmSnapshot := &FSMSnapshot{Tx: tx}
	return fsmSnapshot, nil
}



func (r *RaftFSM) Restore(reader io.ReadCloser) error {
	
	dbPath := r.DB.Path()
	err := r.DB.Close()
	if err != nil {
		return fmt.Errorf("error closing db for restoration: %w", err)
	}

	file, err := os.Create(dbPath)
	if err != nil {
		return fmt.Errorf("error opening up db file: %w", err)
	}
	defer file.Close()

	_, err = io.Copy(file, reader)
	if err != nil {
		return fmt.Errorf("error copying from snapshot to db: %w", err)
	}

	r.DB, err = bolt.Open(dbPath, 0600, nil)
	if err != nil {
		return fmt.Errorf("error opening database at the path: %w", err)
	}

	return nil
}

func (r *RaftFSM) CreateBucket() error {
	err := r.DB.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(BUCKET_NAME))
		return err
	})
	return err
}

func (fsm *FSMSnapshot) Persist(sink raft.SnapshotSink) error {

	_, err := fsm.Tx.WriteTo(sink)
	if err != nil {
		sink.Cancel()
		return fmt.Errorf("error writing to snpashot sink: %w", err)
	}

	return sink.Close()
}

func (fsm *FSMSnapshot) Release() {
	fsm.Tx.Rollback()
}


func MakeDirectory(id string) (string, error) {
	path := filepath.Join(DirPath, id, "storage")

	err := os.MkdirAll(path, 0700)
	if err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	return path, nil
}

func ParseData(data []byte) (*KVOp, error) {
	var kv = &KVOp{}

	err := json.Unmarshal(data, kv)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal log: %w", err)
	}

	return kv, nil
}
