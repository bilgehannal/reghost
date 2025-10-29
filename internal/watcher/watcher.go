package watcher

import (
	"fmt"
	"path/filepath"

	"github.com/bilgehannal/reghost/internal/config"
	"github.com/bilgehannal/reghost/internal/utils"
	"github.com/fsnotify/fsnotify"
)

// Watcher watches the configuration file for changes
type Watcher struct {
	configPath string
	logger     *utils.Logger
	watcher    *fsnotify.Watcher
	onChange   func(*config.Config) error
}

// NewWatcher creates a new file watcher
func NewWatcher(configPath string, logger *utils.Logger, onChange func(*config.Config) error) (*Watcher, error) {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create watcher: %w", err)
	}

	w := &Watcher{
		configPath: configPath,
		logger:     logger,
		watcher:    fw,
		onChange:   onChange,
	}

	// Watch the config file
	if err := w.watcher.Add(configPath); err != nil {
		return nil, fmt.Errorf("failed to watch config file: %w", err)
	}

	// Also watch the directory for atomic renames
	dir := filepath.Dir(configPath)
	if err := w.watcher.Add(dir); err != nil {
		w.logger.Warn("Failed to watch config directory: %v", err)
	}

	return w, nil
}

// Start starts watching for file changes
func (w *Watcher) Start() {
	go w.watch()
}

// watch monitors file system events
func (w *Watcher) watch() {
	w.logger.Info("Started watching config file: %s", w.configPath)

	for {
		select {
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}

			// Check if the event is for our config file
			if event.Name != w.configPath && filepath.Base(event.Name) != filepath.Base(w.configPath) {
				continue
			}

			// Handle write and create events (covers atomic renames too)
			if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
				w.logger.Info("Config file changed, reloading...")
				if err := w.reloadConfig(); err != nil {
					w.logger.Error("Failed to reload config: %v", err)
				}
			}

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			w.logger.Error("Watcher error: %v", err)
		}
	}
}

// reloadConfig reloads the configuration and triggers the onChange callback
func (w *Watcher) reloadConfig() error {
	cfg, err := config.Reload(w.configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if w.onChange != nil {
		if err := w.onChange(cfg); err != nil {
			return fmt.Errorf("onChange callback failed: %w", err)
		}
	}

	w.logger.Info("Config reloaded successfully")
	return nil
}

// Close stops the watcher
func (w *Watcher) Close() error {
	return w.watcher.Close()
}
