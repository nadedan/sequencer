package sequencer

import "errors"

var (
	ErrNoPacketReady = errors.New("no_packet_ready")
	ErrCtxCanceled   = errors.New("ctx_canceled")
)
