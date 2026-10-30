package encrypted

import (
	"bytes"
	"testing"
	"time"

	"github.com/Arceliar/phony"
)

func TestDebugGetSessions(t *testing.T) {
	var pc PacketConn
	pc.sessions.init(&pc)
	pc.Debug.init(&pc)
	if infos := pc.Debug.GetSessions(); len(infos) != 0 {
		t.Fatalf("empty sessions: got %d entries", len(infos))
	}

	key, otherKey := edPub{1}, edPub{2}
	since := time.Now().Add(-time.Minute)
	session := &sessionInfo{ed: key, since: since}
	other := &sessionInfo{ed: otherKey, since: since, rx: 3, tx: 5}
	pc.sessions.sessions[key] = session
	pc.sessions.sessions[otherKey] = other

	for _, counter := range []uint64{0, 1, 1 << 32, ^uint64(0)} {
		// The getter must include updates already queued on the session actor.
		session.Act(nil, func() {
			session.rx, session.tx = counter, ^counter
		})
		before := time.Since(since)
		infos := pc.Debug.GetSessions()
		after := time.Since(since)
		if len(infos) != 2 {
			t.Fatalf("got %d sessions, want 2", len(infos))
		}
		for _, info := range infos {
			if info.Uptime < before || info.Uptime > after {
				t.Errorf("uptime %v outside [%v, %v]", info.Uptime, before, after)
			}
			switch {
			case bytes.Equal(info.Key, key[:]):
				if info.RX != counter || info.TX != ^counter {
					t.Errorf("counters = (%d, %d), want (%d, %d)", info.RX, info.TX, counter, ^counter)
				}
			case bytes.Equal(info.Key, otherKey[:]):
				if info.RX != 3 || info.TX != 5 {
					t.Errorf("other session counters = (%d, %d), want (3, 5)", info.RX, info.TX)
				}
			default:
				t.Fatalf("unexpected session key: %x", info.Key)
			}
			info.Key[0] = 99
		}
	}
	phony.Block(session, func() {
		if session.ed != key {
			t.Error("returned key aliases the session key")
		}
	})
	phony.Block(other, func() {
		if other.ed != otherKey {
			t.Error("returned key aliases the second session key")
		}
	})
}
