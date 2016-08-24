package main

import (
	"bufio"
	"container/list"
	"encoding/json"
	"log"
	"math"
	"net"
	"os"
	"strconv"
	"sync"
	"time"
)

const Period = 1 * time.Second

var mu sync.Mutex
var subs = list.New()

func main() {
	log.SetOutput(os.Stdout)

	amp := amplitude()
	log.Printf("Amplitude: %f", amp)

	waveform := os.Getenv("GEN_WAVEFORM")
	var g Generator
	switch waveform {
	case "square":
		g = &ArrayGenerator{factors: square}
	case "triangle":
		g = &ArrayGenerator{factors: triangle}
	case "sawtooth":
		g = &ArrayGenerator{factors: sawtooth}
	default:
		waveform = "sine"
		g = &SineGenerator{step: math.Pi / 8}
	}
	log.Printf("Waveform: %s", waveform)
	go emit(g, amp)

	ln, err := net.Listen("tcp", net.JoinHostPort("", port()))
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()
	log.Printf("Listen on %s", ln.Addr())
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("Connection from %s", conn.RemoteAddr())
		addSubscriber(conn)
	}
}

func port() string {
	port := os.Getenv("GEN_PORT")
	if port != "" {
		return port
	}
	// TODO: remove backwards compatibility
	port = os.Getenv("GENERATOR_PORT")
	if port != "" {
		return port
	}
	return "3000"
}

func amplitude() float64 {
	vf, err := strconv.ParseFloat(os.Getenv("GEN_AMPLITUDE"), 32)
	if err == nil {
		return vf
	}
	vi, err := strconv.ParseInt(os.Getenv("GEN_AMPLITUDE"), 10, 32)
	if err == nil {
		return float64(vi)
	}
	return 100.0
}

func addSubscriber(conn net.Conn) {
	mu.Lock()
	defer mu.Unlock()
	subs.PushBack(NewSubscriber(conn))
}

func updateSubscribers(sample *Sample) {
	mu.Lock()
	defer mu.Unlock()
	log.Printf("Subscribers: %d", subs.Len())
	e := subs.Front()
	for e != nil {
		s := e.Value.(*Subscriber)
		if err := s.Send(sample); err != nil {
			log.Printf("Subscriber failed, err: %v", err)
			subs.Remove(e)
		}
		e = e.Next()
	}
}

func emit(g Generator, amp float64) {
	ticker := time.NewTicker(Period)
	defer ticker.Stop()
	for {
		select {
		case t := <-ticker.C:
			v := g.Next() * amp
			log.Printf("Emitting value: %f", v)
			s := &Sample{
				Timestamp: t.Unix(),
				Value:     v,
			}
			updateSubscribers(s)
		}
	}
}

type Sample struct {
	Timestamp int64   `json:"timestamp"`
	Value     float64 `json:"value"`
}

type Subscriber struct {
	conn net.Conn
	w    *bufio.Writer
	enc  *json.Encoder
}

func NewSubscriber(conn net.Conn) *Subscriber {
	w := bufio.NewWriter(conn)
	enc := json.NewEncoder(w)
	return &Subscriber{conn: conn, w: w, enc: enc}
}
func (s *Subscriber) Send(sample *Sample) error {
	if err := s.enc.Encode(sample); err != nil {
		return err
	}
	if err := s.w.WriteByte(0); err != nil {
		return err
	}
	if err := s.w.Flush(); err != nil {
		return err
	}
	return nil
}

func (s *Subscriber) Close() {
	s.conn.Close()
}

type Generator interface {
	Next() float64
}

type ArrayGenerator struct {
	factors []float64
	index   int
}

func (g *ArrayGenerator) Next() float64 {
	f := g.factors[g.index]
	g.index++
	if g.index == len(g.factors) {
		g.index = 0
	}
	return f
}

type SineGenerator struct {
	step float64
	x    float64
}

func (g *SineGenerator) Next() float64 {
	f := math.Sin(g.x)
	g.x += g.step
	return f
}

var square = []float64{
	1.000000,
	1.000000,
	1.000000,
	1.000000,
	1.000000,
	1.000000,
	1.000000,
	1.000000,
	-1.000000,
	-1.000000,
	-1.000000,
	-1.000000,
	-1.000000,
	-1.000000,
	-1.000000,
	-1.000000,
}

var triangle = []float64{
	-1.000000,
	-0.750000,
	-0.500000,
	-0.250000,
	0.000000,
	0.250000,
	0.500000,
	0.750000,
	1.000000,
	0.750000,
	0.500000,
	0.250000,
	0.000000,
	-0.250000,
	-0.500000,
	-0.750000,
}

var sawtooth = []float64{
	-1.000000,
	-0.875000,
	-0.750000,
	-0.625000,
	-0.500000,
	-0.375000,
	-0.250000,
	-0.125000,
	0.000000,
	0.125000,
	0.250000,
	0.375000,
	0.500000,
	0.625000,
	0.750000,
	0.875000,
}
