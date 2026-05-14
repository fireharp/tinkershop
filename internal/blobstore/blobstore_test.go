package blobstore

import (
	"context"
	"os"
	"testing"
)

func TestPutStoresCompressedBlob(t *testing.T) {
	store := Store{Root: t.TempDir(), Compression: "gzip"}
	meta, err := store.Put(context.Background(), "text/plain", []byte("hello tinkershop"))
	if err != nil {
		t.Fatal(err)
	}
	if meta.SHA256 == "" {
		t.Fatal("missing sha")
	}
	if meta.Compression != "gzip" {
		t.Fatalf("compression = %q", meta.Compression)
	}
	if _, err := os.Stat(meta.Path); err != nil {
		t.Fatal(err)
	}
}
