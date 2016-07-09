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
	threshold = 500 * time.Millisecond
)

type watch struct {
	*fsnotify.Watcher
	cmd  *exec.Cmd
	args []string
}

var (
	restart = make(chan bool)
	done    = make(chan bool)
	sig     = make(chan os.Signal)
)

// SubDirectories walks through the path(recursively)
// and returns a list of folders
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

// NewWatcher adds all valid directories to watch list
// It assumes the current working directory to be the root of the project
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
		log.Println("error Watching: ", d, err)
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

// Run waits for watcher events sent by fsnotify
// and triggers the restart process if required
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
				log.Println("error watching ", err)
				break
			}
			if filepath.Ext(event.Name) == ".go" || filepath.Ext(event.Name) == ".tmpl" || f.IsDir() {
				Describe(event)
				log.Println("restarting...")
				w.KillProcess()
				go w.StartNewProcess()

				// wait till app restarts
				// before responding to new changes
				<-restart
			}

		case err := <-w.Errors:
			log.Println("error:", err)
		}
	}
}

// StartNewProcess starts the process as an independent(separate) process
// instead of being a child process of gomon
func (w *watch) StartNewProcess() {
	w.cmd = exec.Command("go", w.args...)
	w.cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	w.cmd.Stdout = os.Stdout
	w.cmd.Stderr = os.Stderr

	err := w.cmd.Start()
	if err != nil {
		log.Fatal("unable to start process", err)
	}
	restart <- true
}

// KillProcess kills the new process created by StartNewProcess
// It kills the whole process group to ensure
// all of its child processes are killed
func (w *watch) KillProcess() {
	pgid, err := syscall.Getpgid(w.cmd.Process.Pid)
	if err == nil {
		syscall.Kill(-pgid, syscall.SIGTERM)
	}
}

// HandleSignal waits for program termination signal
// to gracefully shutdown the process
func (w *watch) HandleSignal() {
	<-sig
	w.Stop()
}

// Stop terminates the watch process and kills the app(started by StartNewProcess())
// and then shuts down
func (w *watch) Stop() {
	w.KillProcess()
	w.Watcher.Close()
	log.Println("shutting down...")
	done <- true
}

func main() {
	args := os.Args[1:]
	signal.Notify(sig, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}

	log.Println("starting gomon")

	w := &watch{
		Watcher: watcher,
		args:    args,
	}

	w.NewWatcher()

	go w.StartNewProcess()
	// wait till Process has Started successfully
	<-restart

	go w.HandleSignal()
	defer func() {
		if r := recover(); r != nil {
			w.Stop()
		}
	}()
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
