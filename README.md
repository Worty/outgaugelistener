# OutGauge receiver in GO

## Description:

Library for receiving OutGauge pakets from simulators like BeamNG.drive®

Also contains a test cli program which prints incoming packets in [./cli/main.go](cli/main.go).

run it: `go run main.go <listen ip> <listen udp port>`

For information how setup the game checkout the [Wiki](https://wiki.beamng.com/OutGauge.html)

## Usage:

`go get github.com/worty/outgaugelistener`

```go
    import (
        "github.com/worty/outgaugelistener"
    )
    // listenaddr:= net.UDPAddr{.....}
    conn, err := outgaugelistener.Listen(&listenaddr)
    ...
    defer conn.Close()
    channel := conn.GetChannel()
    // recive data from channel
    ...

```
