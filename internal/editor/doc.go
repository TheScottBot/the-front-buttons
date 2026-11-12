// Package editor is the local web editor: a loopback HTTP server, its handlers
// and its templates (specification section 2).
//
// It is reached from a browser, so every request is untrusted, including
// requests forged by other pages open in the same browser. Every request must
// name the editor's own address as its Host, which defeats DNS rebinding; any
// Origin must be the editor's own; and every change must carry the token minted
// for this launch, which a page from elsewhere cannot read.
package editor
