package main

import (
	"log"
	"time"

	"gopkg.in/fsnotify.v1"
)

const (
	threshold = 100*time.Millisecond
)

func main() {
	watcher,err := fsnotify.NewWatcher()
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
				if(time.Since(t)>=threshold) {
					t = time.Now()
					log.Println("event",event)
					if event.Op&fsnotify.Write == fsnotify.Write {
						log.Println("modified file:", event.Name)
					}
				}
			case err := <-watcher.Errors:
				log.Println("error:",err) 	
			}
		}
	}()

	err = watcher.Add(".")
	if err != nil {
		log.Fatal(err)
	}
	<-done
}