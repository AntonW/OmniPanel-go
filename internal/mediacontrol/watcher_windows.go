// Windows media player integration via SMTC (System Media Transport Controls).
//
// On Windows 10/11, SMTC is the standard mechanism for media player metadata and control.
// It surfaces information from any SMTC-registered player (Spotify, Windows Media Player,
// browsers playing audio via MediaSession API, VLC, etc.) and allows controlling playback.
//
// This file replaces the Linux MPRIS/D-Bus watcher on Windows. It provides the same
// Watcher type and method set so all callers (state.go, routes/mpris.go) work unmodified.
//
// Cover art is fetched from the SMTC thumbnail stream and saved as a JPEG in a temp
// directory so the existing /api/media/cover HTTP endpoint can serve it.

//go:build windows

package mediacontrol

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/go-ole/go-ole"
	"github.com/saltosystems/winrt-go/windows/foundation"
	"github.com/saltosystems/winrt-go/windows/foundation/collections"
	"github.com/saltosystems/winrt-go/windows/media/control"
	"github.com/saltosystems/winrt-go/windows/storage/streams"

	"omnipanel-go/internal/config"
	"omnipanel-go/internal/databus"
)

// -----------------------------------------------------------------------------
// Minimal WinRT COM interfaces needed for thumbnail reading
// -----------------------------------------------------------------------------

// IRandomAccessStream – GUID 905A0FE1-BC53-11DF-8C49-001E4FC686DA
// Only get_Size is needed here; all other slots are present to keep the vtable correct.
type iRandomAccessStream struct{ ole.IInspectable }

type iRandomAccessStreamVtbl struct {
	ole.IInspectableVtbl
	GetSize     uintptr
	SetSize     uintptr
	GetPosition uintptr
	Seek        uintptr
	CloneStream uintptr
	GetCanRead  uintptr
	GetCanWrite uintptr
}

func (v *iRandomAccessStream) VTable() *iRandomAccessStreamVtbl {
	return (*iRandomAccessStreamVtbl)(unsafe.Pointer(v.RawVTable))
}

func (v *iRandomAccessStream) getSize() (uint64, error) {
	var size uint64
	hr, _, _ := syscall.SyscallN(
		v.VTable().GetSize,
		uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(&size)),
	)
	if hr != 0 {
		return 0, ole.NewError(hr)
	}
	return size, nil
}

// IDataReaderFactory – GUID A5BB7005-3333-4F91-A51B-E2B2BDE24CD6
// CreateDataReader(IInputStream*) -> DataReader*
type iDataReaderFactory struct{ ole.IInspectable }

type iDataReaderFactoryVtbl struct {
	ole.IInspectableVtbl
	CreateDataReader uintptr
}

func (v *iDataReaderFactory) VTable() *iDataReaderFactoryVtbl {
	return (*iDataReaderFactoryVtbl)(unsafe.Pointer(v.RawVTable))
}

const (
	guidIRandomAccessStream = "905a0fe1-bc53-11df-8c49-001e4fc686da"
	guidIInputStream        = "905a0fe0-bc53-11df-8c49-001e4fc686da"
	guidIDataReaderFactory  = "a5bb7005-3333-4f91-a51b-e2b2bde24cd6"
)

// -----------------------------------------------------------------------------
// Watcher
// -----------------------------------------------------------------------------

// Watcher monitors Windows SMTC sessions and publishes their state to the DataBus.
// It implements the same interface as the Linux MPRIS Watcher so all callers are
// platform-independent.
type Watcher struct {
	mu             sync.RWMutex
	config         *config.MediaPlayerConfig
	databus        *databus.DataBus
	logger         *slog.Logger
	players        map[string]*PlayerState
	selectedPlayer string
	stopCh         chan struct{}
	running        bool
	tempDir        string
}

