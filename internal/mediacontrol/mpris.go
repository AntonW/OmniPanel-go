// Package mediacontrol provides media player management with platform-specific integrations.
//
// On Linux: MPRIS2 D-Bus integration for monitoring and controlling media players
// on Linux desktop environments (KDE Plasma, GNOME, etc.).
// It automatically discovers MPRIS-compliant players (Spotify, VLC, Firefox, etc.)
// via the session D-Bus and exposes their state (title, artist, cover art, progress,
// playback status) through the application's DataBus for frontend consumption.
//
// On Windows: System Media Transport Controls (SMTC) integration for Windows 10/11.
// SMTC automatically discovers media-enabled applications (Spotify, browsers, VLC, etc.)
// and exposes their state through the DataBus.
//
// When multiple players are running, only the selected player's state is published
// to the mpris_* DataBus keys. The frontend renders source selection tabs on the
// cover art overlay when more than one player is available, letting the user switch
// between players. The full list of available players is published to the
// mpris_available_players DataBus key as a JSON array.

//go:build !windows

package mediacontrol

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/godbus/dbus/v5"

	"omnipanel-go/internal/config"
	"omnipanel-go/internal/databus"
)

const (
	mprisPrefix      = "org.mpris.MediaPlayer2."
	mprisObjectPath  = "/org/mpris/MediaPlayer2"
	mprisPlayerIface = "org.mpris.MediaPlayer2.Player"
	mprisRootIface   = "org.mpris.MediaPlayer2"
)


// Watcher monitors MPRIS media players on the D-Bus session bus
// and publishes their state to the DataBus. When multiple players
// are connected, only the selected player's state is published to
// the mpris_* DataBus keys. The full player list is published to
// mpris_available_players as a JSON array.
type Watcher struct {
	mu             sync.RWMutex
	config         *config.MediaPlayerConfig
	databus        *databus.DataBus
	logger         *slog.Logger
	conn           *dbus.Conn
	players        map[string]*PlayerState
	selectedPlayer string // D-Bus name suffix of the active player
	stopCh         chan struct{}
	running        bool
}

// New creates a new MPRIS watcher. Returns nil if media integration is disabled in config.
func New(cfg *config.MediaPlayerConfig, db *databus.DataBus, logger *slog.Logger) *Watcher {
	if cfg == nil || !cfg.Enabled {
		return nil
	}

	if logger == nil {
		logger = slog.Default()
	}

	w := &Watcher{
		config:  cfg,
		databus: db,
		logger:  logger,
		players: make(map[string]*PlayerState),
		stopCh:  make(chan struct{}),
	}

	return w
}

// Start begins monitoring MPRIS players. It connects to the session bus,
// discovers existing players, and starts a polling loop for updates.
func (w *Watcher) Start() error {
	if w == nil {
		return nil
	}

	w.mu.Lock()

	if w.running {
		w.mu.Unlock()
		return nil
	}

	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		w.mu.Unlock()
		w.logger.Warn("MPRIS: failed to connect to session bus", "error", err)
		return fmt.Errorf("mpris: connect session bus: %w", err)
	}

	w.conn = conn
	w.running = true

	w.mu.Unlock()

	w.logger.Info("MPRIS: watcher started, polling interval", "interval_ms", w.config.PollInterval)

	w.discoverExistingPlayers()
	w.publishPlayersList()

	go w.runPollLoop()
	go w.watchPlayerListChanges()

	return nil
}

// Close stops the watcher and cleans up resources.
func (w *Watcher) Close() error {
	if w == nil {
		return nil
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.running {
		return nil
	}

	close(w.stopCh)
	w.running = false

	if w.conn != nil {
		return w.conn.Close()
	}
	return nil
}

// runPollLoop periodically queries all discovered players for state updates.
func (w *Watcher) runPollLoop() {
	interval := time.Duration(w.config.PollInterval) * time.Millisecond
	if interval < 500*time.Millisecond {
		interval = 1 * time.Second
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopCh:
			return
		case <-ticker.C:
			w.pollPlayers()
		}
	}
}

