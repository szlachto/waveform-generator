package main

import (
	"bufio"
	"container/list"
	"encoding/json"
	"log"
	"math"
	"net"
	"os"
	"sync"
	"time"
)

const Period = 1 * time.Second
const MaxValue = 30
const Step = math.Pi / 24

var mu sync.Mutex
var subs = list.New()

func main() {
	port := os.Getenv("GENERATOR_PORT")
	if port == "" {
		port = "3000"
	}
	ln, err := net.Listen("tcp", net.JoinHostPort("", port))
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Listen on %s", ln.Addr())
	defer ln.Close()
	go ticker()
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("Connection from %s", conn.RemoteAddr())
		addSubscriber(conn)
	}
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

func ticker() {
	ticker := time.NewTicker(Period)
	defer ticker.Stop()

	x := 0.0
	for {
		select {
		case t := <-ticker.C:
			v := MaxValue * math.Sin(x)
			log.Printf("Emitting value: %f", v)
			s := &Sample{
				Timestamp: t.Unix(),
				Value:     v,
			}
			updateSubscribers(s)
			x = x + Step
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
