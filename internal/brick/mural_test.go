package brick_test

import (
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
)

// TestCheckFormat_MuralLink_AcceptsMuralsOwnLinksOnly: a Mural key takes a
// link to Mural itself, a board or an invitation, over https, and nothing
// else, so a key can only ever paste or act on a Mural address.
func TestCheckFormat_MuralLink_AcceptsMuralsOwnLinksOnly(t *testing.T) {
	t.Parallel()

	for value, want := range map[string]bool{
		"https://app.mural.co/t/team/m/team/1712345678901/0a1b2c3d":        true,
		"https://app.mural.co/t/team/m/team/1712345678901/0a1b2c3d?s=1":    true,
		"https://app.mural.co/invitation/mural/team1712345678901?sender=x": true,
		"http://app.mural.co/t/team/m/team/1712345678901/0a1b2c3d":         false,
		"https://app.mural.co.example.com/t/team/m/team/1/a":               false,
		"https://example.com/app.mural.co":                                 false,
		"https://someone@app.mural.co/t/team/m/team/1/a":                   false,
		"https://app.mural.co":                                             false,
		"":                                                                 false,
	} {
		if got := brick.CheckFormat(brick.FormatMuralLink, value); got != want {
			t.Errorf("CheckFormat(mural_link, %q) = %v, want %v", value, got, want)
		}
	}
}

// TestCheckFormat_MuralBoard_TakesABoardsOwnAddress: a timer key takes a
// board's own address, from which the mural it acts on is read, and not an
// invitation or anything else.
func TestCheckFormat_MuralBoard_TakesABoardsOwnAddress(t *testing.T) {
	t.Parallel()

	for value, want := range map[string]bool{
		"https://app.mural.co/t/team/m/team/1712345678901/0a1b2c3d":        true,
		"https://app.mural.co/invitation/mural/team1712345678901?sender=x": false,
		"https://app.mural.co/t/team/m/team/abc/0a1b2c3d":                  false,
		"": false,
	} {
		if got := brick.CheckFormat(brick.FormatMuralBoard, value); got != want {
			t.Errorf("CheckFormat(mural_board, %q) = %v, want %v", value, got, want)
		}
	}
}
