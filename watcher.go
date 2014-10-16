/*
Package watcher implements caching and live-reload of Go templates (htmp/template).

It supports base template (optional) which is automatically added to each cached
template.

The package works by checking template file modification time on each get and
reparse the template if neccessary.
*/
package watcher

import (
	"fmt"
	"html/template"
	"log"
	"os"
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

// RegisterFiles adds the filenames to the cache under key for retrieval.
// The template created is equivalent to template.ParseFiles(filenames...).
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

// RegisterGlob adds the files matched by the Glob pattern to the cache
// under key for retrieval.
// The template created is equivalent to template.ParseGlob(pattern).
func RegisterGlob(key interface{}, pattern string) error {
	filenames, err := parseGlob(pattern)
	if err != nil {
		return err
	}
	return RegisterFiles(key, filenames...)
}

// Get returns the template registered under key. Returns error if nothing
// is found under key. Modifying the returned template will not change
// the cached template.
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

	return t.Clone()
}

type cacheKey int

const baseKey cacheKey = 0

// RegisterBaseFiles adds the filenames as a base template to be added to
// each cached template.
// The template created is equivalent to template.ParseFiles(filenames...).
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

// RegisterBaseGlob adds files matched by the Glob pattern as a base template
// to be added to each cached template.
// The template created is equivalent to template.ParseGlob(pattern).
func RegisterBaseGlob(pattern string) error {
	filenames, err := parseGlob(pattern)
	if err != nil {
		return err
	}
	return RegisterBaseFiles(filenames...)
}

func parseGlob(pattern string) ([]string, error) {
	filenames, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	if len(filenames) == 0 {
		return nil, fmt.Errorf("watcher: pattern matches no files: %#q", pattern)
	}
	return filenames, nil
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

func parseBaseFiles(filenames ...string) (*watched, error) {
	t, err := template.ParseFiles(filenames...)
	if err != nil {
		return nil, err
	}

	return &watched{
		filenames: filenames,
		template:  t,
		cached:    time.Now(),
	}, nil
}

type cacheGet struct {
	key interface{}
	c   chan *template.Template
}

type cacheSet struct {
	key interface{}
	w   *watched
}

var getChan = make(chan *cacheGet, 10)
var setChan = make(chan *cacheSet, 10)

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
	baseChanged := getBaseChangeTime()
	if w.cached.Before(changed) || (key != baseKey && w.cached.Before(baseChanged)) {
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

func getChangeTime(filenames ...string) time.Time {
	var changed time.Time
	for _, f := range filenames {
		fi, err := os.Stat(f)
		if err != nil {
			log.Println(err)
			continue
		}
		if fi.ModTime().After(changed) {
			changed = fi.ModTime()
		}
	}
	return changed
}

func getBaseChangeTime() time.Time {
	var changed time.Time
	cacheLock.RLock()
	w, ok := cache[baseKey]
	cacheLock.RUnlock()
	if !ok {
		return changed
	}
	changed = getChangeTime(w.filenames...)
	if w.cached.After(changed) {
		// solve same time issue (time not accurate enough)
		return w.cached.Add(time.Nanosecond)
	}
	return changed
}

func init() {
	go watcher()
}
