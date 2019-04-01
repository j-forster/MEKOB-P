package main

import (
	"encoding/binary"
	"errors"
	"io"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/j-forster/Wazihub-API"
	"github.com/tarm/serial"
	"periph.io/x/periph/conn/gpio"
	"periph.io/x/periph/conn/gpio/gpioreg"
	"periph.io/x/periph/host"
	_ "periph.io/x/periph/host/rpi"
)

const BoundsXYZ = 100000

const PktLength = 34
const PosPktIdentifier = 7

var codeStart = []byte("W\r")

var codeStop = []byte("w\r")
var codeReset = []byte("R\r")
var codeSleep = []byte("S\r")
var codeSync = []byte("CTc\r")

//var liveness chan struct{}

var ports [4]*serial.Port

func main() {
	host.Init()
	log.SetFlags(0) // remove timestamps from log
	log.Println("[      ] MEKOB-P")

	//////////

	go httpServer()
	go GPS()

	//liveness = make(chan struct{})
	//go livenessStatus()

	//////////

	// The ID of the device this program is running on.
	// We use 'CurrentDeviceId' to create a unique Id for this device.
	log.Println("[      ] This device id is:", device.Id)

	//////////

	// Login to Wazihub
	if err := wazihub.Login("cdupont", "password"); err != nil {
		Fatal("Login failed!", err)
	}

	//////////

	// We register the device, even though it might already be registered ...
	err := wazihub.CreateDevice(device)
	if err != nil {
		Fatal("Failed to register!", err)
	}
	log.Println("[      ] Device registered!")
	log.Println("[      ] ---------------------------")

	//////////

	go Zigpos()

	//////////

	gpio23 := gpioreg.ByName("23")
	gpioState := false
	isMeasuring := false
	ticker := time.NewTicker(500 * time.Millisecond)

	ports[0], err = createPort(0)
	if err != nil {
		wazihub.PostValues(device.Id, "event", EventError{err.Error()})
		log.Println("[UART  ] Port 0 error:", err)
		return
	}
	n := 1
	go readPort(0)
	for i := 1; i < 4; i++ {
		ports[i], _ = createPort(i)
		if ports[i] != nil {
			go readPort(i)
			n++
		}
	}
	log.Println("[UART  ] Total", n, "ports available.")

	actuation, _ := wazihub.Actuation(device.Id, "event")
	for {
		select {
		case event := <-actuation:
			switch event {

			case "\"start\"":
				log.Println("[Event ] !! Start !!")

				for i, port := range ports {
					if port != nil {
						port.Write([]byte(codeStart))
						log.Println("[UART  ] Port", i, "send:", strings.TrimSpace(string(codeStart)))
						log.Println("[UART  ] Port", i, "started.")
					}
				}

				isMeasuring = true
				wazihub.PostValues(device.Id, "event", EventStart{n})

			case "\"stop\"":
				log.Println("[Event ] !! Stop !!")
				for i, port := range ports {
					if port != nil {
						port.Write([]byte(codeStop))
						log.Println("[UART  ] Port", i, "send:", strings.TrimSpace(string(codeStop)))
						log.Println("[UART  ] Port", i, "stopped.")
					}
				}
				isMeasuring = false
				gpio23.Out(gpio.Level(false))

			case "\"reset\"":
				log.Println("[Event ] !! Reset !!")
				for i, port := range ports {
					if port != nil {
						port.Write([]byte(codeReset))
						log.Println("[UART  ] Port", i, "send:", strings.TrimSpace(string(codeReset)))
						log.Println("[UART  ] Port", i, "resetted.")
					}
				}
				isMeasuring = false
				gpio23.Out(gpio.Level(false))

			case "\"sleep\"":
				log.Println("[Event ] !! Sleep !!")
				for i, port := range ports {
					if port != nil {
						port.Write([]byte(codeSleep))
						log.Println("[UART  ] Port", i, "send:", strings.TrimSpace(string(codeSleep)))
						log.Println("[UART  ] Port", i, "sleeping.")
					}
				}
				isMeasuring = false
				gpio23.Out(gpio.Level(false))

			case "\"sync\"":
				log.Println("[Event ] !! Sync !!")
				for i, port := range ports {
					if port != nil {
						port.Write([]byte(codeSync))
						log.Println("[UART  ] Port", i, "send:", strings.TrimSpace(string(codeSync)))
						log.Println("[UART  ] Port", i, "synced.")
					}
				}
				isMeasuring = false
				gpio23.Out(gpio.Level(false))

			default:
				log.Println("[Event ] Unknown actuation:", event)
				wazihub.PostValues(device.Id, "event", EventError{"Unknown Event"})
			}

		case <-ticker.C:

			if isMeasuring {
				gpioState = !gpioState
				gpio23.Out(gpio.Level(gpioState))
			}
		}
	}
}

