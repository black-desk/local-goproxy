package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/mod/module"
	"golang.org/x/mod/zip"
)

type proxy struct {
	dir    string
	gopath string
}

var pathMux sync.Map

func (p *proxy) handle(w http.ResponseWriter, r *http.Request) {

	path := r.URL.EscapedPath()
	path = strings.Split(path, "/@v/")[0]
	if path[0] == '/' {
		path = path[1:]
	}

	index := strings.Index(path, "mod/")
	if index == -1 {
		log.Fatal(fmt.Sprintf("fail to find \"mod/\" in url path (%s)", path))
		http.NotFound(w, r)
		return
	}

	path = path[strings.Index(path, "mod/")+4:]

	err := p.genGoProxyFiles(path)
	if err != nil {
		log.Print(fmt.Errorf("fail to generate go proxy files: %w", err))
		http.NotFound(w, r)
		return
	}

	http.Redirect(w, r, "/cache"+r.URL.Path[4:], http.StatusFound)

	return
}

func (p *proxy) genGoProxyFiles(path string) (err error) {
	lockI, ok := pathMux.LoadOrStore(path, &sync.Mutex{})
	lock := lockI.(*sync.Mutex)
	lock.Lock()
	defer func() {
		if err != nil {
			pathMux.Delete(path)
		}
		lock.Unlock()
	}()
	if ok {
		return
	}

	dirPath := filepath.Join(p.gopath, path)
	modPath := filepath.Join(dirPath, "go.mod")

	if _, err = os.Stat(modPath); errors.Is(err, os.ErrNotExist) {
		err = fmt.Errorf("Cannot found go.mod \"%s\"", modPath)
		return
	}

	// version

	version, err := GetVersion(modPath)
	if err != nil {
		err = fmt.Errorf("Fail to get version of go.mod \"%s\": %w", path, err)
		return
	}

	// zip

	f, err := os.CreateTemp(p.dir, "*mod.zip")
	if err != nil {
		err = fmt.Errorf("fail to create zip file: %w", err)
		return
	}

	w := bufio.NewWriter(f)

	err = zip.CreateFromDir(w, module.Version{Path: path, Version: version}, dirPath)
	if err != nil {
		err = fmt.Errorf("fail to create zip from dir: %w", err)
		return
	}

	err = w.Flush()
	if err != nil {
		err = fmt.Errorf("fail to flush: %w", err)
		return
	}

	err = f.Close()
	if err != nil {
		err = fmt.Errorf("fail to close zip: %w", err)
		return
	}

	err = os.MkdirAll(filepath.Join(p.dir, "cache", path, "@v"), 0744)
	if err != nil {
		err = fmt.Errorf("fail to mkdir: %w", err)
		return
	}

	err = os.Rename(f.Name(), filepath.Join(p.dir, "cache", path, "@v", version+".zip"))
	if err != nil {
		err = fmt.Errorf("fail to move zip: %w", err)
		return
	}

	// mod

	err = os.Symlink(modPath, filepath.Join(p.dir, "cache", path, "@v", version+".mod"))
	if err != nil {
		err = fmt.Errorf("fail to create symlink of version.mod: %w", err)
		return
	}

	// list

	err = os.WriteFile(filepath.Join(p.dir, "cache", path, "@v", "list"), []byte(version), 0644)
	if err != nil {
		err = fmt.Errorf("fail to create list file: %w", err)
		return
	}

	// info

	err = os.WriteFile(filepath.Join(p.dir, "cache", path, "@v", version+".info"), []byte("{Version:"+version+"}"), 0644)
	if err != nil {
		err = fmt.Errorf("fail to create info file: %w", err)
		return
	}
	return
}
