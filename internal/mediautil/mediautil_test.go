package mediautil

import (
	"errors"
	"path/filepath"
	"reflect"
	"testing"

	"omnipanel-go/internal/mpris"
)

type fakeWatcher struct {
	players         []string
	states          map[string]*mpris.PlayerState
	selected        string
	setSelectedErr  error
	callErr         error
	calledPlayer    string
	calledMethod    string
	setSelectedWith string
}

func (f *fakeWatcher) ListPlayers() []string { return f.players }
func (f *fakeWatcher) GetPlayerState(playerName string) *mpris.PlayerState {
	return f.states[playerName]
}
func (f *fakeWatcher) GetSelectedPlayer() string { return f.selected }
func (f *fakeWatcher) SetSelectedPlayer(playerName string) error {
	f.setSelectedWith = playerName
	if f.setSelectedErr != nil {
		return f.setSelectedErr
	}
	f.selected = playerName
	return nil
}
func (f *fakeWatcher) CallMethod(playerName, method string) error {
	f.calledPlayer = playerName
	f.calledMethod = method
	return f.callErr
}

func TestBuildPlayersResponse_Disabled(t *testing.T) {
	got := BuildPlayersResponse(nil, 0.25)
	want := map[string]any{
		"enabled": false,
		"players": []string{},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("BuildPlayersResponse(nil) = %#v, want %#v", got, want)
	}
}

func TestBuildPlayersResponse_WithPlayersSkipsNilStates(t *testing.T) {
	w := &fakeWatcher{
		players: []string{"spotify", "vlc"},
		states: map[string]*mpris.PlayerState{
			"spotify": {
				PlayerName:     "spotify",
				Identity:       "Spotify",
				PlaybackStatus: "Playing",
				Title:          "Track",
				Artist:         "Artist",
				Album:          "Album",
				ArtURL:         "file:///tmp/cover.jpg",
				CanControl:     true,
				Volume:         0.8,
			},
			"vlc": nil,
		},
	}

	got := BuildPlayersResponse(w, 0.42)
	if got["enabled"] != true {
		t.Fatalf("enabled = %v, want true", got["enabled"])
	}
	if got["systemVolume"] != 0.42 {
		t.Fatalf("systemVolume = %v, want 0.42", got["systemVolume"])
	}
	players, ok := got["players"].([]map[string]any)
	if !ok {
		t.Fatalf("players has unexpected type %T", got["players"])
	}
	if len(players) != 1 {
		t.Fatalf("len(players) = %d, want 1", len(players))
	}
	if players[0]["name"] != "spotify" || players[0]["identity"] != "Spotify" {
		t.Fatalf("unexpected player payload: %#v", players[0])
	}
}

