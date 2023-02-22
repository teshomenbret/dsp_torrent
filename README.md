# [Torrent]
BitTorrent is a peer-to-peer (P2P) file sharing protocol that enables users to distribute and download large files quickly and efficiently. The technology was developed by Bram Cohen in 2001 and has since become one of the most popular methods of sharing files over the internet.
## [A torrent cliant implementation in go]

### Installation
> 1 Clone the repository
### Using the torrent
---write this command on the terminal

```
npm install @reduxjs/toolkit react-redux
```

### some word about **BitTorrent**
BitTorrent is a peer-to-peer (P2P) file sharing protocol that enables users to distribute and download large files quickly and efficiently. The technology was developed by Bram Cohen in 2001 and has since become one of the most popular methods of sharing files over the internet.

#### Process in Downloading Torrent file
##### Parsing a .torrent file
>In order to use BitTorrent to download content, you must first obtain a torrent file. This file includes details about the data you want to download and provides information about the location of the tracker. for this assignment we use **debian-edu-11.6.0-amd64-netinst.iso** torrent downloader.
 The .torrent file contains metadata for the torrent encoded using **Bencode**, Bencode is a simple and efficient format that represents data as a series of key-value pairs and uses a compact binary representation.
#### Get Peers from Tracker
>To participate in a BitTorrent swarm and start downloading or sharing content, you need to communicate with a **tracker**. This is done by sending a GET request to the announce URL specified in the .torrent file, along with some query parameters. The tracker's role is to keep track of all the peers participating in the swarm and to provide the requesting peer with a list of other available peers.
#### Parsing the tracker response
> After sending a GET request to the tracker's announce URL, the tracker will respond with an interval and a list of available peers in the swarm. To be able to use this information for later operations, you will need to parse the response and extract the relevant data.

The interval represents the number of seconds that should elapse before the client sends another request to the tracker. The list of peers contains information about other clients participating in the swarm, such as their IP address, port number, and peer ID.

#### Downloading from peers(Peer to Peer Communication)
>To start downloading pieces from the list of peers provided by the tracker, we need to follow a few steps. For each peer in the list, we will:

1 Initiate a TCP connection with the peer.
2 Perform a two-way BitTorrent handshake with the peer to establish a connection. This handshake includes sending and receiving information about the client's capabilities, such as the BitTorrent protocol version and supported extensions.
2 Exchange messages with the peer to request and download pieces. This involves sending messages that specify which pieces are needed and receiving messages that contain the actual data. The peer will also request pieces from the client in exchange.
