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
	flag_shift = 1 << 0  // key
	flag_ctrl  = 1 << 1  // key
	flag_turbo = 1 << 13 // show turbo gauge
	flag_km    = 1 << 14 // user prefers km instead of miles
	flag_bar   = 1 << 15 // user prefers bar instead of psi
)

// bitshift constants for dash lights
const (
	light_shift      = 1 << 0
	light_fullbeam   = 1 << 1
	light_handbrake  = 1 << 2
	light_pitspeed   = 1 << 3 // seems not used by BeamNG
	light_tc         = 1 << 4
	light_signal_l   = 1 << 5
	light_signal_r   = 1 << 6
	light_signal_any = 1 << 7 // seems not used by BeamNG
	light_oilwarn    = 1 << 8
	light_battery    = 1 << 9
	light_abs        = 1 << 10
)

type Flags struct {
	ShiftKey bool
	CtrlKey  bool
	Turbo    bool
	Km       bool
	Bar      bool
}

type Lights struct {
	Shift           bool
	Fullbeam        bool
	Handbrake       bool
	PitSpeedLimiter bool
	TractionControl bool
	SignalLeft      bool
	SignalRight     bool
	SignalAny       bool
	OilWarning      bool
	Battery         bool
	Abs             bool
}

type OutGaugeDataRaw struct {
	Time        uint32   // 4 bytes // to check order (in ms) but always 0 atm
	Car         [4]byte  // 4 bytes // always "beam" from BeamNG
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
	Display1    [16]byte // 16 bytes // Usually Fuel but empty from BeamNG, so i cant test it
	Display2    [16]byte // 16 bytes // Usually Settings but empty from BeamNG, so i cant test it
	ID          int32    // 4 bytes // only if OutGauge ID is specified
} // 96 bytes of data per package
const buffersize = 96

type OutGaugeData struct {
	OutGaugeDataRaw
	Flags      Flags
	HasLights  Lights
	ShowLights Lights
}

type OutGaugeListener struct {
	conn               *net.UDPConn
	outgoingDatastream chan *OutGaugeData
	logger             *log.Logger
}

const outgoingDatastreamBuffersize = 100

// Creates a new OutGauge UDP listener on the specified ip:port
func NewListener(listen *net.UDPAddr) (*OutGaugeListener, error) {
	conn, err := net.ListenUDP("udp", listen)
	if err != nil {
		return nil, err
	}
	obj := OutGaugeListener{
		conn:               conn,
		outgoingDatastream: make(chan *OutGaugeData, outgoingDatastreamBuffersize),
		logger:             log.New(os.Stderr, "[OutGauge] ", log.LstdFlags),
	}
	go obj.getData() // start listening for data in background
	return &obj, nil
}

// Close the listener
func (l *OutGaugeListener) Close() {
	l.conn.Close()
}

// Return a read-only channel that will receive the data
func (l *OutGaugeListener) GetChannel() <-chan *OutGaugeData {
	return l.outgoingDatastream
}

func (l *OutGaugeListener) getData() {
	for {
		recvbuffer := make([]byte, buffersize)
		n, _, err := l.conn.ReadFrom(recvbuffer)
		if err != nil {
			// if udp socket is closed, close everything else and exit go routine
			close(l.outgoingDatastream)
			l.conn.Close() // just to be sure if some other error occurs
			return
		}
		if n != buffersize {
			// if we don't get the right amount of bytes, ignore it
			continue
		}

		data, err := parseData(recvbuffer)
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
		ShiftKey: raw&flag_shift != 0,
		CtrlKey:  raw&flag_ctrl != 0,
		Turbo:    raw&flag_turbo != 0,
		Km:       raw&flag_km != 0,
		Bar:      raw&flag_bar != 0,
	}
}

func rawBytesToLights(raw uint32) Lights {
	return Lights{
		Shift:           raw&light_shift != 0,
		Fullbeam:        raw&light_fullbeam != 0,
		Handbrake:       raw&light_handbrake != 0,
		PitSpeedLimiter: raw&light_pitspeed != 0,
		TractionControl: raw&light_tc != 0,
		SignalLeft:      raw&light_signal_l != 0,
		SignalRight:     raw&light_signal_r != 0,
		SignalAny:       raw&light_signal_any != 0,
		OilWarning:      raw&light_oilwarn != 0,
		Battery:         raw&light_battery != 0,
		Abs:             raw&light_abs != 0,
	}
}
