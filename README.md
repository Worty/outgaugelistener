# OutGauge receiver in GO

## Description:

Library for receiving OutGauge pakets from simulators like BeamNG.drive®

Also contains a test cli program which prints incoming packets in `./cli` (`go run main.go <listen ip> <listen udp port>`)

## Usage:

`go get github.com/worty/outgaugelistener`

```go
    import (
        "github.com/worty/outgaugelistener"
    )
    conn, err := outgaugelistener.Listen(&listenaddr, false)
    ...
    defer conn.Close()
    channel := conn.GetChannel()
    // recive data from channel
    ...

```
