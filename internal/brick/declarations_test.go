package brick_test

import (
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
)

// sampleParameters is a well formed parameter declaration covering every field
// type and tag, shared by the schema and registry tests.
type sampleParameters struct {
	brick.Appearance

	Address    string        `brick:"address,required"  bounds:"1,200"                  format:"https_url"               description:"Where the key goes."`
	Count      int           `brick:"count"             bounds:"1,10"                   description:"How many times."`
	Enabled    bool          `brick:"enabled"           description:"Whether it is on."`
	Interval   time.Duration `brick:"interval"          bounds:"30s,1h"                 description:"How often it runs."`
	Credential string        `brick:"credential,secret" bounds:"1,64"                   format:"identifier"              description:"Name of a stored credential."`
}

// The declarations below each break exactly one tag rule.

type missingBrickTag struct {
	Address string `description:"No brick tag."`
}

type unexportedField struct {
	Address string `brick:"address" description:"Fine."`
	hidden  string
}

type badFieldName struct {
	Address string `brick:"Address" description:"Upper case name."`
}

type duplicateFieldName struct {
	First  string `brick:"address" description:"One."`
	Second string `brick:"address" description:"Two."`
}

type duplicateOfAppearanceField struct {
	brick.Appearance

	Label string `brick:"label" description:"Clashes with the appearance label."`
}

type missingDescription struct {
	Address string `brick:"address"`
}

type unsupportedType struct {
	Ratio float64 `brick:"ratio" description:"Floats are not supported."`
}

type unknownTagOption struct {
	Address string `brick:"address,requird" description:"Misspelt option."`
}

type boundsOnBoolean struct {
	Enabled bool `brick:"enabled" bounds:"0,1" description:"Bounds make no sense here."`
}

type malformedBounds struct {
	Count int `brick:"count" bounds:"one,ten" description:"Not numbers."`
}

type invertedBounds struct {
	Count int `brick:"count" bounds:"10,1" description:"Minimum above maximum."`
}

type negativeTextBounds struct {
	Label string `brick:"label" bounds:"-1,10" description:"A length cannot be negative."`
}

type formatOnInteger struct {
	Count int `brick:"count" format:"digits" description:"Formats apply to text."`
}

type unknownFormat struct {
	Address string `brick:"address" format:"telephone" description:"No such format."`
}

type secretOnInteger struct {
	Count int `brick:"count,secret" description:"Secrets are credential names, so text."`
}

type otherEmbedded struct {
	Extra string `brick:"extra" description:"Only Appearance may be embedded."`
}

type embeddedOtherStruct struct {
	otherEmbedded
}

// The fields of unexportedField exist only to be rejected; reading hidden keeps
// the compiler and linters from flagging it unused.
var _ = unexportedField{Address: "", hidden: ""}.hidden
