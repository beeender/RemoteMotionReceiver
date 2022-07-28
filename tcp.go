package main

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	const RMOTION_PATH = "/tmp/rmotion"
	const HOST = "127.0.0.1"
	const PORT = "6000"

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc,
		syscall.SIGINT,
		syscall.SIGTERM)
	go func() {
		<-sigc
		// ... do something ...
		_ = os.Remove(RMOTION_PATH)
	}()

	f, err := os.Create(RMOTION_PATH)
	if err != nil {
		log.Fatal(err)
	}
	_, err = f.WriteString(HOST + " : " + PORT)
	if err != nil {
		log.Fatal(err)
	}
	_ = f.Close()

	listen, err := net.Listen("tcp4", HOST+":"+PORT)
	if err != nil {
		log.Fatal(err)
	}
	// close listener
	defer listen.Close()
	for {
		conn, err := listen.Accept()
		if err != nil {
			log.Fatal(err)
		}
		go handleIncomingRequest(conn)
	}
}

type TupleBuf struct {
	buf    []byte
	pos    int
	length int
}

var intHeader = []byte{0x0e, 0x00, 0x00, 0x00, 0x0a, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x18, 0x00}
var eos = []byte{0x00, 0x00, 0x04, 0x00}

const EOS = 0xffffffff

func NewTupleBuf() *TupleBuf {
	l := 8192
	tb := TupleBuf{
		buf:    make([]byte, l),
		pos:    4,
		length: l,
	}
	return &tb
}

var count = 0

func (tb *TupleBuf) Append(v int) bool {
	if tb.pos+len(intHeader)+4 > tb.length {
		binary.LittleEndian.PutUint32(tb.buf[0:4], uint32(tb.pos))
		return true
	}

	copy(tb.buf[tb.pos:], intHeader)
	tb.pos += len(intHeader)
	binary.LittleEndian.PutUint32(tb.buf[tb.pos:tb.pos+4], uint32(v))
	tb.pos += 4
	count++
	log.Printf(fmt.Sprintf("%d", count))
	return false
}

func (tb *TupleBuf) AppendEos() bool {
	if tb.pos+4 > tb.length {
		binary.LittleEndian.PutUint32(tb.buf[0:4], uint32(tb.pos))
		return true
	}

	copy(tb.buf[tb.pos:], eos)
	tb.pos += len(eos)
	binary.LittleEndian.PutUint32(tb.buf[0:4], uint32(tb.pos))
	return false
}

func createReceiverChannel(context context.Context, addr string) (net.Conn, chan<- int, error) {
	conn, err := net.Dial("tcp", addr)
	reader := make(chan int)

	go func() {
		defer func(conn net.Conn) {
			err := conn.Close()
			if err != nil {
			}
		}(conn)

		tb := NewTupleBuf()
	out:
		for {
			select {
			case v := <-reader:
				if v == EOS {
					if tb.AppendEos() {
						_, err = conn.Write(tb.buf[:tb.pos])
						if err != nil {
							log.Printf(err.Error())
						}
						tb.AppendEos()
					}
					_, err = conn.Write(tb.buf[:tb.pos])
					if err != nil {
						log.Printf(err.Error())
					}
					break out
				} else if tb.Append(v) {
					// buffer full,send it
					_, err = conn.Write(tb.buf[:tb.pos])
					if err != nil {
						log.Printf(err.Error())
					}
					// Append again
					tb.Append(v)
				}
			case <-context.Done():
				break out
			}
		}
	}()

	return conn, reader, err
}

func readRMotionHeader(conn net.Conn) (string, error) {
	headerBuf := make([]byte, 128)

	n, err := conn.Read(headerBuf)
	if err != nil {
		return "", err
	}
	if n != 128 {
		return "", errors.New("the first packet should be 128 bytes long")
	}
	return string(headerBuf), nil
}

func forwardRegisterMessage(conn net.Conn, rev net.Conn) error {
	buf := make([]byte, 32)
	n, err := conn.Read(buf)
	if err != nil {
		return err
	}
	if n != 32 {
		return errors.New("the register packet should be 32 bytes long")
	}

	_, err = rev.Write(buf)
	if err != nil {
		return err
	}

	return nil
}

func countPrimeNumbers(num1, num2 int) int {
	count := 0
	if num2 < 2 {
		return count
	}
	for num1 <= num2 {
		isPrime := true
		for i := 2; i <= int(math.Sqrt(float64(num1))); i++ {
			if num1%i == 0 {
				isPrime = false
				break
			}
		}
		if isPrime {
			count++
		}
		num1++
	}
	return count
}

var countb = 0

func createCalculateChannel(ctx context.Context, revCh chan<- int) chan<- []byte {
	calcCh := make(chan []byte)
	go func() {
		defer close(calcCh)

		for {
			select {
			case buf := <-calcCh:
				pos := 4
				for {
					if pos == len(buf) {
						break
					}
					if pos+18 > len(buf) {
						revCh <- EOS
						break
					}
					countb++
					orgNum := int(binary.LittleEndian.Uint32((buf)[pos+14 : pos+18]))
					//newNum := countPrimeNumbers(0, orgNum)
					newNum := orgNum
					pos += 18
					log.Printf("countb %d %d", countb, newNum)
					revCh <- newNum
				}
			case <-ctx.Done():
				break
			}
		}
	}()
	return calcCh
}

func handleIncomingRequest(conn net.Conn) {
	ctx, _ := context.WithCancel(context.Background())

	header, err := readRMotionHeader(conn)
	if err != nil {
		log.Fatal(err)
	}

	revConn, receiverChan, err := createReceiverChannel(ctx, header)
	err = forwardRegisterMessage(conn, revConn)
	if err != nil {
		log.Fatal(err)
	}

	calCh := createCalculateChannel(ctx, receiverChan)
outer:
	for {
		sizeHeader := make([]byte, 4)
		sizeHeaderOff := 0

		for {
			n, err := conn.Read(sizeHeader[sizeHeaderOff:])
			if err == io.EOF {
				break outer
			}
			if err != nil {
				log.Fatal(err)
			}
			if n+sizeHeaderOff < 4 {
				sizeHeaderOff += n
				continue
			}
			break
		}

		bodySize := int(binary.LittleEndian.Uint32(sizeHeader))
		bodyBuf := make([]byte, bodySize)
		copy(bodyBuf, sizeHeader)
		bodyBufOff := 4
		for {
			n, err := conn.Read(bodyBuf[bodyBufOff:])
			if err == io.EOF {
				break outer
			}
			if err != nil {
				log.Fatal(err)
			}
			if bodySize > bodyBufOff+n {
				sizeHeaderOff += n
				continue
			}
			break
		}
		calCh <- bodyBuf
	}

	// close conn
	//cancel()
	//_ = conn.Close()
}
