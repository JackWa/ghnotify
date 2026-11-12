package tray

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/getlantern/systray"

	"github.com/boyvinall/ghnotify/internal/config"
	"github.com/boyvinall/ghnotify/internal/notify"
)

// appStatus owns the tray icon and the config status menu item. The icon
// shows, in order of priority: a config problem (amber), new PR activity
// (red), or idle.
type appStatus struct {
	mu       sync.Mutex
	activity bool
	cfgErr   error // last reload failed; the previous config is still in use
	restart  bool  // a setting changed that only applies after a restart
	item     *systray.MenuItem
}

func (s *appStatus) setActivity(active bool) {
	s.mu.Lock()
	s.activity = active
	s.mu.Unlock()
	s.render()
}

func (s *appStatus) set(cfgErr error, restart bool) {
	s.mu.Lock()
	s.cfgErr, s.restart = cfgErr, restart
	s.mu.Unlock()
	s.render()
}

// needsRestart reports whether clicking the status item should relaunch.
func (s *appStatus) needsRestart() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cfgErr == nil && s.restart
}

func (s *appStatus) render() {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch {
	case s.cfgErr != nil:
		systray.SetIcon(iconWarningBytes())
	case s.activity:
		systray.SetIcon(iconActiveBytes())
	default:
		systray.SetIcon(iconBytes())
	}

	if s.item == nil {
		return
	}
	switch {
	case s.cfgErr != nil:
		s.item.SetTitle("⚠ Config error: " + truncate(s.cfgErr.Error(), 60))
		s.item.SetTooltip(s.cfgErr.Error() + "\nClick to open config.toml")
		s.item.Show()
	case s.restart:
		s.item.SetTitle("↻ Restart to apply config")
		s.item.SetTooltip("max_prs_per_section only changes after a restart")
		s.item.Show()
	default:
		s.item.Hide()
	}
}

// configReloader applies config file changes to the live store.
type configReloader struct {
	store   *config.Store
	status  *appStatus
	builtAt int    // max_prs_per_section the menu was built with
	refresh func() // re-poll so new filters and intervals apply now
	recheck func() // re-render the lists and icon with the new notification prefs
}

// startupError reports a config file that could not be loaded at startup.
func (r *configReloader) startupError(err error) {
	r.status.set(err, false)
	go notify.App("ghnotify: config error", err.Error()+"\nUsing default settings until fixed.")
}

func (r *configReloader) handle(res config.ParseResult) {
	if res.Err != nil {
		slog.Warn("config reload failed, keeping previous config", "err", res.Err)
		r.status.set(res.Err, r.restartPending(r.store.Get()))
		go notify.App("ghnotify: config error", res.Err.Error()+"\nPrevious settings still in use.")
		return
	}

	changes := config.Diff(r.store.Get(), res.Config)
	r.store.Set(res.Config)
	restart := r.restartPending(res.Config)
	r.status.set(nil, restart)
	slog.Info("config reloaded", "changes", changes, "unknown", res.Unknown)

	r.recheck()
	if len(changes) > 0 {
		r.refresh()
	}
	go notify.App(reloadTitle(changes, restart), reloadBody(changes, res.Unknown, restart))
}

func (r *configReloader) restartPending(cfg *config.AppConfig) bool {
	return cfg.MaxPRsPerSection != r.builtAt
}

func reloadTitle(changes []string, restart bool) string {
	switch {
	case restart:
		return "ghnotify: restart to apply config"
	case len(changes) == 0:
		return "ghnotify: config reloaded, no changes"
	default:
		return "ghnotify: config reloaded"
	}
}

func reloadBody(changes, unknown []string, restart bool) string {
	var lines []string
	lines = append(lines, changes...)
	if len(changes) == 0 {
		lines = append(lines, "Settings are the same as before.")
	}
	if restart {
		lines = append(lines, "max_prs_per_section applies after a restart (see menu).")
	}
	if len(unknown) > 0 {
		lines = append(lines, "Ignored unknown keys: "+strings.Join(unknown, ", "))
	}
	return strings.Join(lines, "\n")
}

// relaunch starts a new copy of ghnotify once this one has exited, then quits.
func relaunch() {
	exe, err := os.Executable()
	if err != nil {
		slog.Warn("relaunch: cannot find executable", "err", err)
		return
	}
	// Inside a bundle, relaunch the .app so macOS treats it as the same app.
	target := exe
	if i := strings.Index(exe, ".app/Contents/MacOS/"); i >= 0 {
		target = exe[:i+len(".app")]
	}
	// Wait for this process to exit so it has released its tray icon first.
	wait := fmt.Sprintf("while kill -0 %d 2>/dev/null; do sleep 0.1; done", os.Getpid())
	script := fmt.Sprintf("%s; open -n %q", wait, target)
	if target == exe {
		script = fmt.Sprintf("%s; %q &", wait, exe)
	}
	cmd := exec.Command("/bin/sh", "-c", script)
	if err := cmd.Start(); err != nil {
		slog.Warn("relaunch failed", "err", err)
		return
	}
	systray.Quit()
}

func truncate(s string, n int) string {
	if len([]rune(s)) <= n {
		return s
	}
	return string([]rune(s)[:n-1]) + "…"
}