func createPort(n int) (*serial.Port, error) {
	config := &serial.Config{
		Name: "/dev/ttyUSB" + strconv.Itoa(n),
		Baud: 921600,
	}
	port, err := serial.OpenPort(config)
	if err != nil {
		return nil, err
	}
	return port, nil
}

var errNoPattern = errors.New("Pattern not found.")

func syncPort(n int) error {
	o := 0
	port := ports[n]
	line := make([]byte, PktLength)
	pattern := make([]byte, 4)
	for pattern[0] != 13 && pattern[1] != 10 && pattern[2] != 0xca && pattern[3] != 0xfe {
		pattern[0] = pattern[1]
		pattern[1] = pattern[2]
		pattern[2] = pattern[3]
		_, err := port.Read(pattern[3:])
		if err != nil {
			return err
		}
		o++
		if o > 2*PktLength {
			return errNoPattern
		}
	}
	port.Read(line[2:])
	return nil
}

func readPort(n int) error {

	port := ports[n]
	line := make([]byte, PktLength)
	//port.Read(buf)
	//reader := bufio.NewReader(port)

	port.Write([]byte(codeReset))
	log.Println("[UART  ] Port", n, "send:", strings.TrimSpace(string(codeReset)))
	log.Println("[UART  ] Port", n, "resetted.")

	log.Println("[UART  ] Port", n, "started ...")
	log.Println("[UART  ] Waiting for serial port", n, "data ...")
	var err error

	for i := 0; i < 3; i++ {
		err = syncPort(n)
		if err != nil {
			return err
		}
	}

	log.Println("[UART  ] Port", n, "synced ...")

	stepTime := time.Now()
	//begin := time.Now()
	//var counter int64
	//var dropped int64
	var avg Avg
	avg.Min.X = BoundsXYZ
	avg.Min.Y = BoundsXYZ
	avg.Min.Z = BoundsXYZ
	avg.Max.X = -BoundsXYZ
	avg.Max.Y = -BoundsXYZ
	avg.Max.Z = -BoundsXYZ

	sensor := "acc" + strconv.Itoa(n)
	sensorCondensed := ".acc" + strconv.Itoa(n)
	var accs [6]Acceleration
	var force Force

	l, err := port.Read(line)
	if err != nil {
		return err
	}
	if l < 15 {
		// hot fix to align with packet boundaries
		port.Read(line)
	}

	for {
		l, err = io.ReadFull(port, line)
		//line, err := reader.ReadBytes('\n')
		if ports[n] != port {
			log.Println("[UART  ] Port", n, "dropped.")
			return nil
		}
		if err != nil {
			log.Println("[UART  ] Port", n, "read error:", err)
			return err
		}
		if l != PktLength {
			avg.Dropps++
			log.Print("[UART  ] Port", n, "dropped (length):", l)
			continue
		}
		if line[0] != 0xca || line[1] != 0xfe {
			avg.Dropps++
			log.Print("[UART  ] Port", n, "dropped (0xcafe) [", l, "] ", line)
			continue
		}

		//log.Print("Port ", n, " ok", line)

		//if len(line) != 63 {
		// log.Println("Invalid line. ", line)
		//	avg.Dropps++
		//	continue
		//}

		// n := int(line[9]-'0')*100 + int(line[10]-'0')*10 + int(line[11]-'0')

		avg.Packets++
		switch line[PosPktIdentifier] {
		case 'A':
			o := PosPktIdentifier + 1
			for i := 0; i < 2; i++ {
				accs[i] = readAcceleration(line[o:]) // 6
				o += 6
				if accs[i].X == 0 && accs[i].Y == 0 && accs[i].Z == 0 {
					continue
				}
				avg.AValues++
				avg.Mean.X += accs[i].X
				avg.Mean.Y += accs[i].Y
				avg.Mean.Z += accs[i].Z
				avg.Min.X = min(avg.Min.X, accs[i].X)
				avg.Min.Y = min(avg.Min.Y, accs[i].Y)
				avg.Min.Z = min(avg.Min.Z, accs[i].Z)
				avg.Max.X = max(avg.Max.X, accs[i].X)
				avg.Max.Y = max(avg.Max.Y, accs[i].Y)
				avg.Max.Z = max(avg.Max.Z, accs[i].Z)
			}
			//log.Print("Port ", n, " accs ", accs)
			wazihub.PostValues(device.Id, sensor, accs)

		case 'P':

			o := PosPktIdentifier + 1
			force = readForce(line[o:]) // 24B
			avg.PValues++
			for i := 0; i < 6; i++ {
				avg.Force[i] += force[i]
			}

		case 'p':

			o := PosPktIdentifier + 1
			force = readForce(line[o:]) // 24B
			avg.P2Values++
			for i := 0; i < 6; i++ {
				avg.Force2[i] += force[i]
			}

		case 'i':
			o := PosPktIdentifier + 1
			str := strings.TrimSpace(string(line[o:]))
			log.Println("[UART  ] Port", n, "Info:", str)
			wazihub.PostValues(device.Id, "info", str)
		}

		if time.Since(stepTime) > time.Second/5 {

			if avg.AValues != 0 {
				avg.Mean.X /= float32(avg.AValues)
				avg.Mean.Y /= float32(avg.AValues)
				avg.Mean.Z /= float32(avg.AValues)
			}

			if avg.PValues != 0 {
				for i := 0; i < 6; i++ {
					avg.Force[i] /= avg.PValues
				}
			}
			if avg.P2Values != 0 {
				for i := 0; i < 6; i++ {
					avg.Force2[i] /= avg.P2Values
				}
			}

			wazihub.PostValue(device.Id, sensorCondensed, &avg)
			log.Println("[UART  ] Port", n, "Data: A", avg.AValues, "P", avg.PValues, "P2", avg.P2Values, "X", avg.Dropps)
			//liveness <- struct{}{}
			avg = Avg{}
			avg.Min.X = BoundsXYZ
			avg.Min.Y = BoundsXYZ
			avg.Min.Z = BoundsXYZ
			avg.Max.X = -BoundsXYZ
			avg.Max.Y = -BoundsXYZ
			avg.Max.Z = -BoundsXYZ
			stepTime = time.Now()
		}
		//avg.Mean.X

		//

		// log.Println(strconv.Itoa(n), strconv.Itoa(x), strconv.Itoa(y), strconv.Itoa(z))
	}
}