func TestResolvePlayer(t *testing.T) {
	tests := []struct {
		name      string
		watcher   Watcher
		requested string
		want      string
		wantErr   error
	}{
		{name: "requested wins", watcher: &fakeWatcher{selected: "selected", players: []string{"first"}}, requested: "req", want: "req"},
		{name: "nil watcher", watcher: nil, requested: "", wantErr: ErrIntegrationDisabled},
		{name: "selected fallback", watcher: &fakeWatcher{selected: "selected"}, want: "selected"},
		{name: "first player fallback", watcher: &fakeWatcher{players: []string{"first", "second"}}, want: "first"},
		{name: "no players", watcher: &fakeWatcher{}, wantErr: ErrNoPlayersConnected},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolvePlayer(tt.watcher, tt.requested)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ResolvePlayer() error = %v, want %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("ResolvePlayer() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExecuteControl(t *testing.T) {
	setCalls := 0
	setVol := 0.0
	setFn := func(v float64) error {
		setCalls++
		setVol = v
		return nil
	}

	w := &fakeWatcher{selected: "spotify", players: []string{"spotify"}}
	player, err := ExecuteControl(w, ControlRequest{Action: "playpause"}, setFn)
	if err != nil {
		t.Fatalf("ExecuteControl(playpause) error = %v", err)
	}
	if player != "spotify" || w.calledPlayer != "spotify" || w.calledMethod != "PlayPause" {
		t.Fatalf("unexpected call routing: player=%q calledPlayer=%q method=%q", player, w.calledPlayer, w.calledMethod)
	}
	if setCalls != 0 {
		t.Fatalf("setSystemVolume called unexpectedly")
	}

	player, err = ExecuteControl(w, ControlRequest{Action: "volume", Volume: 0.75}, setFn)
	if err != nil {
		t.Fatalf("ExecuteControl(volume) error = %v", err)
	}
	if player != "spotify" || setCalls != 1 || setVol != 0.75 {
		t.Fatalf("volume control mismatch: player=%q setCalls=%d setVol=%v", player, setCalls, setVol)
	}

	explicit := &fakeWatcher{selected: "spotify", players: []string{"spotify", "vlc"}}
	player, err = ExecuteControl(explicit, ControlRequest{Player: "vlc", Action: "next"}, setFn)
	if err != nil {
		t.Fatalf("ExecuteControl(explicit player) error = %v", err)
	}
	if player != "vlc" || explicit.calledPlayer != "vlc" || explicit.calledMethod != "Next" {
		t.Fatalf("explicit player routing mismatch: player=%q calledPlayer=%q method=%q", player, explicit.calledPlayer, explicit.calledMethod)
	}

	branchCases := []struct {
		action string
		method string
	}{
		{action: "play", method: "Play"},
		{action: "pause", method: "Pause"},
		{action: "stop", method: "Stop"},
		{action: "previous", method: "Previous"},
	}
	for _, tc := range branchCases {
		branchWatcher := &fakeWatcher{selected: "spotify", players: []string{"spotify"}}
		player, err := ExecuteControl(branchWatcher, ControlRequest{Action: tc.action}, setFn)
		if err != nil {
			t.Fatalf("ExecuteControl(%s) error = %v", tc.action, err)
		}
		if player != "spotify" || branchWatcher.calledPlayer != "spotify" || branchWatcher.calledMethod != tc.method {
			t.Fatalf("ExecuteControl(%s) routed player=%q calledPlayer=%q method=%q, want player=spotify method=%s", tc.action, player, branchWatcher.calledPlayer, branchWatcher.calledMethod, tc.method)
		}
	}
}

func TestExecuteControl_Errors(t *testing.T) {
	noOpSet := func(float64) error { return nil }
	callErr := errors.New("boom")
	wCallErr := &fakeWatcher{selected: "spotify", callErr: callErr}
	setErr := errors.New("set volume failed")
	setErrFn := func(float64) error { return setErr }

	tests := []struct {
		name    string
		watcher Watcher
		req     ControlRequest
		setFn   func(float64) error
		wantErr error
		msg     string
	}{
		{name: "disabled", watcher: nil, req: ControlRequest{Action: "play"}, setFn: noOpSet, wantErr: ErrIntegrationDisabled},
		{name: "no players", watcher: &fakeWatcher{}, req: ControlRequest{Action: "play"}, setFn: noOpSet, wantErr: ErrNoPlayersConnected},
		{name: "invalid volume", watcher: &fakeWatcher{selected: "spotify"}, req: ControlRequest{Action: "volume", Volume: 2}, setFn: noOpSet, wantErr: ErrInvalidVolume},
		{name: "set volume failure", watcher: &fakeWatcher{selected: "spotify"}, req: ControlRequest{Action: "volume", Volume: 0.4}, setFn: setErrFn, wantErr: setErr},
		{name: "call failure", watcher: wCallErr, req: ControlRequest{Action: "next"}, setFn: noOpSet, wantErr: callErr},
		{name: "unknown action", watcher: &fakeWatcher{selected: "spotify"}, req: ControlRequest{Action: "dance"}, setFn: noOpSet, msg: "Unknown action: dance"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ExecuteControl(tt.watcher, tt.req, tt.setFn)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err == nil || err.Error() != tt.msg {
				t.Fatalf("error = %v, want %q", err, tt.msg)
			}
		})
	}
}

func TestSelectPlayer(t *testing.T) {
	w := &fakeWatcher{}
	if err := SelectPlayer(nil, "spotify"); !errors.Is(err, ErrIntegrationDisabled) {
		t.Fatalf("SelectPlayer(nil) error = %v", err)
	}
	if err := SelectPlayer(w, ""); !errors.Is(err, ErrPlayerRequired) {
		t.Fatalf("SelectPlayer(empty) error = %v", err)
	}
	setErr := errors.New("missing")
	w.setSelectedErr = setErr
	if err := SelectPlayer(w, "spotify"); !errors.Is(err, setErr) {
		t.Fatalf("SelectPlayer(setErr) error = %v", err)
	}
	w.setSelectedErr = nil
	if err := SelectPlayer(w, "spotify"); err != nil {
		t.Fatalf("SelectPlayer(valid) error = %v", err)
	}
	if w.setSelectedWith != "spotify" {
		t.Fatalf("selected with = %q, want spotify", w.setSelectedWith)
	}
}

func TestNormalizeFileURLPath(t *testing.T) {
	if got := NormalizeFileURLPath("file:///tmp/cover.jpg"); got != "/tmp/cover.jpg" {
		t.Fatalf("NormalizeFileURLPath() = %q", got)
	}
	if got := NormalizeFileURLPath("C:/plain/path.jpg"); got != "C:/plain/path.jpg" {
		t.Fatalf("NormalizeFileURLPath(no prefix) = %q", got)
	}
}

func TestAllowedCoverPrefixes(t *testing.T) {
	home := "/home/tester"
	temp := filepath.Join("C:", "Temp")
	got := AllowedCoverPrefixes(home, temp)
	want := []string{"/tmp/", "/var/tmp/", "/home/tester/.cache/", temp + string(filepath.Separator)}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("AllowedCoverPrefixes() = %#v, want %#v", got, want)
	}
}

func TestIsAllowedCoverPath(t *testing.T) {
	prefixes := []string{"/tmp/", "C:\\Temp\\"}
	if !IsAllowedCoverPath("/tmp/cover.jpg", prefixes) {
		t.Fatalf("expected linux temp path to be allowed")
	}
	if !IsAllowedCoverPath("C:\\Temp\\cover.jpg", prefixes) {
		t.Fatalf("expected windows temp path to be allowed")
	}
	if IsAllowedCoverPath("C:\\Windows\\secret.jpg", prefixes) {
		t.Fatalf("unexpected allowed path")
	}
}

func TestContentTypeForPath(t *testing.T) {
	cases := map[string]string{
		"cover.png":  "image/png",
		"cover.gif":  "image/gif",
		"cover.webp": "image/webp",
		"cover.jpg":  "image/jpeg",
		"cover.jpeg": "image/jpeg",
	}
	for path, want := range cases {
		if got := ContentTypeForPath(path); got != want {
			t.Fatalf("ContentTypeForPath(%q) = %q, want %q", path, got, want)
		}
	}
}


