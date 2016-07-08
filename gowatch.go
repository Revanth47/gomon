// gomon is a simple file watcher for golang
// Usage: gomon run <file.go>

package main

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/fsnotify.v1"
)

const (
	threshold = 100 * time.Millisecond
)

type watch struct {
	*fsnotify.Watcher
}

// SubDirectories walks through the path(passed as arg)
// and returns a list of folders recursively
func SubDirectories(p string) []string {
	filelist := []string{}
	err := filepath.Walk(p, func(file string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			dir := info.Name()
			if ShouldIgnore(dir) && dir != "." && dir != ".." {
				return filepath.SkipDir
			}
			filelist = append(filelist, file)
		}

		return nil
	})

	if err != nil {
		log.Fatal(err)
	}
	return filelist
}

func (watcher *watch) NewWatcher() {
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	list := SubDirectories(wd)
	for _, dir := range list {
		watcher.AddFolder(dir)
	}
}

func (watcher *watch) AddFolder(d string) {
	err := watcher.Add(d)
	if err != nil {
		log.Println("Error Watching: ", d, err)
	}
}

// Describe checks for the type of operation occured and
// logs the event info
func Describe(event fsnotify.Event) {
	desc := ""
	base := filepath.Base(event.Name)
	if event.Op&fsnotify.Create == fsnotify.Create {
		desc = "create file "
	} else if event.Op&fsnotify.Remove == fsnotify.Remove {
		desc = "delete file "
	} else if event.Op&fsnotify.Write == fsnotify.Write {
		desc = "modify file "
	} else if event.Op&fsnotify.Rename == fsnotify.Rename {
		desc = "rename file "
	} else if event.Op&fsnotify.Chmod == fsnotify.Chmod {
		desc = "chmod file "
	}
	log.Println(desc + base)
}

func (watcher *watch) Run() {
	t := time.Now()
	for {
		select {
		case event := <-watcher.Events:
			if time.Since(t) >= threshold {
				t = time.Now()
				Describe(event)
			}
		case err := <-watcher.Errors:
			log.Println("error:", err)
		}
	}
}

func main() {
	args := os.Args[1:]
	cmd := exec.Command("go", args...)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Start()
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Starting App")

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	w := &watch{Watcher: watcher}
	w.NewWatcher()

	done := make(chan bool)
	go w.Run()
	<-done
}

// ShouldIgnore checks if the directory can be ignored
// so that no of watched files doesn't go beyond the ulimit
func ShouldIgnore(d string) bool {
	if d == "node_modules" || d == "vendor" || strings.HasPrefix(d, ".") {
		return true
	}
	return false
}
