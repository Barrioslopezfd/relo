package main

import (
	"log"
	"os"
	"os/exec"
	"sync"
	"syscall"

	"github.com/fsnotify/fsnotify"
)

func main() {
	startCmd := func() *exec.Cmd {
		cmd := exec.Command("npm", "run", "start")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Start(); err != nil {
			log.Fatal("Error -> ", err)
		}
		return cmd
	}

	cmd := startCmd()
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	var mu sync.Mutex

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Has(fsnotify.Create) ||
					event.Has(fsnotify.Rename) ||
					event.Has(fsnotify.Remove) ||
					event.Has(fsnotify.Write) {
					mu.Lock()
					if cmd.Process != nil {
						_ = cmd.Process.Signal(syscall.SIGTERM)
						_ = cmd.Wait()
					}
					cmd = startCmd()
					mu.Unlock()
				}
			case _, ok := <-watcher.Errors:
				if !ok {
					return
				}
			}
		}
	}()

	wd, err := os.Getwd()
	if err != nil {
		log.Fatal("os.Getwd() -> ", err)
	}
	dirNames := getDirs(wd)
	for _, dir := range dirNames {
		err = watcher.Add(dir)
		if err != nil {
			log.Fatal(err)
		}
	}
	<-make(chan struct{})
}

func getDirs(wd string) []string {
	var dirNames []string
	content, err := os.ReadDir(wd)
	if err != nil {
		log.Fatal("os.ReadDir(wd) -> ", err)
	}
	for _, entry := range content {
		if entry.IsDir() {
			if isIgnored(entry.Name()) {
				continue
			}
			tmp := getDirs(wd + "/" + entry.Name())
			dirNames = append(dirNames, tmp...)
		}
	}
	dirNames = append(dirNames, wd)
	return dirNames
}

func isIgnored(name string) bool {
	IGNORED := [...]string{
		"node_modules",
		".git",
		"dst",
	}
	for _, a := range IGNORED {
		if a == name {
			return true
		}
	}
	return false
}