func readForce(data []byte) Force {
	var f Force
	f[0] = int32(binary.LittleEndian.Uint32(data))
	f[1] = int32(binary.LittleEndian.Uint32(data[4:]))
	f[2] = int32(binary.LittleEndian.Uint32(data[8:]))
	f[3] = int32(binary.LittleEndian.Uint32(data[12:]))
	f[4] = int32(binary.LittleEndian.Uint32(data[16:]))
	f[5] = int32(binary.LittleEndian.Uint32(data[20:]))
	return f
}

func readAcceleration(data []byte) Acceleration {
	x := int(data[1])*256 + int(data[0])
	if x > 32767 {
		x = -(65536 - x)
	}
	y := int(data[3])*256 + int(data[2])
	if y > 32767 {
		y = -(65536 - y)
	}
	z := int(data[5])*256 + int(data[4])
	if z > 32767 {
		z = -(65536 - z)
	}
	return Acceleration{
		X: float32(x) * 49 / 1000,
		Y: float32(y) * 49 / 1000,
		Z: float32(z) * 49 / 1000,
	}
}

/*
func livenessStatus() {
	gpio23 := gpioreg.ByName("23")
	state := true
	for range liveness {
		gpio23.Out(gpio.Level(state))
		state = !state
	}
}
*/

func Fatal(v ...interface{}) {

	gpio18 := gpioreg.ByName("18")

	for i := 0; i < 3; i++ {
		gpio18.Out(gpio.Level(true))
		time.Sleep(time.Second / 2)
		gpio18.Out(gpio.Level(false))
		time.Sleep(time.Second / 2)
	}

	log.Fatalln(v...)
}

func min(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}
func max(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}
