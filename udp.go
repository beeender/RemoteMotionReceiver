package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
)

func handleUDPConnection(conn *net.UDPConn) {

	// here is where you want to do stuff like read or write to client

	buffer := make([]byte, 1024)

	n, addr, err := conn.ReadFromUDP(buffer)

	fmt.Println("UDP client : ", addr)
	fmt.Println("Received from UDP client :  ", string(buffer[:n]))

	if err != nil {
		log.Fatal(err)
	}

	// NOTE : Need to specify client address in WriteToUDP() function
	//        otherwise, you will get this error message
	//        write udp : write: destination address required if you use Write() function instead of WriteToUDP()

	// write message back to client
	message := []byte("Hello UDP client!")
	_, err = conn.WriteToUDP(message, addr)

	if err != nil {
		log.Println(err)
	}

}

func main1() {
	const RMOTION_PATH = "/tmp/rmotion"
	hostName := "127.0.0.1"
	portNum := "6000"
	service := hostName + ":" + portNum

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
	_, err = f.WriteString(hostName + " : " + portNum)
	if err != nil {
		log.Fatal(err)
	}
	_ = f.Close()

	udpAddr, err := net.ResolveUDPAddr("udp4", service)

	if err != nil {
		log.Fatal(err)
	}

	// setup listener for incoming UDP connection
	ln, err := net.ListenUDP("udp", udpAddr)

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("UDP server up and listening on port 6000")

	defer ln.Close()

	for {
		// wait for UDP client to connect
		handleUDPConnection(ln)
	}
}
