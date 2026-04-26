package panes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/initgrep-apps/spotnik/internal/api"
	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/initgrep-apps/spotnik/internal/uikit"
)

func TestTopArtistsPipeline_PopularityAndFollowers(t *testing.T) {
	body := `{"items":[{"id":"a1","name":"AP Dhillon","uri":"spotify:artist:a1","genres":["punjabi pop"],"popularity":77,"followers":{"href":null,"total":10390370},"external_urls":{}}],"total":1,"limit":25,"offset":0}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	client := api.NewUserClient(srv.URL, "tok")
	artists, err := client.TopArtists(context.Background(), "short_term", 25)
	if err != nil {
		t.Fatalf("TopArtists: %v", err)
	}
	if artists[0].Popularity != 77 {
		t.Errorf("popularity: got %d want 77", artists[0].Popularity)
	}
	if artists[0].Followers.Total != 10390370 {
		t.Errorf("followers: got %d want 10390370", artists[0].Followers.Total)
	}

	st := state.New()
	st.SetTopArtists("short_term", artists)
	st.SetTopTracks("short_term", []domain.Track{})
	st.StampStatsFetchedAt("short_term")

	pane := NewTopArtistsPane(st, theme.Load("black"), false)
	pane.SetSize(120, 20)
	view := pane.View()

	m := uikit.ActiveMode()
	on := uikit.GlyphFor(uikit.GlyphPinned, m)
	if !strings.Contains(view, on) {
		t.Errorf("expected filled stars in view, got:\n%s", view)
	}
	if !strings.Contains(view, "10M") {
		t.Errorf("expected 10M followers in view, got:\n%s", view)
	}
}
