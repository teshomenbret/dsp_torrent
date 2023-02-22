package leecher

import (
	"encoding/binary"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type bencodeTrackerResponse struct {
	Interval int    `bencode:"interval"`
	Peers    string `bencode:"peers"`
}

type Peer struct {
	IP   net.IP
	Port uint16
}

const peerSize = 6

func (t *TorrentFile) BuildTrackerURL(peerID [20]byte, port uint16) (string, error) {
	base, err := url.Parse(t.Announce)
	if err != nil {
		return "", fmt.Errorf("failed to parse tracker URL: %w", err)
	}
	params := url.Values{
		"info_hash":  {string(t.InfoHash[:])},
		"peer_id":    {string(peerID[:])},
		"port":       {strconv.Itoa(int(port))},
		"uploaded":   {"0"},
		"downloaded": {"0"},
		"compact":    {"1"},
		"left":       {strconv.Itoa(t.Length)},
	}
	base.RawQuery = params.Encode()
	return base.String(), nil
}

func (t *TorrentFile) requestPeers(peerID [20]byte, port uint16) ([]Peer, error) {
	urlStr, err := t.BuildTrackerURL(peerID, port)
	if err != nil {
		return nil, fmt.Errorf("failed to build tracker URL: %w", err)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(urlStr)
	if err != nil {
		return nil, fmt.Errorf("failed to get tracker response: %w", err)
	}
	defer resp.Body.Close()

	var trackerResp bencodeTrackerResponse
	if err := UnmarshalResponse(resp.Body, &trackerResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tracker response: %w", err)
	}

	return unmarshalPeers([]byte(trackerResp.Peers))
}

func unmarshalPeers(peersBin []byte) ([]Peer, error) {
	if len(peersBin)%peerSize != 0 {
		return nil, fmt.Errorf("received malformed peers")
	}
	numPeers := len(peersBin) / peerSize
	peers := make([]Peer, numPeers)
	for i := 0; i < numPeers; i++ {
		offset := i * peerSize
		peers[i].IP = net.IP(peersBin[offset : offset+4])
		peers[i].Port = binary.BigEndian.Uint16(peersBin[offset+4 : offset+6])
	}
	return peers, nil
}

func (p Peer) String() string {
	return net.JoinHostPort(p.IP.String(), strconv.FormatUint(uint64(p.Port), 10))
}