// discoverExistingPlayers scans the D-Bus for already-running MPRIS players
// when the watcher starts. This catches players that were launched before the server.
func (w *Watcher) discoverExistingPlayers() {
	w.logger.Info("MPRIS: scanning for existing players")

	var names []string
	err := w.conn.BusObject().Call("org.freedesktop.DBus.ListNames", 0).Store(&names)
	if err != nil {
		w.logger.Error("MPRIS: failed to list D-Bus names", "error", err)
		return
	}

	w.logger.Info("MPRIS: found D-Bus names", "count", len(names))

	for _, name := range names {
		if strings.HasPrefix(name, mprisPrefix) {
			playerName := strings.TrimPrefix(name, mprisPrefix)
			w.logger.Info("MPRIS: discovering existing player", "player", playerName)
			w.discoverPlayer(playerName)
		}
	}
}

// watchPlayerListChanges listens for NameOwnerChanged signals to detect
// new players appearing or disappearing.
func (w *Watcher) watchPlayerListChanges() {
	ch := make(chan *dbus.Signal, 10)
	w.conn.Signal(ch)

	w.conn.BusObject().Call("org.freedesktop.DBus.AddMatch", 0,
		"type='signal',interface='org.freedesktop.DBus',member='NameOwnerChanged'")

	for {
		select {
		case <-w.stopCh:
			w.conn.RemoveSignal(ch)
			return
		case signal := <-ch:
			if len(signal.Body) < 3 {
				continue
			}

			name, ok := signal.Body[0].(string)
			if !ok {
				continue
			}

			newOwner, ok := signal.Body[2].(string)
			if !ok {
				continue
			}

			if strings.HasPrefix(name, mprisPrefix) {
				playerName := strings.TrimPrefix(name, mprisPrefix)
				if newOwner == "" {
					w.removePlayer(playerName)
				} else {
					w.discoverPlayer(playerName)
				}
			}
		}
	}
}

// pollPlayers queries all known players for state updates.
func (w *Watcher) pollPlayers() {
	w.mu.RLock()
	playerNames := make([]string, 0, len(w.players))
	for name := range w.players {
		playerNames = append(playerNames, name)
	}
	w.mu.RUnlock()

	for _, name := range playerNames {
		w.updatePlayer(name)
	}
}

// discoverPlayer attempts to connect to a newly appeared MPRIS player.
func (w *Watcher) discoverPlayer(playerName string) {
	w.logger.Info("MPRIS: player discovered", "player", playerName)

	state := &PlayerState{
		PlayerName: playerName,
	}

	w.mu.Lock()
	w.players[playerName] = state
	if w.selectedPlayer == "" {
		w.selectedPlayer = playerName
	}
	w.mu.Unlock()

	w.updatePlayer(playerName)
	w.publishPlayersList()
}

// removePlayer removes a player that has disconnected.
func (w *Watcher) removePlayer(playerName string) {
	w.mu.Lock()
	delete(w.players, playerName)
	if w.selectedPlayer == playerName {
		w.selectedPlayer = ""
		for name := range w.players {
			w.selectedPlayer = name
			break
		}
	}
	w.mu.Unlock()

	w.clearPlayerDataBus(playerName)
	w.publishPlayersList()
	if w.selectedPlayer != "" {
		w.updatePlayer(w.selectedPlayer)
	}
}

