package watcher

import (
	"bytes"
	"testing"
)

type key int

const (
	filesKey key = iota
	globKey
	partialKey
)

func TestBaseFilesThenFiles(t *testing.T) {
	if err := RegisterBaseFiles("base.html"); err != nil {
		t.Error(err)
	}
	if err := RegisterFiles(filesKey, "test.html"); err != nil {
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

func TestFilesThenBaseFiles(t *testing.T) {
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

func TestBaseGlobThenGlob(t *testing.T) {
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
	if err := temp.ExecuteTemplate(b, "base.html", nil); err != nil {
		t.Error(err)
	}
	if bytes.Compare(b.Bytes(), []byte("Base Test")) != 0 {
		t.Fatalf("expected %q, got %q", "Base Test", b.String())
	}
}

func TestGlobThenBaseGlob(t *testing.T) {
	if err := RegisterGlob(globKey, "test.html"); err != nil {
		t.Error(err)
	}
	if err := RegisterBaseGlob("base.html"); err != nil {
		t.Error(err)
	}
	temp, err := Get(globKey)
	if err != nil {
		t.Error(err)
	}
	b := new(bytes.Buffer)
	if err := temp.ExecuteTemplate(b, "base.html", nil); err != nil {
		t.Error(err)
	}
	if bytes.Compare(b.Bytes(), []byte("Base Test")) != 0 {
		t.Fatalf("expected %q, got %q", "Base Test", b.String())
	}
}

func TestPartial(t *testing.T) {
	DelimLeft, DelimRight = "[[", "]]"
	if err := RegisterBaseGlob("partial/*.html"); err != nil {
		t.Fatal(err)
	}
	if err := RegisterFiles(partialKey, "job.html"); err != nil {
		t.Fatal(err)
	}
	temp, err := Get(partialKey)
	if err != nil {
		t.Fatal(err)
	}
	b := new(bytes.Buffer)
	type Job struct {
		ID    string
		Title string
	}
	if err := temp.ExecuteTemplate(b, "job.html", &Job{ID: "1", Title: "Test"}); err != nil {
		t.Fatal(err)
	}
	if bytes.Compare(b.Bytes(), []byte("1, Test")) != 0 {
		t.Fatalf("expected %q, got %q", "1, Test", b.String())
	}
}
