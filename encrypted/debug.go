package encrypted

import (
	"crypto/ed25519"
	"time"

	"github.com/Arceliar/phony"
)

type Debug struct {
	pc *PacketConn
}

func (d *Debug) init(pc *PacketConn) {
	d.pc = pc
}

type DebugSessionInfo struct {
	Key    ed25519.PublicKey
	Uptime time.Duration
	RX     uint64
	TX     uint64
}

func (d *Debug) GetSessions() (infos []DebugSessionInfo) {
	var sessions []*sessionInfo
	phony.Block(&d.pc.sessions, func() {
		for _, session := range d.pc.sessions.sessions {
			sessions = append(sessions, session)
		}
	})
	// The counters belong to each session actor. Release the session
	// manager before waiting for those actors to avoid nested blocking.
	for _, session := range sessions {
		var info DebugSessionInfo
		phony.Block(session, func() {
			info.Key = append(info.Key, session.ed[:]...)
			info.Uptime = time.Since(session.since)
			info.RX, info.TX = session.rx, session.tx
		})
		infos = append(infos, info)
	}
	return
}
