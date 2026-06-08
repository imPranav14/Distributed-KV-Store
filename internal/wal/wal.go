package wal

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/imPranav14/Distributed-KV-Store/internal/store"
)

const (
	lengthFieldSize = 8
	maxRecordSize   = 16 << 20 // 16 MiB
)

type WAL struct {
	mu     sync.Mutex
	file   *os.File
	writer *bufio.Writer
	path   string
}

type WALStore struct {
	wal   *WAL
	store *store.Store
}

func Open(path string) (*WAL, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create wal dir: %w", err)
	}

	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open wal file: %w", err)
	}

	return &WAL{
		file:   file,
		writer: bufio.NewWriter(file),
		path:   path,
	}, nil
}

func NewWALStore(store *store.Store, wal *WAL) *WALStore {
	if store == nil {
		panic("WALStore requires a non-nil store")
	}
	if wal == nil {
		panic("WALStore requires a non-nil WAL")
	}
	return &WALStore{wal: wal, store: store}
}

func (w *WAL) Append(entry *Entry) error {
	if entry == nil {
		return fmt.Errorf("wal entry cannot be nil")
	}

	data, err := entry.Marshal()
	if err != nil {
		return fmt.Errorf("marshal wal entry: %w", err)
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	var lenBuf [lengthFieldSize]byte
	binary.BigEndian.PutUint64(lenBuf[:], uint64(len(data)))

	if _, err := w.writer.Write(lenBuf[:]); err != nil {
		return fmt.Errorf("write wal length: %w", err)
	}
	if _, err := w.writer.Write(data); err != nil {
		return fmt.Errorf("write wal record: %w", err)
	}
	if err := w.writer.Flush(); err != nil {
		return fmt.Errorf("flush wal writer: %w", err)
	}
	if err := w.file.Sync(); err != nil {
		return fmt.Errorf("sync wal file: %w", err)
	}

	return nil
}

func (w *WAL) Replay(replayFn func(*Entry) error) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if err := w.writer.Flush(); err != nil {
		return fmt.Errorf("flush wal writer before replay: %w", err)
	}
	if _, err := w.file.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("seek wal file: %w", err)
	}

	reader := bufio.NewReader(w.file)
	for {
		var lenBuf [lengthFieldSize]byte
		if _, err := io.ReadFull(reader, lenBuf[:]); err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				return nil
			}
			return fmt.Errorf("read wal length: %w", err)
		}

		recordSize := binary.BigEndian.Uint64(lenBuf[:])
		if recordSize == 0 {
			return fmt.Errorf("wal record has zero length")
		}
		if recordSize > maxRecordSize {
			return fmt.Errorf("wal record size %d exceeds max %d", recordSize, maxRecordSize)
		}

		record := make([]byte, recordSize)
		if _, err := io.ReadFull(reader, record); err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				return nil
			}
			return fmt.Errorf("read wal record: %w", err)
		}

		entry, err := UnmarshalEntry(record)
		if err != nil {
			return fmt.Errorf("unmarshal wal entry: %w", err)
		}
		if err := replayFn(entry); err != nil {
			return err
		}
	}
}

func (w *WAL) ReplayInto(store *store.Store) error {
	return w.Replay(func(entry *Entry) error {
		switch entry.Type {
		case OpTypePut:
			w.storePut(store, entry)
		case OpTypeAppend:
			w.storeAppend(store, entry)
		default:
			return fmt.Errorf("unsupported wal op type %v", entry.Type)
		}
		return nil
	})
}

func (w *WAL) storePut(store *store.Store, entry *Entry) {
	store.Put(entry.Key, entry.Value)
}

func (w *WAL) storeAppend(store *store.Store, entry *Entry) {
	store.Append(entry.Key, entry.Value)
}

func (w *WAL) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.writer != nil {
		if err := w.writer.Flush(); err != nil {
			return err
		}
	}
	if w.file != nil {
		return w.file.Close()
	}
	return nil
}

func (s *WALStore) Get(key string) (string, bool) {
	return s.store.Get(key)
}

func (s *WALStore) Put(key, value string) error {
	if err := s.wal.Append(&Entry{Type: OpTypePut, Key: key, Value: value}); err != nil {
		return err
	}
	s.store.Put(key, value)
	return nil
}

func (s *WALStore) Append(key, value string) error {
	if err := s.wal.Append(&Entry{Type: OpTypeAppend, Key: key, Value: value}); err != nil {
		return err
	}
	s.store.Append(key, value)
	return nil
}
