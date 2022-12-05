package configs

import (
	"gopkg.in/fsnotify.v1"
)

type Read[T any] interface {
	FilePath() string
	ReadConfig() (T, error)
}

type fileManager[T any] struct {
	file    Read[T]
	watcher *fsnotify.Watcher
}

func NewFileManger[T any](conf Read[T]) (*ConfigManager[T], error) {
	watcher, fileWatchErr := fsnotify.NewWatcher()
	if fileWatchErr != nil {
		return nil, fileWatchErr
	}

	if addErr := watcher.Add(conf.FilePath()); addErr != nil {
		return nil, addErr
	}
	manager := NewManager[T](&fileManager[T]{
		file:    conf,
		watcher: watcher,
	})
	return manager, nil
}

func (f *fileManager[T]) Reload(update chan<- T) {
	defer f.watcher.Close()

	for {
		select {
		case event, ok := <-f.watcher.Events:
			if !ok {
				return
			}
			if event.Op&fsnotify.Write == fsnotify.Write {
				data, readErr := f.file.ReadConfig()
				if readErr != nil {
					continue
				}
				update <- data
			}
		case _, ok := <-f.watcher.Errors:
			if !ok {
				return
			}
		}
	}
}
