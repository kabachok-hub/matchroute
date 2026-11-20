package main

import (
	"errors"
	"net"
	"net/netip"
	"strings"
	"sync"
	"testing"
)

func TestControllerStopsAfterAdapterReadFailure(t *testing.T) {
	dev := newChanDev()
	tcp, err := newTCPRelay(dev)
	if err != nil {
		t.Fatal(err)
	}
	engine := newEngine(&config{}, dev, clientIP, tcp)
	c := newController(defaultSettings())
	ticker := make(chan struct{})
	c.st, c.eng, c.tcp, c.dev, c.ticker = stateOn, engine, tcp, dev, ticker
	dev.Close()
	if err = engine.run(); !errors.Is(err, net.ErrClosed) {
		t.Fatalf("read error = %v", err)
	}
	c.engineFailed(engine, err)
	st, reason := c.state()
	if st != stateOff || !strings.Contains(reason, err.Error()) {
		t.Fatalf("state = %v, reason = %q", st, reason)
	}
	select {
	case <-ticker:
	default:
		t.Fatal("stats ticker was not stopped")
	}
}

func TestConcurrentControllerStops(t *testing.T) {
	dev := newChanDev()
	tcp, err := newTCPRelay(dev)
	if err != nil {
		t.Fatal(err)
	}
	engine := newEngine(&config{}, dev, clientIP, tcp)
	c := newController(defaultSettings())
	c.st, c.eng, c.tcp, c.dev, c.ticker = stateOn, engine, tcp, dev, make(chan struct{})
	var stops sync.WaitGroup
	for range 8 {
		stops.Go(c.stop)
	}
	stops.Wait()
	if st, _ := c.state(); st != stateOff {
		t.Fatalf("state = %v", st)
	}
}

func TestGameDestinationUsesOnlyGameNetworks(t *testing.T) {
	cfg := config{
		captureAll:     true,
		gameRanges:     [][2]uint16{{9000, 9999}},
		routes:         []netip.Prefix{netip.MustParsePrefix("192.0.2.0/24")},
		routedPrefixes: []netip.Prefix{netip.MustParsePrefix("0.0.0.0/0")},
	}
	if !cfg.gameDestination(netip.MustParseAddrPort("192.0.2.1:9000")) {
		t.Fatal("configured game endpoint was not recognized")
	}
	if cfg.gameDestination(netip.MustParseAddrPort("198.51.100.1:9000")) {
		t.Fatal("optional full IPv4 routing expanded game detection")
	}
}
