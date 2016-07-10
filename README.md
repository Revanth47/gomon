# gomon
![Go Report Card](https://goreportcard.com/badge/github.com/Revanth47/gomon)
### A command line tool to rerun go programs on file change

## Installation
This assumes you have set the [path variables](https://golang.org/doc/install#install)
```bash
$ go get -u github.com/Revanth47/gomon
```

## Usage 
```bash
$ gomon run main.go [args]       // instead of 'go' use 'gomon' and run it as usual
```

### Dependencies
gopkg.in/fsnotify.v1