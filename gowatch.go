package main

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"gopkg.in/fsnotify.v1"
)

const (
	threshold = 100 * time.Millisecond
)

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

	done := make(chan bool)
	go func() {
		t := time.Now()
		for {
			select {
			case event := <-watcher.Events:
				if time.Since(t) >= threshold && filepath.Ext(event.Name) == ".go" || filepath.Ext(event.Name) == ".tmpl" {
					t = time.Now()
					log.Println("event", event)
					if event.Op&fsnotify.Write == fsnotify.Write {
						log.Println("modified file:", event.Name)
					}
				}
			case err := <-watcher.Errors:
				log.Println("error:", err)
			}
		}
	}()

	err = watcher.Add(".")

	if err != nil {
		log.Fatal(err)
	}
	<-done
}
