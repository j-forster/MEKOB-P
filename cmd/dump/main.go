package main

import (
	"bufio"
	"log"
	"time"

	"github.com/tarm/serial"
)

func main() {

	log.Println("Opening serial port ...")

	config := &serial.Config{
		Name: "/dev/ttyUSB0",
		Baud: 921600,
	}

	port, err := serial.OpenPort(config)
	if err != nil {
		log.Fatal("Serial port 0 error:", err)
	}

	reader := bufio.NewReader(port)

	log.Println("Waiting for serial port 0 data ...")

	var drops int64
	var jumps int64
	var packets int64
	var counter int64
	step := time.Now()

	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			log.Fatal("serial port 0 read error:", err)
		}
		packets++

		if len(line) != 63 {
			drops++
		} else {

			c := int64(line[11]-'0')*1000 + int64(line[10]-'0')*100 + int64(line[9]-'0')*10 + int64(line[8]-'0')
			if c != counter+1 {
				jumps++
				counter = c
			}
		}

		if time.Since(step) > time.Second {
			step = time.Now()
			log.Println("counter:", counter, "packets:", packets, "jumps:", jumps, "drops:", drops)
			packets = 0
			jumps = 0
			drops = 0
		}
	}
}
