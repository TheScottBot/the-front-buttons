package murals_test

import (
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/murals"
)

// teamBoard is the mural the test's board addresses name.
const teamBoard = "team1234.1712345678901"

// TestBoardID_ReadsTheMuralFromItsAddress: a board's address, as its browser
// shows it, names the mural Mural's API knows as its workspace and number;
// anything else names no mural.
func TestBoardID_ReadsTheMuralFromItsAddress(t *testing.T) {
	t.Parallel()

	for link, want := range map[string]string{
		"https://app.mural.co/t/team1234/m/team1234/1712345678901/0a1b2c3d":           teamBoard,
		"https://app.mural.co/t/team1234/m/team1234/1712345678901/0a1b2c3d?sender=me": teamBoard,
		"https://app.mural.co/t/team1234/m/team1234/1712345678901":                    teamBoard,
		"https://app.mural.co/invitation/mural/team1234.1712345678901":                "",
		"https://app.mural.co/t/team1234/m/team1234/notanumber/0a1b":                  "",
		"https://app.mural.co/t/team1234/m/other/1712345678901/0a1b":                  "other.1712345678901",
		"https://example.com/t/team1234/m/team1234/1712345678901/0a1b":                "",
		"http://app.mural.co/t/team1234/m/team1234/1712345678901/0a1b":                "",
		"": "",
	} {
		got, found := murals.BoardID(link)
		if got != want || found != (want != "") {
			t.Errorf("%q: want %q, got %q (%v)", link, want, got, found)
		}
	}
}
