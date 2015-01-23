package main

import (
	"io"
	"log"
	"net"
)

type packet struct {
	name   string
	bucket string
	value  float64
}

var packets = make(chan packet, 10000)

func listenTcp() {
	listener, err := net.Listen("tcp", *address)
	if err != nil {
		log.Fatal("unable to listen on tcp %s: %s", *address, err)
	}

	log.Printf("listening for events at tcp %s...\n", *address)

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}

		log.Printf("new connection from tcp %s", conn.RemoteAddr())

		go handleTcpConn(conn)
	}
}

func handleTcpConn(conn net.Conn) {
	defer conn.Close()

	msg := make([]byte, 0)
	tmp := make([]byte, 8192)

	for {
		n, err := conn.Read(tmp)
		if err != nil {
			if err == io.EOF {
				break
			}

			log.Printf("unable to read from tcp: %s\n", err)
			return
		}
		msg = append(msg, tmp[0:n]...)
	}

	if *debug {
		log.Printf("received %d bytes from tcp %s:\n%s\n", len(msg), conn.RemoteAddr(), string(msg))
	}

	handle(string(msg))
}

func listenUdp() {
	addr, err := net.ResolveUDPAddr("udp", *address)
	if err != nil {
		log.Fatal("unable to resolve service address: %s", err)
	}

	listener, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Fatal("unable to listen on udp %s: %s", *address, err)
	}
	defer listener.Close()

	log.Printf("listening for events at udp %s...\n", *address)

	msg := make([]byte, 512)

	for {
		n, _, err := listener.ReadFrom(msg)
		if err != nil {
			if err == io.EOF {
				continue
			}

			log.Printf("listener: unable to read: %s\n", err)
			continue
		}

		if *debug {
			log.Printf("received metric: %s\n", string(msg[0:n]))
		}

		handle(string(msg[0:n]))
	}
}

func handle(msg string) {
	for _, p := range parsePacket(msg) {
		if *debug {
			log.Printf("received packet: %+v\n", p)
		}
		packets <- p
	}
}
