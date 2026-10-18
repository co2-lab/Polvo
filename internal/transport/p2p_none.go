//go:build !(cgo && polvo_p2p)

// @anchors
//   code: TRNON
//   ref: TRNSP

package transport

// open is TRNSP-E02 in an executable built without the transport.
func open([]byte, string, int) (Peer, error) { return nil, ErrNotBuilt }