// New creates a new SMTC watcher. Returns nil if media integration is disabled.
func New(cfg *config.MediaPlayerConfig, db *databus.DataBus, logger *slog.Logger) *Watcher {
	if cfg == nil || !cfg.Enabled {
		return nil
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Watcher{
		config:  cfg,
		databus: db,
		logger:  logger,
		players: make(map[string]*PlayerState),
		stopCh:  make(chan struct{}),
		tempDir: filepath.Join(os.TempDir(), "omnipanel-smtc"),
	}
}

// Start begins SMTC polling.
func (w *Watcher) Start() error {
	if w == nil {
		return nil
	}

	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return nil
	}
	w.running = true
	w.mu.Unlock()

	// WinRT requires COM/WinRT initialization.
	// S_FALSE (0x00000001) means already initialised – that is fine.
	if err := ole.RoInitialize(1 /* RO_INIT_MULTITHREADED */); err != nil {
		if oleErr, ok := err.(*ole.OleError); !ok || uintptr(oleErr.Code()) != 0x00000001 {
			w.logger.Warn("SMTC: RoInitialize warning", "error", err)
		}
	}

	if err := os.MkdirAll(w.tempDir, 0o755); err != nil {
		w.logger.Warn("SMTC: cannot create temp dir for cover art", "path", w.tempDir, "error", err)
	}

	w.logger.Info("SMTC: watcher started", "interval_ms", w.config.PollInterval)

	go w.runPollLoop()
	return nil
}

// Close stops the watcher and removes temporary cover art files.
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
	_ = os.RemoveAll(w.tempDir)
	return nil
}