// updatePlayer queries the D-Bus properties for a single player.
func (w *Watcher) updatePlayer(playerName string) {
	defer func() {
		if r := recover(); r != nil {
			w.logger.Error("MPRIS: panic in updatePlayer", "player", playerName, "panic", r)
		}
	}()

	w.mu.RLock()
	state, exists := w.players[playerName]
	w.mu.RUnlock()

	if !exists {
		return
	}

	obj := w.conn.Object(mprisPrefix+playerName, mprisObjectPath)

	rootProps, err := w.getRootProperties(obj)
	if err != nil {
		w.logger.Debug("MPRIS: failed to get root properties", "player", playerName, "error", err)
		return
	}

	playerProps, err := w.getPlayerProperties(obj)
	if err != nil {
		w.logger.Debug("MPRIS: failed to get player properties", "player", playerName, "error", err)
		return
	}

	if v, ok := rootProps["Identity"].(string); ok {
		state.Identity = v
	}
	if v, ok := playerProps["PlaybackStatus"].(string); ok {
		state.PlaybackStatus = v
	}
	if v, ok := playerProps["Title"].(string); ok {
		state.Title = v
	}
	if v, ok := playerProps["Artist"].(string); ok {
		state.Artist = v
	}
	if v, ok := playerProps["Album"].(string); ok {
		state.Album = v
	}
	if v, ok := playerProps["ArtURL"].(string); ok {
		state.ArtURL = v
	}
	if v, ok := playerProps["Length"].(int64); ok {
		state.Length = v
	}
	if v, ok := playerProps["Position"].(int64); ok {
		state.Position = v
	}
	if v, ok := playerProps["Volume"].(float64); ok {
		state.Volume = v
	}
	if v, ok := playerProps["CanPlay"].(bool); ok {
		state.CanPlay = v
	}
	if v, ok := playerProps["CanPause"].(bool); ok {
		state.CanPause = v
	}
	if v, ok := playerProps["CanGoNext"].(bool); ok {
		state.CanGoNext = v
	}
	if v, ok := playerProps["CanGoPrevious"].(bool); ok {
		state.CanGoPrevious = v
	}
	if v, ok := playerProps["CanControl"].(bool); ok {
		state.CanControl = v
	}

	w.publishToDataBus(state)
}

// getRootProperties fetches properties from the root MPRIS interface.
func (w *Watcher) getRootProperties(obj dbus.BusObject) (map[string]any, error) {
	variant, err := obj.GetProperty(mprisRootIface + ".Identity")
	if err != nil {
		return nil, err
	}

	identity, _ := variant.Value().(string)
	return map[string]any{
		"Identity": identity,
	}, nil
}

// getPlayerProperties fetches properties from the Player interface.
func (w *Watcher) getPlayerProperties(obj dbus.BusObject) (map[string]any, error) {
	result := make(map[string]any)

	status, err := obj.GetProperty(mprisPlayerIface + ".PlaybackStatus")
	if err == nil {
		result["PlaybackStatus"], _ = status.Value().(string)
	}

	metadata, err := obj.GetProperty(mprisPlayerIface + ".Metadata")
	if err == nil {
		meta := parseMetadata(metadata)
		result["Title"] = meta["Title"]
		result["Artist"] = meta["Artist"]
		result["Album"] = meta["Album"]
		result["ArtURL"] = meta["ArtURL"]
		result["Length"] = meta["Length"]
	}

	position, err := obj.GetProperty(mprisPlayerIface + ".Position")
	if err == nil {
		pos, _ := position.Value().(int64)
		result["Position"] = pos
	}

	volume, err := obj.GetProperty(mprisPlayerIface + ".Volume")
	if err == nil {
		vol, _ := volume.Value().(float64)
		result["Volume"] = vol
	}

	canPlay, err := obj.GetProperty(mprisPlayerIface + ".CanPlay")
	if err == nil {
		result["CanPlay"], _ = canPlay.Value().(bool)
	}

	canPause, err := obj.GetProperty(mprisPlayerIface + ".CanPause")
	if err == nil {
		result["CanPause"], _ = canPause.Value().(bool)
	}

	canGoNext, err := obj.GetProperty(mprisPlayerIface + ".CanGoNext")
	if err == nil {
		result["CanGoNext"], _ = canGoNext.Value().(bool)
	}

	canGoPrev, err := obj.GetProperty(mprisPlayerIface + ".CanGoPrevious")
	if err == nil {
		result["CanGoPrevious"], _ = canGoPrev.Value().(bool)
	}

	canControl, err := obj.GetProperty(mprisPlayerIface + ".CanControl")
	if err == nil {
		result["CanControl"], _ = canControl.Value().(bool)
	}

	return result, nil
}

