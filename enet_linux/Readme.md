# GotoPS
Enet bindings in Go for Growtopia Private Server using cgo. Based on [codecat/go-enet](https://github.com/codecat/go-enet)

> [!TIP]
> Want to see an example of a server that uses this library? Look at [gotps](https://github.com/eikarna/gotps).

## Installation
First, you might need to build enet from enet/ directory and install it as library:

* **Windows**: Build the enet dynamic library (.dll) with Code::Blocks. (Tested with: TDM-GCC 10.3.0 installed, Windows 11 Intel Core I7-6700HQ).
```sh
git clone --recurse-submodules https://github.com/eikarna/gotops
cd gotops/enet
"C:\Program Files\CodeBlocks\codeblocks.exe" /na /nd enet_dll.cbp
```
If you got error message: 'IP_TTL' undeclared (first use in this function), just add `#include <Ws2tcpip.h>` to the top of win32.c.

#### REMEMBER TO BUILD TARGET "Release" ONLY.

Got Any Error again? just try to fix it by yourself.

After successfully build, you must copy build output files (include: libenet.a, libenet.dll, libenet.def) to your root project. And then you can use this go module without any problem.
* **Linux**: Install the enet library with make.
```sh
git clone --recurse-submodules https://github.com/eikarna/gotops
cd gotops/enet
autoreconf -vfi
./configure && make && sudo make install
```
* **Termux**: Same as Linux, but after run `autoreconf -vfi`:
```sh
./configure
make
make prefix=/data/data/com.termux/files/usr libdir=/data/data/com.termux/files/usr/lib install
```
If you got warning message: `libtool: warning: remember to run 'libtool --finish /usr/local/lib'`, just run:
```sh
libtool --finish /data/data/com.termux/files/usr/lib
```
* **MacOS**: Install the enet package with brew: `brew install enet`

```sh
$ go get github.com/eikarna/gotops
```

## Usage
```go
import "github.com/eikarna/gotops"
```

The API is mostly the same as the C API, except it's more object-oriented.

## Server example
This is a basic server example that responds to packets `"ping"` and `"bye"`.

```go
package main

import (
	"github.com/eikarna/gotops"
	"github.com/codecat/go-libs/log"
)

func main() {
	// Initialize enet
	enet.Initialize()

	// Create a host listening on 0.0.0.0:17091
	host, err := enet.NewHost(enet.NewListenAddress(17091), 1024, 1, 0, 0)
	if err != nil {
		log.Error("Couldn't create host: %s", err.Error())
		return
	}

	// The event loop
	for true {
		// Wait until the next event
		ev := host.Service(1000)

		// Do nothing if we didn't get any event
		if ev.GetType() == enet.EventNone {
			continue
		}

		switch ev.GetType() {
		case enet.EventConnect: // A new peer has connected
			log.Info("New peer connected: %s", ev.GetPeer().GetAddress())

		case enet.EventDisconnect: // A connected peer has disconnected
			log.Info("Peer disconnected: %s", ev.GetPeer().GetAddress())

		case enet.EventReceive: // A peer sent us some data
			// Get the packet
			packet := ev.GetPacket()

			// We must destroy the packet when we're done with it
			defer packet.Destroy()

			// Get the bytes in the packet
			packetBytes := packet.GetData()

			// Respond "pong" to "ping"
			if string(packetBytes) == "ping" {
				ev.GetPeer().SendString("pong", ev.GetChannelID(), enet.PacketFlagReliable)
				continue
			}

			// Disconnect the peer if they say "bye"
			if string(packetBytes) == "bye" {
				log.Info("Bye!")
				ev.GetPeer().Disconnect(0)
				continue
			}
		}
	}

	// Destroy the host when we're done with it
	host.Destroy()

	// Uninitialize enet
	enet.Deinitialize()
}
```

## Client example
This is a basic client example that sends a ping to the server every second that there is no event.

```go
package main

import (
	"github.com/eikarna/gotops"
	"github.com/codecat/go-libs/log"
)

func main() {
	// Initialize enet
	enet.Initialize()

	// Create a client host
	client, err := enet.NewHost(nil, 1, 1, 0, 0)
	if err != nil {
		log.Error("Couldn't create host: %s", err.Error())
		return
	}

	// Connect the client host to the server
	peer, err := client.Connect(enet.NewAddress("127.0.0.1", 17091), 1, 0)
	if err != nil {
		log.Error("Couldn't connect: %s", err.Error())
		return
	}

	// The event loop
	for true {
		// Wait until the next event
		ev := client.Service(1000)

		// Send a ping if we didn't get any event
		if ev.GetType() == enet.EventNone {
			peer.SendString("ping", 0, enet.PacketFlagReliable)
			continue
		}

		switch ev.GetType() {
		case enet.EventConnect: // We connected to the server
			log.Info("Connected to the server!")

		case enet.EventDisconnect: // We disconnected from the server
			log.Info("Lost connection to the server!")

		case enet.EventReceive: // The server sent us data
			packet := ev.GetPacket()
			log.Info("Received %d bytes from server", len(packet.GetData()))
			packet.Destroy()
		}
	}

	// Destroy the host when we're done with it
	client.Destroy()

	// Uninitialize enet
	enet.Deinitialize()
}
```
