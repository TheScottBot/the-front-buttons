package brick

import (
	"reflect"
	"strings"
)

// Appearance is the set of parameters every brick kind shares, declared once
// (specification section 2). A kind embeds it in its parameter struct, and its
// fields join that kind's schema ahead of the kind's own.
//
// The label bound is provisional: TE4's legibility gate on the physical deck may
// change it, which is a contract change recorded with its reason while the
// configuration contract is still provisional (datum 0.11).
type Appearance struct {
	Label      string `brick:"label"      bounds:"0,24"   description:"Text shown on the key."`
	Icon       string `brick:"icon"       bounds:"1,64"   format:"identifier"                                   description:"Name of a bundled icon."`
	Background string `brick:"background" format:"colour" description:"Key background colour, written #RRGGBB."`
	IconFile   string `brick:"icon_file"  bounds:"5,128"  format:"png_file"                                     description:"A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB."`
}

// AppearanceFieldNames lists the names of the shared appearance fields, read
// from Appearance's own tags, so a preset can pass them through to its target
// without keeping its own list.
func AppearanceFieldNames() []string {
	appearanceType := reflect.TypeFor[Appearance]()
	names := make([]string, 0, appearanceType.NumField())

	for structField := range appearanceType.Fields() {
		fieldName, _, _ := strings.Cut(structField.Tag.Get(brickTagKey), ",")
		names = append(names, fieldName)
	}

	return names
}