// parseMetadata extracts track metadata from the MPRIS Metadata property.
func parseMetadata(metadata dbus.Variant) map[string]any {
	result := map[string]any{
		"Title":  "",
		"Artist": "",
		"Album":  "",
		"ArtURL": "",
		"Length": int64(0),
	}

	metaMap, ok := metadata.Value().(map[string]dbus.Variant)
	if !ok {
		return result
	}

	if v, ok := metaMap["xesam:title"]; ok {
		result["Title"], _ = v.Value().(string)
	}

	if v, ok := metaMap["xesam:artist"]; ok {
		switch val := v.Value().(type) {
		case []string:
			if len(val) > 0 {
				result["Artist"] = strings.Join(val, ", ")
			}
		case string:
			result["Artist"] = val
		case []any:
			var artists []string
			for _, a := range val {
				if s, ok := a.(string); ok {
					artists = append(artists, s)
				}
			}
			if len(artists) > 0 {
				result["Artist"] = strings.Join(artists, ", ")
			}
		}
	}

	if v, ok := metaMap["xesam:album"]; ok {
		result["Album"], _ = v.Value().(string)
	}

	if v, ok := metaMap["mpris:artUrl"]; ok {
		result["ArtURL"], _ = v.Value().(string)
	}

	if v, ok := metaMap["mpris:length"]; ok {
		switch val := v.Value().(type) {
		case int64:
			result["Length"] = val
		case uint64:
			result["Length"] = int64(val)
		}
	}

	return result
}

// publishToDataBus sends player state to the DataBus for frontend consumption.
// Only publishes if the state belongs to the currently selected player.
func (w *Watcher) publishToDataBus(state *PlayerState) {
	w.mu.RLock()
	if w.selectedPlayer != state.PlayerName {
		w.mu.RUnlock()
		return
	}
	w.mu.RUnlock()

	keyPrefix := "mpris_"

	w.databus.SetSource(keyPrefix+"player_name", state.PlayerName, "", "mpris")
	w.databus.SetSource(keyPrefix+"identity", state.Identity, "", "mpris")
	w.databus.SetSource(keyPrefix+"playback_status", state.PlaybackStatus, "", "mpris")
	w.databus.SetSource(keyPrefix+"title", state.Title, "", "mpris")
	w.databus.SetSource(keyPrefix+"artist", state.Artist, "", "mpris")
	w.databus.SetSource(keyPrefix+"album", state.Album, "", "mpris")
	w.databus.SetSource(keyPrefix+"cover_url", state.ArtURL, "", "mpris")

	if state.Length > 0 {
		progress := float64(state.Position) / float64(state.Length) * 100.0
		w.databus.SetSource(keyPrefix+"progress", progress, "%", "mpris")
	} else {
		w.databus.SetSource(keyPrefix+"progress", 0.0, "%", "mpris")
	}

	volumePercent := state.Volume * 100.0
	w.databus.SetSource(keyPrefix+"volume", volumePercent, "%", "mpris")

	w.databus.SetSource(keyPrefix+"can_play", state.CanPlay, "", "mpris")
	w.databus.SetSource(keyPrefix+"can_pause", state.CanPause, "", "mpris")
	w.databus.SetSource(keyPrefix+"can_go_next", state.CanGoNext, "", "mpris")
	w.databus.SetSource(keyPrefix+"can_go_previous", state.CanGoPrevious, "", "mpris")
	w.databus.SetSource(keyPrefix+"can_control", state.CanControl, "", "mpris")
}

