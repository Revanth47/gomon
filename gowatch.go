// Package gomon is a simple command line file watcher for Go.
// Usage: gomon run <file.go> [args]

package main

import (
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"gopkg.in/fsnotify.v1"
)

const (
	threshold = 100 * time.Millisecond
)

type watch struct {
	*fsnotify.Watcher
	cmd  *exec.Cmd
	args []string
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

func (w *watch) NewWatcher() {
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	list := SubDirectories(wd)
	for _, dir := range list {
		w.AddFolder(dir)
	}
}

func (w *watch) AddFolder(d string) {
	err := w.Add(d)
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
		desc = "create"
	} else if event.Op&fsnotify.Remove == fsnotify.Remove {
		desc = "delete"
	} else if event.Op&fsnotify.Write == fsnotify.Write {
		desc = "modify"
	} else if event.Op&fsnotify.Rename == fsnotify.Rename {
		desc = "rename"
	} else if event.Op&fsnotify.Chmod == fsnotify.Chmod {
		desc = "chmod"
	}
	log.Println(desc, base)
}

func (w *watch) Run() {
	t := time.Now()
	for {
		select {
		case event := <-w.Events:
			if time.Since(t) < threshold {
				break
			}

			t = time.Now()
			f, err := os.Stat(event.Name)

			if err != nil {
				log.Println("Error watching ", err)
				break
			}

			if filepath.Ext(event.Name) == ".go" || filepath.Ext(event.Name) == ".tmpl" || f.IsDir() {
				Describe(event)
				w.StartNewProcess()
			}

		case err := <-w.Errors:
			log.Println("error:", err)
		}
	}
}

func (w *watch) StartNewProcess() {
	if w.cmd != nil && !w.cmd.ProcessState.Exited() {
		w.KillProcess()
	}

	w.cmd = exec.Command("go", w.args...)

	w.cmd.Stdout = os.Stdout
	w.cmd.Stderr = os.Stderr

	err := w.cmd.Start()
	if err != nil {
		log.Fatal("Unable to start process", err)
	}

	err = w.cmd.Wait()
	if err != nil {
		log.Println("App Crashed", err)
	}
}

func (w *watch) KillProcess() {
	err := w.cmd.Process.Kill()
	if err != nil {
		log.Fatal("Unable to Stop Process", err)
	}
}

func main() {

	defer func() {
		if r := recover(); r != nil {
			log.Println("Exciting App", r)
			os.Exit(0)
		}
	}()

	args := os.Args[1:]
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP, syscall.SIGQUIT)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}

	w := &watch{
		Watcher: watcher,
		args:    args,
	}

	log.Println("Starting App")
	w.StartNewProcess()
	w.NewWatcher()
	defer watcher.Close()

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
