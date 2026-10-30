package network

import (
	"bytes"
	"net"
	"testing"
	"time"

	"github.com/Arceliar/phony"
)

func TestDebugGetPeers(t *testing.T) {
	var c core
	c.peers.init(&c)
	var d Debug
	d.init(&c)
	if infos := d.GetPeers(); len(infos) != 0 {
		t.Fatalf("empty peers: got %d entries", len(infos))
	}

	left, right := net.Pipe()
	defer left.Close()
	defer right.Close()
	key := publicKey{1}
	// Two connections to the same key must both be included.
	p := &peer{key: key, port: 7, prio: 3, conn: left}
	q := &peer{key: key, port: 7, prio: 5, conn: right}
	c.peers.peers[key] = map[*peer]struct{}{p: {}, q: {}}
	c.router.lags = map[*peer]time.Duration{p: 7 * time.Millisecond}
	start := time.Now()

	for _, tc := range []struct {
		delta time.Duration
		want  time.Duration
	}{
		{-time.Millisecond, 0},
		{0, 0},
		{4 * time.Microsecond, 0},
		{5 * time.Microsecond, 10 * time.Microsecond},
		{16 * time.Microsecond, 20 * time.Microsecond},
		{time.Second, time.Second},
	} {
		t.Run(tc.delta.String(), func(t *testing.T) {
			// The getter must include updates already queued on the peer actor.
			p.Act(nil, func() {
				p.srst = start
				p.srrt = start.Add(tc.delta)
			})
			infos := d.GetPeers()
			if len(infos) != 2 {
				t.Fatalf("got %d peers, want 2", len(infos))
			}
			for _, info := range infos {
				if !bytes.Equal(info.Key, key[:]) || info.Port != 7 {
					t.Fatalf("unexpected identity: %+v", info)
				}
				switch info.Conn {
				case left:
					if info.Cost != 7 || info.Priority != 3 || info.Latency != tc.want {
						t.Errorf("unexpected peer snapshot: %+v; want latency %v", info, tc.want)
					}
				case right:
					if info.Cost != 0 || info.Priority != 5 || info.Latency != 0 {
						t.Errorf("unexpected peer without cost/timestamps: %+v", info)
					}
				default:
					t.Fatalf("unexpected connection: %v", info.Conn)
				}
				info.Key[0] = 99
			}
			phony.Block(p, func() {
				if p.key != key {
					t.Error("returned key aliases the peer key")
				}
			})
			phony.Block(q, func() {
				if q.key != key {
					t.Error("returned key aliases the second peer key")
				}
			})
		})
	}
}
