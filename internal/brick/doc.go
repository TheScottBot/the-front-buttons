// Package brick holds the brick kind registry, the schema derived from each kind's
// parameter struct, and the small interfaces a brick may implement.
//
// Parameters are declared once per kind; validation, decoding, editor forms and the
// generated reference all derive from that one declaration.
package brick
