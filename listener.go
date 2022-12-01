package outgaugelistener

import (
	"bytes"
	"encoding/binary"
	"log"
	"net"
	"os"
)

// bitshift constants for flags
const (
	flag_turbo = 1 << 13 // show turbo gauge
	flag_km    = 1 << 14 // if not set - user prefers miles
	flag_bar   = 1 << 15 // if not set - user prefers psi
)

// bitshift constants for dash lights
const (
	light_shift     = 1 << 0
	light_fullbeam  = 1 << 1
	light_handbrake = 1 << 2
	light_tc        = 1 << 4
	light_signal_l  = 1 << 5
	light_signal_r  = 1 << 6
	light_oilwarn   = 1 << 8
	light_battery   = 1 << 9
	light_abs       = 1 << 10
)

type Flags struct {
	Turbo bool
	Km    bool
	Bar   bool
}

type Lights struct {
	Shift           bool
	Fullbeam        bool
	Handbrake       bool
	TractionControl bool
	SignalLeft      bool
	SignalRight     bool
	OilWarning      bool
	Battery         bool
	Abs             bool
}

type OutGaugeDataRaw struct {
	Time        uint32   // 4 bytes // to check order (in ms) but always 0 atm
	Car         [4]byte  // 4 bytes // always "beam"
	Flags       uint16   // 2 bytes // Info
	Gear        byte     // 1 byte // Reverse 0, Neutral:1, First:2...
	PLID        byte     // 1 byte // unique player ID
	Speed       float32  // 4 bytes // meter per second
	RPM         float32  // 4 bytes // RPM
	Turbo       float32  // 4 bytes // bar
	EngTemp     float32  // 4 bytes // Celsius
	Fuel        float32  // 4 bytes // 0 to 1 (percentage)
	OilPressure float32  // 4 bytes // bar
	OilTemp     float32  // 4 bytes // Celsius
	DashLights  uint32   // 4 bytes // bitfield of available dash lights
	ShowLights  uint32   // 4 bytes // bitfield of currently active dash lights
	Throttle    float32  // 4 bytes // 0 to 1 (percentage)
	Brake       float32  // 4 bytes // 0 to 1 (percentage)
	Clutch      float32  // 4 bytes // 0 to 1 (percentage)
	Display1    [16]byte // 16 bytes // Usually Fuel but empty atm depending on car
	Display2    [16]byte // 16 bytes // Usually Settings but empty atm depending on car
	ID          int32    // 4 bytes // only if OutGauge ID is specified
} // 96 bytes of data

type OutGaugeData struct {
	OutGaugeDataRaw
	Flags      Flags
	HasLights  Lights
	ShowLights Lights
}

type OutGaugeListener struct {
	conn               *net.UDPConn
	outgoingDatastream chan *OutGaugeData
	closeListener      chan bool
	logger             *log.Logger
}

const channelbuffersize = 100

// Creates a new OutGauge UDP listener on the specified ip:port, set debug to true to enable more logging
func NewListener(listen *net.UDPAddr, debug bool) (*OutGaugeListener, error) {
	conn, err := net.ListenUDP("udp", listen)
	if err != nil {
		return nil, err
	}
	obj := OutGaugeListener{
		conn:               conn,
		outgoingDatastream: make(chan *OutGaugeData, channelbuffersize),
		closeListener:      make(chan bool, 1),
		logger:             log.New(os.Stderr, "[OutGauge] ", log.LstdFlags),
	}
	go obj.getData()
	return &obj, nil
}

// Close the listener
func (l *OutGaugeListener) Close() error {
	l.closeListener <- true
	err := l.conn.Close()
	close(l.outgoingDatastream)
	return err
}

// Return a channel that will receive the data
func (l *OutGaugeListener) GetChannel() <-chan *OutGaugeData {
	return l.outgoingDatastream
}

func (l *OutGaugeListener) getData() {
	for {
		select {
		case <-l.closeListener:
			l.logger.Println("Closing channel")
			return
		default:
			var target OutGaugeData
			size := binary.Size(&target)
			buffer := make([]byte, size)
			_, _, err := l.conn.ReadFrom(buffer)
			if err != nil {
				if err.Error() == net.ErrClosed.Error() {
					// if udp socket is closed, close the channel
					l.Close()
					return
				}
				l.logger.Panicf("Error reading from UDP: %v", err)
				continue
			}

			data, err := parseData(buffer)
			if err != nil {
				l.logger.Printf("Error decoding incoming data: %v", err)
				continue
			}

			select {
			case l.outgoingDatastream <- data:
			default:
				l.logger.Println("Channel full, dropping data...")
			}
		}
	}
}

func parseData(buffer []byte) (*OutGaugeData, error) {
	var target OutGaugeDataRaw
	if err := binary.Read(bytes.NewReader(buffer), binary.LittleEndian, &target); err != nil {
		return nil, err
	}

	result := OutGaugeData{
		OutGaugeDataRaw: target,
		Flags:           rawBytesToFlags(target.Flags),
		HasLights:       rawBytesToLights(target.DashLights),
		ShowLights:      rawBytesToLights(target.ShowLights),
	}
	return &result, nil
}

func rawBytesToFlags(raw uint16) Flags {
	return Flags{
		Turbo: raw&flag_turbo != 0,
		Km:    raw&flag_km != 0,
		Bar:   raw&flag_bar != 0,
	}
}

func rawBytesToLights(raw uint32) Lights {
	return Lights{
		Shift:           raw&light_shift != 0,
		Fullbeam:        raw&light_fullbeam != 0,
		Handbrake:       raw&light_handbrake != 0,
		TractionControl: raw&light_tc != 0,
		SignalLeft:      raw&light_signal_l != 0,
		SignalRight:     raw&light_signal_r != 0,
		OilWarning:      raw&light_oilwarn != 0,
		Battery:         raw&light_battery != 0,
		Abs:             raw&light_abs != 0,
	}
}