func (w *Watcher) runPollLoop() {
	interval := time.Duration(w.config.PollInterval) * time.Millisecond
	if interval < 500*time.Millisecond {
		interval = 1 * time.Second
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	w.pollSessions()

	for {
		select {
		case <-w.stopCh:
			return
		case <-ticker.C:
			w.pollSessions()
		}
	}
}

// pollSessions queries the SMTC session manager for all active media sessions,
// builds PlayerState records, and publishes them to the DataBus.
func (w *Watcher) pollSessions() {
	defer func() {
		if r := recover(); r != nil {
			w.logger.Error("SMTC: panic in pollSessions", "panic", r)
		}
	}()

	asyncOp, err := control.GlobalSystemMediaTransportControlsSessionManagerRequestAsync()
	if err != nil {
		w.logger.Debug("SMTC: RequestAsync failed", "error", err)
		return
	}

	rawManager, err := awaitAsync(asyncOp)
	asyncOp.Release()
	if err != nil || rawManager == nil {
		w.logger.Debug("SMTC: await manager failed", "error", err)
		return
	}
	manager := (*control.GlobalSystemMediaTransportControlsSessionManager)(rawManager)
	defer manager.Release()

	sessionsVec, err := manager.GetSessions()
	if err != nil {
		w.logger.Debug("SMTC: GetSessions failed", "error", err)
		return
	}
	defer sessionsVec.Release()

	count, err := sessionsVec.GetSize()
	if err != nil {
		return
	}

	if count == 0 {
		w.mu.Lock()
		w.players = make(map[string]*PlayerState)
		w.selectedPlayer = ""
		w.mu.Unlock()
		w.publishPlayersList()
		w.clearDataBus()
		return
	}

	newPlayers := make(map[string]*PlayerState, count)

	for i := uint32(0); i < count; i++ {
		rawSession, err := sessionsVec.GetAt(i)
		if err != nil || rawSession == nil {
			continue
		}
		session := (*control.GlobalSystemMediaTransportControlsSession)(rawSession)

		appID, err := session.GetSourceAppUserModelId()
		if err != nil || appID == "" {
			session.Release()
			continue
		}

		state := w.buildPlayerState(appID, session)
		session.Release()
		if state != nil {
			newPlayers[appID] = state
		}
	}

	w.mu.Lock()
	w.players = newPlayers
	if w.selectedPlayer == "" {
		for name := range newPlayers {
			w.selectedPlayer = name
			break
		}
	} else if _, exists := newPlayers[w.selectedPlayer]; !exists {
		w.selectedPlayer = ""
		for name := range newPlayers {
			w.selectedPlayer = name
			break
		}
	}
	selected := w.selectedPlayer
	w.mu.Unlock()

	w.publishPlayersList()
	if selected != "" {
		if state, ok := newPlayers[selected]; ok {
			w.publishToDataBus(state)
		}
	}
}

// buildPlayerState queries metadata, playback info, and timeline for one SMTC session.
func (w *Watcher) buildPlayerState(appID string, session *control.GlobalSystemMediaTransportControlsSession) *PlayerState {
	state := &PlayerState{
		PlayerName: appID,
		Identity:   appIDToName(appID),
	}

	// --- Media properties (async) ---
	propsOp, err := session.TryGetMediaPropertiesAsync()
	if err == nil && propsOp != nil {
		rawProps, err := awaitAsync(propsOp)
		propsOp.Release()
		if err == nil && rawProps != nil {
			props := (*control.GlobalSystemMediaTransportControlsSessionMediaProperties)(rawProps)

			if v, err := props.GetTitle(); err == nil {
				state.Title = v
			}
			if v, err := props.GetArtist(); err == nil {
				state.Artist = v
			}
			if v, err := props.GetAlbumTitle(); err == nil {
				state.Album = v
			}
		if thumbRef, err := props.GetThumbnail(); err == nil && thumbRef != nil {
			state.ArtURL = w.extractThumbnail(thumbRef, appID, state)
			thumbRef.Release()
		} else {
			// Chrome/YouTube doesn't provide thumbnails through SMTC for web content
			// The frontend will show the placeholder icon instead
			if strings.Contains(appID, "chrome") || strings.Contains(appID, "msedge") {
				slog.Debug("smtc: Chrome/Edge web content does not provide cover art through SMTC", "appID", appID, "title", state.Title)
			}
		}
			props.Release()
		}
	}

	// --- Playback info (synchronous) ---
	playbackInfo, err := session.GetPlaybackInfo()
	if err == nil && playbackInfo != nil {
		if status, err := playbackInfo.GetPlaybackStatus(); err == nil {
			switch status {
			case control.GlobalSystemMediaTransportControlsSessionPlaybackStatusPlaying:
				state.PlaybackStatus = "Playing"
			case control.GlobalSystemMediaTransportControlsSessionPlaybackStatusPaused:
				state.PlaybackStatus = "Paused"
			default:
				state.PlaybackStatus = "Stopped"
			}
		}
		if ctrls, err := playbackInfo.GetControls(); err == nil && ctrls != nil {
			if v, e := ctrls.GetIsPlayEnabled(); e == nil {
				state.CanPlay = v
			}
			if v, e := ctrls.GetIsPauseEnabled(); e == nil {
				state.CanPause = v
			}
			if v, e := ctrls.GetIsNextEnabled(); e == nil {
				state.CanGoNext = v
			}
			if v, e := ctrls.GetIsPreviousEnabled(); e == nil {
				state.CanGoPrevious = v
			}
			state.CanControl = true
			ctrls.Release()
		}
		playbackInfo.Release()
	}

	// --- Timeline (synchronous) ---
	timeline, err := session.GetTimelineProperties()
	if err == nil && timeline != nil {
		if pos, err := timeline.GetPosition(); err == nil {
			// TimeSpan.Duration is in 100-nanosecond intervals → convert to microseconds
			state.Position = pos.Duration / 10
		}
		if end, err := timeline.GetEndTime(); err == nil {
			state.Length = end.Duration / 10
		}
		timeline.Release()
	}

	return state
}

// extractThumbnail reads the cover art from the SMTC thumbnail stream and saves
// it to a temp file. Returns a "file://" URL or empty string on failure.
// For YouTube/Chrome, attempts to extract video ID and fetch thumbnail from YouTube CDN.
func (w *Watcher) extractThumbnail(thumbRef *streams.IRandomAccessStreamReference, appID string, state *PlayerState) string {
	// Try SMTC thumbnail first
	if thumbRef != nil {
		if url := w.extractSMTCThumbnail(thumbRef, appID); url != "" {
			return url
		}
	}

	// Fallback for Chrome/YouTube when SMTC doesn't provide thumbnail
	if state != nil && (strings.Contains(appID, "chrome") || strings.Contains(appID, "msedge")) {
		if url := extractYouTubeThumbnail(state); url != "" {
			return url
		}
	}

	return ""
}

// extractSMTCThumbnail reads the cover art from the SMTC thumbnail stream.
func (w *Watcher) extractSMTCThumbnail(thumbRef *streams.IRandomAccessStreamReference, appID string) string {
	if thumbRef == nil {
		return ""
	}

	openOp, err := thumbRef.OpenReadAsync()
	if err != nil || openOp == nil {
		slog.Debug("smtc: OpenReadAsync failed", "appID", appID, "error", err)
		return ""
	}

	rawStream, err := awaitAsync(openOp)
	openOp.Release()
	if err != nil || rawStream == nil {
		slog.Debug("smtc: awaitAsync failed for thumbnail stream", "appID", appID, "error", err)
		return ""
	}

	streamUnk := (*ole.IUnknown)(rawStream)

	// QI to IRandomAccessStream to get the byte count
	rasItf, err := streamUnk.QueryInterface(ole.NewGUID(guidIRandomAccessStream))
	if err != nil {
		streamUnk.Release()
		slog.Debug("smtc: QI to IRandomAccessStream failed", "appID", appID, "error", err)
		return ""
	}
	ras := (*iRandomAccessStream)(unsafe.Pointer(rasItf))
	size, err := ras.getSize()
	rasItf.Release()
	if err != nil || size == 0 || size > 4*1024*1024 {
		streamUnk.Release()
		slog.Debug("smtc: thumbnail size invalid", "appID", appID, "size", size, "error", err)
		return ""
	}

	// QI to IInputStream so a DataReader can consume it
	isItf, err := streamUnk.QueryInterface(ole.NewGUID(guidIInputStream))
	streamUnk.Release()
	if err != nil {
		// Some SMTC sources (like YouTube on Chrome) may not support IInputStream
		// This is expected behavior and not an error
		slog.Debug("smtc: thumbnail stream does not support IInputStream interface", "appID", appID, "error", err)
		return ""
	}

	dr, err := newDataReaderFromInputStream(unsafe.Pointer(isItf))
	isItf.Release()
	if err != nil || dr == nil {
		slog.Debug("smtc: newDataReaderFromInputStream failed", "appID", appID, "error", err)
		return ""
	}
	defer dr.Release()

	// LoadAsync(count) – vtable slot 29 of IDataReader
	var loadOp *foundation.IAsyncOperation
	hr, _, _ := syscall.SyscallN(
		dr.VTable().LoadAsync,
		uintptr(unsafe.Pointer(dr)),
		uintptr(uint32(size)),
		uintptr(unsafe.Pointer(&loadOp)),
	)
	if hr != 0 || loadOp == nil {
		slog.Debug("smtc: LoadAsync failed", "appID", appID, "hr", hr)
		return ""
	}

	if _, err = awaitAsync(loadOp); err != nil {
		loadOp.Release()
		slog.Debug("smtc: awaitAsync for LoadAsync failed", "appID", appID, "error", err)
		return ""
	}
	loadOp.Release()

	data, err := dr.ReadBytes(uint32(size))
	if err != nil || len(data) == 0 {
		slog.Debug("smtc: ReadBytes failed or returned empty", "appID", appID, "error", err)
		return ""
	}

	safeID := strings.NewReplacer("!", "_", "\\", "_", "/", "_", ":", "_").Replace(appID)
	tmpFile := filepath.Join(w.tempDir, safeID+".jpg")
	if err := os.WriteFile(tmpFile, data, 0o644); err != nil {
		slog.Debug("smtc: WriteFile failed", "appID", appID, "path", tmpFile, "error", err)
		return ""
	}
	return "file://" + filepath.ToSlash(tmpFile)
}

// extractYouTubeThumbnail attempts to fetch a thumbnail for YouTube videos.
// Currently returns empty string because:
// 1. Chrome/Edge via SMTC doesn't provide thumbnail streams for YouTube
// 2. The video ID cannot be extracted from MediaSession metadata
// 3. YouTube's public API would be required to fetch thumbnails from video IDs
// 
// When a thumbnail is unavailable, the media player block shows a
// placeholder icon, which is the expected behavior on web platforms.
func extractYouTubeThumbnail(state *PlayerState) string {
	if state == nil || state.Title == "" {
		return ""
	}

	// This could be enhanced in the future by:
	// - Using MediaSession ExtendedMediaSessionAPI if available
	// - Proxying YouTube's public thumbnail endpoint
	// - Implementing native YouTube API integration
	return ""
}

// newDataReaderFromInputStream creates a WinRT DataReader from an IInputStream pointer.
func newDataReaderFromInputStream(inputStreamPtr unsafe.Pointer) (*streams.IDataReader, error) {
	factory, err := ole.RoGetActivationFactory(
		"Windows.Storage.Streams.DataReader",
		ole.NewGUID(guidIDataReaderFactory),
	)
	if err != nil {
		return nil, fmt.Errorf("smtc: DataReaderFactory: %w", err)
	}
	defer factory.Release()

	f := (*iDataReaderFactory)(unsafe.Pointer(factory))
	var out *streams.IDataReader
	hr, _, _ := syscall.SyscallN(
		f.VTable().CreateDataReader,
		uintptr(unsafe.Pointer(f)),
		uintptr(inputStreamPtr),
		uintptr(unsafe.Pointer(&out)),
	)
	if hr != 0 {
		return nil, ole.NewError(hr)
	}
	return out, nil
}

// awaitAsync polls an IAsyncOperation until it completes (or times out).
// Returns the raw result pointer (to be cast to the concrete type by the caller).
func awaitAsync(op *foundation.IAsyncOperation) (unsafe.Pointer, error) {
	if op == nil {
		return nil, errors.New("smtc: nil IAsyncOperation")
	}

	asyncInfoItf, err := op.QueryInterface(ole.NewGUID(foundation.GUIDIAsyncInfo))
	if err != nil {
		return nil, fmt.Errorf("smtc: QI IAsyncInfo: %w", err)
	}
	defer asyncInfoItf.Release()
	info := (*foundation.IAsyncInfo)(unsafe.Pointer(asyncInfoItf))

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		status, err := info.GetStatus()
		if err != nil {
			return nil, fmt.Errorf("smtc: GetStatus: %w", err)
		}
		switch status {
		case foundation.AsyncStatusCompleted:
			return op.GetResults()
		case foundation.AsyncStatusError:
			return nil, errors.New("smtc: async operation failed")
		case foundation.AsyncStatusCanceled:
			return nil, errors.New("smtc: async operation canceled")
		}
		time.Sleep(5 * time.Millisecond)
	}
	return nil, errors.New("smtc: async operation timed out")
}

