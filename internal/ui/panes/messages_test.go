package panes

import (
	"testing"

	"github.com/initgrep-apps/spotnik/internal/domain"
)

// TestPodcastMessageTypes_Compile verifies that all podcast message types can be
// created with zero values (compilation check).
func TestPodcastMessageTypes_Compile(t *testing.T) {
	_ = FetchFollowedShowsRequestMsg{}

	_ = FetchSavedEpisodesRequestMsg{}

	_ = FetchShowEpisodesRequestMsg{ShowID: "show-1"}

	_ = FollowedShowsLoadedMsg{
		Items: []domain.SavedShow{},
		Err:   nil,
	}

	_ = SavedEpisodesLoadedMsg{
		Items: []domain.SavedEpisode{},
		Err:   nil,
	}

	_ = ShowEpisodesLoadedMsg{
		ShowID:  "show-1",
		Items:   []domain.Episode{},
		Total:   42,
		HasNext: true,
		Err:     nil,
	}

	_ = SelectedShowChangedMsg{ShowID: "show-abc"}

	_ = PlayEpisodeMsg{
		EpisodeURI:  "spotify:episode:ep-1",
		PlaylistURI: "spotify:show:show-1",
	}
}
