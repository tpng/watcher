package watcher

import (
	"fmt"
	"html/template"
	"log"
	"path/filepath"
	"sync"
	"time"
)

type watched struct {
	filenames []string
	template  *template.Template
	cached    time.Time
}

var (
	cache     = make(map[interface{}]*watched)
	cacheLock sync.RWMutex
)

var (
	baseChanged time.Time
)

func RegisterFiles(key interface{}, filenames ...string) error {
	w, err := parseFiles(filenames...)
	if err != nil {
		return err
	}

	setChan <- &cacheSet{
		key: key,
		w:   w,
	}

	return nil
}

func mergeTemplate(base *template.Template, t *template.Template) (*template.Template, error) {
	nt, err := base.Clone()
	if err != nil {
		return nil, err
	}

	for _, sub := range t.Templates() {
		if _, err := nt.AddParseTree(sub.Name(), sub.Tree); err != nil {
			return nil, err
		}
	}

	return nt, nil
}

func RegisterGlob(key interface{}, pattern string) error {
	filenames, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}
	if len(filenames) == 0 {
		return fmt.Errorf("watcher: pattern matches no files: %#q", pattern)
	}
	return RegisterFiles(key, filenames...)
}

func Get(key interface{}) (*template.Template, error) {
	c := make(chan *template.Template, 1)
	getChan <- &cacheGet{
		key: key,
		c:   c,
	}
	t, ok := <-c
	if !ok {
		return nil, fmt.Errorf("watcher: template not found with key: %T=%v", key, key)
	}

	return t, nil
}

type cacheKey int

const baseKey cacheKey = 0

func RegisterBaseFiles(filenames ...string) error {
	w, err := parseBaseFiles(filenames...)
	if err != nil {
		return err
	}

	setChan <- &cacheSet{
		key: baseKey,
		w:   w,
	}

	return nil
}

func RegisterBaseGlob(pattern string) error {
	filenames, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}
	if len(filenames) == 0 {
		return fmt.Errorf("watcher: pattern matches no files: %#q", pattern)
	}
	return RegisterBaseFiles(filenames...)
}

type cacheGet struct {
	key interface{}
	c   chan *template.Template
}

type cacheSet struct {
	key interface{}
	w   *watched
}

func watcher() {
	for {
		select {
		case g := <-getChan:
			go get(g.key, g.c)
		case s := <-setChan:
			set(s.key, s.w)
		}
	}
}

var getChan = make(chan *cacheGet, 10)
var setChan = make(chan *cacheSet, 10)

func set(key interface{}, w *watched) {
	cacheLock.Lock()
	cache[key] = w
	cacheLock.Unlock()
}

func get(key interface{}, c chan<- *template.Template) {
	defer close(c)
	cacheLock.RLock()
	w, ok := cache[key]
	cacheLock.RUnlock()
	if !ok {
		return
	}
	changed := getChangeTime(w.filenames...)
	if changed.After(w.cached) || (key != baseKey && baseChanged.After(w.cached)) {
		var err error
		if key == baseKey {
			w, err = parseBaseFiles(w.filenames...)
		} else {
			w, err = parseFiles(w.filenames...)
		}
		if err != nil {
			log.Println(err)
			return
		}
		setChan <- &cacheSet{
			key: key,
			w:   w,
		}
	}
	c <- w.template
}

func init() {
	go watcher()
}

func getChangeTime(filenames ...string) time.Time {
	return time.Now().Add(-time.Minute)
}

func parseFiles(filenames ...string) (*watched, error) {
	t, err := template.ParseFiles(filenames...)
	if err != nil {
		return nil, err
	}

	if base, err := Get(baseKey); err == nil {
		if t, err = mergeTemplate(base, t); err != nil {
			return nil, err
		}
	}

	return &watched{
		filenames: filenames,
		template:  t,
		cached:    time.Now(),
	}, nil
}

func parseBaseFiles(filenames ...string) (*watched, error) {
	t, err := template.ParseFiles(filenames...)
	if err != nil {
		return nil, err
	}

	baseChanged = time.Now().Add(time.Second)

	return &watched{
		filenames: filenames,
		template:  t,
		cached:    time.Now(),
	}, nil
}