// -----------------------------------------------------------------------------
// Public API – mirrors the Linux MPRIS Watcher interface
// -----------------------------------------------------------------------------

// ListPlayers returns the SourceAppUserModelId of all currently active SMTC sessions.
func (w *Watcher) ListPlayers() []string {
	if w == nil {
		return nil
	}
	w.mu.RLock()
	defer w.mu.RUnlock()
	names := make([]string, 0, len(w.players))
	for name := range w.players {
		names = append(names, name)
	}
	return names
}

// GetPlayerState returns the last polled state of a player by its appID.
func (w *Watcher) GetPlayerState(playerName string) *PlayerState {
	if w == nil {
		return nil
	}
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.players[playerName]
}

// GetSelectedPlayer returns the currently active player's appID.
func (w *Watcher) GetSelectedPlayer() string {
	if w == nil {
		return ""
	}
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.selectedPlayer
}

// SetSelectedPlayer sets the active player. Returns an error if the player is unknown.
func (w *Watcher) SetSelectedPlayer(playerName string) error {
	if w == nil {
		return errors.New("smtc: watcher is nil")
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if _, exists := w.players[playerName]; !exists {
		return fmt.Errorf("smtc: player %q not found", playerName)
	}
	w.selectedPlayer = playerName
	return nil
}

// CallMethod sends a playback control command to the specified SMTC session.
// Supported methods: Play, Pause, PlayPause, Stop, Next, Previous.
func (w *Watcher) CallMethod(playerName, method string) error {
	if w == nil {
		return errors.New("smtc: watcher is nil")
	}

	// Re-acquire the live session for the player
	session, err := w.getSession(playerName)
	if err != nil {
		return err
	}
	defer session.Release()

	var asyncOp *foundation.IAsyncOperation

	switch method {
	case "Play":
		asyncOp, err = session.TryPlayAsync()
	case "Pause":
		asyncOp, err = session.TryPauseAsync()
	case "PlayPause":
		asyncOp, err = session.TryTogglePlayPauseAsync()
	case "Stop":
		asyncOp, err = session.TryStopAsync()
	case "Next":
		asyncOp, err = session.TrySkipNextAsync()
	case "Previous":
		asyncOp, err = session.TrySkipPreviousAsync()
	default:
		return fmt.Errorf("smtc: unknown method: %s", method)
	}

	if err != nil {
		return fmt.Errorf("smtc: %s: %w", method, err)
	}
	if asyncOp != nil {
		_, _ = awaitAsync(asyncOp)
		asyncOp.Release()
	}
	return nil
}

// SetVolume is a no-op on Windows: per-player volume is not exposed by SMTC.
// System volume is controlled directly by the /api/media/control HTTP endpoint
// (routes/mediacontrol_volume_windows.go) via WASAPI without going through the Watcher.
func (w *Watcher) SetVolume(_ string, _ float64) error {
	return nil
}

// getSession resolves a live SMTC session for the given appID by polling SMTC.
func (w *Watcher) getSession(playerName string) (*control.GlobalSystemMediaTransportControlsSession, error) {
	asyncOp, err := control.GlobalSystemMediaTransportControlsSessionManagerRequestAsync()
	if err != nil {
		return nil, err
	}
	rawManager, err := awaitAsync(asyncOp)
	asyncOp.Release()
	if err != nil || rawManager == nil {
		return nil, fmt.Errorf("smtc: cannot get session manager: %w", err)
	}
	manager := (*control.GlobalSystemMediaTransportControlsSessionManager)(rawManager)
	defer manager.Release()

	sessionsVec, err := manager.GetSessions()
	if err != nil {
		return nil, err
	}
	defer sessionsVec.Release()

	count, _ := sessionsVec.GetSize()
	for i := uint32(0); i < count; i++ {
		rawSession, err := sessionsVec.GetAt(i)
		if err != nil || rawSession == nil {
			continue
		}
		session := (*control.GlobalSystemMediaTransportControlsSession)(rawSession)
		appID, err := session.GetSourceAppUserModelId()
		if err != nil || appID != playerName {
			session.Release()
			continue
		}
		return session, nil
	}
	return nil, fmt.Errorf("smtc: session %q not found", playerName)
}

// -----------------------------------------------------------------------------
// DataBus publishing
// -----------------------------------------------------------------------------

func (w *Watcher) publishToDataBus(state *PlayerState) {
	w.mu.RLock()
	if w.selectedPlayer != state.PlayerName {
		w.mu.RUnlock()
		return
	}
	w.mu.RUnlock()

	const pfx = "mediacontrol_"
	w.databus.SetSource(pfx+"player_name", state.PlayerName, "", "mediacontrol")
	w.databus.SetSource(pfx+"identity", state.Identity, "", "mediacontrol")
	w.databus.SetSource(pfx+"playback_status", state.PlaybackStatus, "", "mediacontrol")
	w.databus.SetSource(pfx+"title", state.Title, "", "mediacontrol")
	w.databus.SetSource(pfx+"artist", state.Artist, "", "mediacontrol")
	w.databus.SetSource(pfx+"album", state.Album, "", "mediacontrol")
	w.databus.SetSource(pfx+"cover_url", state.ArtURL, "", "mediacontrol")

	progress := 0.0
	if state.Length > 0 {
		progress = float64(state.Position) / float64(state.Length) * 100.0
	}
	w.databus.SetSource(pfx+"progress", progress, "%", "mediacontrol")
	w.databus.SetSource(pfx+"volume", state.Volume*100.0, "%", "mediacontrol")
	w.databus.SetSource(pfx+"can_play", state.CanPlay, "", "mediacontrol")
	w.databus.SetSource(pfx+"can_pause", state.CanPause, "", "mediacontrol")
	w.databus.SetSource(pfx+"can_go_next", state.CanGoNext, "", "mediacontrol")
	w.databus.SetSource(pfx+"can_go_previous", state.CanGoPrevious, "", "mediacontrol")
	w.databus.SetSource(pfx+"can_control", state.CanControl, "", "mediacontrol")
}

func (w *Watcher) publishPlayersList() {
	w.mu.RLock()
	type pInfo struct {
		Name     string `json:"name"`
		Identity string `json:"identity"`
	}
	list := make([]pInfo, 0, len(w.players))
	for name, st := range w.players {
		list = append(list, pInfo{Name: name, Identity: st.Identity})
	}
	w.mu.RUnlock()

	data, _ := json.Marshal(list)
	w.databus.SetSource("mediacontrol_available_players", string(data), "", "mediacontrol")
}

func (w *Watcher) clearDataBus() {
	keys := []string{
		"mediacontrol_player_name", "mediacontrol_identity", "mediacontrol_playback_status",
		"mediacontrol_title", "mediacontrol_artist", "mediacontrol_album", "mediacontrol_cover_url",
		"mediacontrol_progress", "mediacontrol_volume",
		"mediacontrol_can_play", "mediacontrol_can_pause", "mediacontrol_can_go_next",
		"mediacontrol_can_go_previous", "mediacontrol_can_control",
	}
	for _, k := range keys {
		w.databus.SetSource(k, "", "", "mediacontrol")
	}
}

// -----------------------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------------------

// appIDToName converts a Windows SourceAppUserModelId to a human-readable name.
// Examples: "Spotify.exe" → "Spotify", "Microsoft.ZuneMusic_…" → "Groove Music".
func appIDToName(appID string) string {
	known := map[string]string{
		"Spotify.exe":          "Spotify",
		"Microsoft.ZuneMusic":  "Groove Music",
		"wmplayer.exe":         "Windows Media Player",
		"vlc.exe":              "VLC",
		"foobar2000.exe":       "foobar2000",
		"chrome.exe":           "Chrome",
		"msedge.exe":           "Edge",
		"firefox.exe":          "Firefox",
		"opera.exe":            "Opera",
		"brave.exe":            "Brave",
	}
	for prefix, name := range known {
		if strings.HasPrefix(strings.ToLower(appID), strings.ToLower(prefix)) {
			return name
		}
	}
	// Strip .exe suffix and package family name suffix
	name := appID
	if idx := strings.Index(name, "_"); idx > 0 {
		name = name[:idx] // "Microsoft.ZuneMusic_8wekyb3d8bbwe" → "Microsoft.ZuneMusic"
	}
	if strings.HasSuffix(strings.ToLower(name), ".exe") {
		name = name[:len(name)-4]
	}
	// Take last segment of dotted name
	parts := strings.Split(name, ".")
	return parts[len(parts)-1]
}

// Compile-time check: ensure *Watcher satisfies the expected method set used by routes and state.
var _ interface {
	Start() error
	Close() error
	ListPlayers() []string
	GetPlayerState(string) *PlayerState
	GetSelectedPlayer() string
	SetSelectedPlayer(string) error
	CallMethod(string, string) error
	SetVolume(string, float64) error
} = (*Watcher)(nil)

// Reference to keep the collections import (used by GetSessions return type)
var _ *collections.IVectorView

