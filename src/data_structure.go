package leecher

import "net"

// message
type messageID uint8

const (
	MsgChoke         messageID = 0
	MsgUnchoke       messageID = 1
	MsgInterested    messageID = 2
	MsgNotInterested messageID = 3
	MsgHave          messageID = 4
	MsgBitfield      messageID = 5
	MsgRequest       messageID = 6
	MsgPiece         messageID = 7
	MsgCancel        messageID = 8
)

type Message struct {
	ID      messageID
	Payload []byte
}

// TorrentFile encodes the metadata from a .torrent file
type TorrentFile struct {
	Announce    string
	InfoHash    [20]byte
	PieceHashes [][20]byte
	PieceLength int
	Length      int
	Name        string
}

// Torrent holds data required to download a torrent from a list of peers
type Torrent struct {
	Peers       []Peer
	PeerID      [20]byte
	InfoHash    [20]byte
	PieceHashes [][20]byte
	PieceLength int
	Length      int
	Name        string
}

// clianrt object
type Client struct {
	Conn     net.Conn
	Choked   bool
	Bitfield Bitfield
	peer     Peer
	infoHash [20]byte
	peerID   [20]byte
}

// A Handshake is a special message that a peer uses to identify itself
type HandShake struct {
	Pstr     string
	InfoHash [20]byte
	PeerID   [20]byte
}
