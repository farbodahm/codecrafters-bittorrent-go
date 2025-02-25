[![progress-banner](https://backend.codecrafters.io/progress/bittorrent/83c9971f-35d3-470b-b692-49241327d32b)](https://app.codecrafters.io/users/codecrafters-bot?r=2qF)

My approach to the ['Build Your Own BitTorrent'](https://app.codecrafters.io/courses/bittorrent/overview) challenge.

## Features  

- **Encode/Decode Bencode Encoding**: 
  - `Strings`
  - `Int`
  - `Nested List`
  - `Nested Dictionary`
- **Parsing Torrent File**
- **Discovering Peers & Handshake**
- **Download Pieces from Peers concurrently**  


## RUN

- **Build**:
  - `go build -o bittorrent cmd/bittorrent/main.go`
- **Download full torrent**:
  - `./bittorrent download -o test.txt sample.torrent`
- **Download specific piece**:
  - `./bittorrent download_piece -o ./file-piece11 sample.torrent 11`
- **Discover Peers**:
  - `./bittorrent peers sample.torrent`
- **Handshake Peer**:
  - `./bittorrent handshake sample.torrent PEER_IP:PEER_PORT`
- **Parse Torrent**:
  - `./bittorrent info sample.torrent`
- **Decode Beencode**:
  - `./bittorrent decode d10:inner_dictd4:key16:value14:key2i42e8:list_keyl5:item15:item2i3eeee`
