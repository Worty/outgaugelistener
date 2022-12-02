package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/worty/outgaugelistener"
)

func main() {
	ip := net.IP{}
	port := 4444
	flag.Parse()
	if len(flag.Args()) > 1 {
		ip = net.ParseIP(flag.Arg(1))
		if inputport, err := strconv.Atoi(flag.Arg(2)); err != nil {
			fmt.Println("Could not parse port, using default ", port)
		} else {
			port = inputport

		}
	}
	listenaddr := net.UDPAddr{IP: ip, Port: port}
	conn, err := outgaugelistener.NewListener(&listenaddr)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Listening to %v:%v\n", listenaddr.IP, listenaddr.Port)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	channel := conn.GetChannel()
	for {
		select {
		case data := <-channel:
			// fmt.Printf("%+v\n", *data) // Print all data
			fmt.Printf("Gear: %d Speed: %3.1f RPM: %4.0f Turbo: %+1.2f Lights: %+v\n", data.Gear, data.Speed, data.RPM, data.Turbo, data.ShowLights)
		case <-c:
			conn.Close()
			os.Exit(0)
			return
		}
	}
}
