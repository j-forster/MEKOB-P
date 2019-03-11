package main

import (
	"bufio"
	"log"
	"strconv"
	"strings"

	wazihub "github.com/j-forster/Wazihub-API"
	"github.com/tarm/serial"
)

type GPSPosition struct {
	Lat, Long float64
}

func GPS() {

	config := &serial.Config{
		Name: "/dev/ttyS0",
		Baud: 9600,
	}
	port, err := serial.OpenPort(config)
	if err != nil {
		log.Println("[GPS  ] Port Error:", err)
		return
	}

	reader := bufio.NewReader(port)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			log.Println("[GPS  ] Port Read Error:", err)
			return
		}
		if strings.HasPrefix(string(line), "$GPGGA") {
			split := strings.Split(string(line), ",")
			if len(split) == 15 && split[3] == "N" {
				var pos = GPSPosition{
					Lat:  pos(split[2]),
					Long: pos(split[4]),
				}
				wazihub.PostValue(device.Id, "gps", &pos)
				// log.Printf("[GPS  ] Lat %.6f / Long %.6f\n", pos.Lat, pos.Long)
			}
		}
	}
}

func pos(s string) float64 {
	i := strings.IndexRune(s, '.')
	p, _ := strconv.ParseFloat(s[i-2:], 32)
	j, _ := strconv.ParseFloat(s[0:i-2], 32)
	if j > 0 {
		return j + p/60
	}
	return j - p/60
}