// publishPlayersList publishes the list of available players to the DataBus.
func (w *Watcher) publishPlayersList() {
	w.mu.RLock()
	type PlayerInfo struct {
		Name     string `json:"name"`
		Identity string `json:"identity"`
	}
	players := make([]PlayerInfo, 0, len(w.players))
	for name, state := range w.players {
		players = append(players, PlayerInfo{
			Name:     name,
			Identity: state.Identity,
		})
	}
	w.mu.RUnlock()

	data, _ := json.Marshal(players)
	w.databus.SetSource("mpris_available_players", string(data), "", "mpris")
}

// SetSelectedPlayer sets the active player whose state is published to the
// DataBus. Only the selected player's metadata appears in the mpris_* keys
// that the frontend reads. Returns an error if the player is not connected.
// When the selection changes, the new player's state is immediately fetched
// and published.
func (w *Watcher) SetSelectedPlayer(playerName string) error {
	w.mu.Lock()
	if _, exists := w.players[playerName]; !exists {
		w.mu.Unlock()
		return fmt.Errorf("mpris: player %q not found", playerName)
	}

	oldSelected := w.selectedPlayer
	w.selectedPlayer = playerName
	w.mu.Unlock()

	w.logger.Info("MPRIS: selected player changed", "from", oldSelected, "to", playerName)

	if w.selectedPlayer != oldSelected {
		w.updatePlayer(w.selectedPlayer)
	}
	return nil
}

// GetSelectedPlayer returns the D-Bus name suffix of the currently selected
// player. Returns an empty string if no player is selected.
func (w *Watcher) GetSelectedPlayer() string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.selectedPlayer
}

// clearPlayerDataBus removes all MPRIS keys from the DataBus when a player disconnects.
func (w *Watcher) clearPlayerDataBus(playerName string) {
	keys := []string{
		"mpris_player_name", "mpris_identity", "mpris_playback_status",
		"mpris_title", "mpris_artist", "mpris_album", "mpris_cover_url",
		"mpris_progress", "mpris_volume",
		"mpris_can_play", "mpris_can_pause", "mpris_can_go_next",
		"mpris_can_go_previous", "mpris_can_control",
	}

	for _, key := range keys {
		w.databus.SetSource(key, "", "", "mpris")
	}
}

// CallMethod sends an MPRIS method call to the specified player.
func (w *Watcher) CallMethod(playerName, method string) error {
	if w == nil || !w.running {
		return fmt.Errorf("mpris: watcher not running")
	}

	obj := w.conn.Object(mprisPrefix+playerName, mprisObjectPath)
	call := obj.Call(mprisPlayerIface+"."+method, 0)
	return call.Store()
}

// SetVolume sets the volume for the specified player (0.0 to 1.0).
func (w *Watcher) SetVolume(playerName string, volume float64) error {
	if w == nil || !w.running {
		return fmt.Errorf("mpris: watcher not running")
	}

	obj := w.conn.Object(mprisPrefix+playerName, mprisObjectPath)

	propsIface := "org.freedesktop.DBus.Properties"
	call := obj.Call(propsIface+".Set", 0, mprisPlayerIface, "Volume", dbus.MakeVariant(volume))
	if err := call.Store(); err != nil {
		w.logger.Error("mpris: set volume failed", "player", playerName, "volume", volume, "error", err)
		return err
	}
	w.logger.Debug("mpris: volume set", "player", playerName, "volume", volume)
	return nil
}

// ListPlayers returns the names of all currently connected players.
func (w *Watcher) ListPlayers() []string {
	if w == nil {
		return nil
	}

	w.mu.RLock()
	defer w.mu.RUnlock()

	players := make([]string, 0, len(w.players))
	for name := range w.players {
		players = append(players, name)
	}
	return players
}

// GetPlayerState returns the current state of a specific player.
func (w *Watcher) GetPlayerState(playerName string) *PlayerState {
	if w == nil {
		return nil
	}

	w.mu.RLock()
	defer w.mu.RUnlock()

	state, exists := w.players[playerName]
	if !exists {
		return nil
	}

	return state
}
