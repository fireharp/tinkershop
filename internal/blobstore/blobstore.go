package blobstore

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const compressionGzip = "gzip"

type Store struct {
	Root        string
	Compression string
}

type Meta struct {
	SHA256            string
	Path              string
	Compression       string
	MediaType         string
	BytesUncompressed int64
	BytesStored       int64
	CreatedAt         time.Time
}

func (s Store) Put(ctx context.Context, mediaType string, data []byte) (Meta, error) {
	select {
	case <-ctx.Done():
		return Meta{}, ctx.Err()
	default:
	}

	compression := s.Compression
	if compression == "" {
		compression = compressionGzip
	}

	sum := sha256.Sum256(data)
	sha := hex.EncodeToString(sum[:])

	stored := data
	ext := ".blob"
	if compression == compressionGzip {
		var buf bytes.Buffer
		zw := gzip.NewWriter(&buf)
		if _, err := zw.Write(data); err != nil {
			return Meta{}, err
		}
		if err := zw.Close(); err != nil {
			return Meta{}, err
		}
		stored = buf.Bytes()
		ext = ".gz"
	} else if compression != "none" {
		return Meta{}, fmt.Errorf("unsupported compression %q", compression)
	}

	dir := filepath.Join(s.Root, sha[:2])
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Meta{}, err
	}
	path := filepath.Join(dir, sha+ext)
	if _, err := os.Stat(path); err == nil {
		info, statErr := os.Stat(path)
		if statErr != nil {
			return Meta{}, statErr
		}
		return Meta{
			SHA256:            sha,
			Path:              path,
			Compression:       compression,
			MediaType:         mediaType,
			BytesUncompressed: int64(len(data)),
			BytesStored:       info.Size(),
			CreatedAt:         info.ModTime(),
		}, nil
	}

	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return Meta{}, err
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	if _, err := tmp.Write(stored); err != nil {
		_ = tmp.Close()
		return Meta{}, err
	}
	if err := tmp.Close(); err != nil {
		return Meta{}, err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return Meta{}, err
	}

	return Meta{
		SHA256:            sha,
		Path:              path,
		Compression:       compression,
		MediaType:         mediaType,
		BytesUncompressed: int64(len(data)),
		BytesStored:       int64(len(stored)),
		CreatedAt:         time.Now().UTC(),
	}, nil
}
