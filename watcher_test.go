package watcher

import (
	"bytes"
	"testing"
)

type key int

const (
	filesKey key = iota
	globKey
)

func TestFiles(t *testing.T) {
	if err := RegisterFiles(filesKey, "test.html"); err != nil {
		t.Error(err)
	}
	if err := RegisterBaseFiles("base.html"); err != nil {
		t.Error(err)
	}
	temp, err := Get(filesKey)
	if err != nil {
		t.Fatal(err)
	}
	b := new(bytes.Buffer)
	if err := temp.ExecuteTemplate(b, "base.html", nil); err != nil {
		t.Error(err)
	}
	if bytes.Compare(b.Bytes(), []byte("Base Test")) != 0 {
		t.Fatalf("expected %q, got %q", "Base Test", b.String())
	}
}

func TestGlob(t *testing.T) {
	if err := RegisterBaseGlob("base.html"); err != nil {
		t.Error(err)
	}
	if err := RegisterGlob(globKey, "test.html"); err != nil {
		t.Error(err)
	}
	temp, err := Get(globKey)
	if err != nil {
		t.Error(err)
	}
	b := new(bytes.Buffer)
	if err := temp.Execute(b, nil); err != nil {
		t.Error(err)
	}
	if bytes.Compare(b.Bytes(), []byte("Base Test")) != 0 {
		t.Fatalf("expected %q, got %q", "Base Test", b.String())
	}
}
