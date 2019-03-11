package main

import (
	"encoding/binary"
	"io"
	"log"
	"strconv"
	"time"

	"github.com/j-forster/Wazihub-API"
	"github.com/tarm/serial"
	"periph.io/x/periph/conn/gpio"
	"periph.io/x/periph/conn/gpio/gpioreg"
	"periph.io/x/periph/host"
	_ "periph.io/x/periph/host/rpi"
)

const BoundsXYZ = 100000

//var liveness chan struct{}

var ports [4]*serial.Port

func main() {
	host.Init()
	log.SetFlags(0) // remove timestamps from log
	log.Println("MEKOB-P")

	//////////

	go httpServer()
	go GPS()

	//liveness = make(chan struct{})
	//go livenessStatus()

	//////////

	// The ID of the device this program is running on.
	// We use 'CurrentDeviceId' to create a unique Id for this device.
	log.Println("This device id is:", device.Id)

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
	log.Println("Device registered!")

	//////////

	go Zigpos()

	//////////

	gpio23 := gpioreg.ByName("23")
	gpioState := false
	isMeasuring := false
	ticker := time.NewTicker(500 * time.Millisecond)

	actuation, _ := wazihub.Actuation(device.Id, "event")
	for {
		select {
		case event := <-actuation:
			switch event {

			case "\"start\"":
				log.Println("Event: Start")

				ports[0], err = createPort(0)
				if err != nil {
					wazihub.PostValues(device.Id, "event", EventError{err.Error()})
					log.Println("Port 0 error:", err)
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
				isMeasuring = true
				log.Println("Total", n, "ports available.")
				wazihub.PostValues(device.Id, "event", EventStart{n})

			case "\"stop\"":
				log.Println("Event: Stop")
				for i, port := range ports {
					if port != nil {
						port.Close()
						log.Println("Port", i, "closed.")
						ports[i] = nil
					}
				}
				isMeasuring = false
				gpio23.Out(gpio.Level(false))

			default:
				log.Println("Unknown actuation:", event)
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

func readPort(n int) error {

	port := ports[n]
	line := make([]byte, 63)
	//port.Read(buf)
	//reader := bufio.NewReader(port)

	log.Println("Waiting for serial port", strconv.Itoa(n), "data ...")

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

	// sensor := "acc" + strconv.Itoa(n)
	sensorCondensed := ".acc" + strconv.Itoa(n)
	var accs [6]Acceleration
	var force Force

	l, err := port.Read(line)
	if l < 30 {
		// hot fix to align with packet boundaries
		port.Read(line)
	}

	for {
		l, err = io.ReadFull(port, line)
		//line, err := reader.ReadBytes('\n')
		if ports[n] != port {
			log.Println("serial port", n, "dropped.")
			return nil
		}
		if err != nil {
			log.Println("serial port", n, "read error:", err)
			return err
		}
		if l != 63 {
			avg.Dropps++
			log.Print("Dropped:", l)
			continue
		}

		//if len(line) != 63 {
		// log.Println("Invalid line. ", line)
		//	avg.Dropps++
		//	continue
		//}

		// n := int(line[9]-'0')*100 + int(line[10]-'0')*10 + int(line[11]-'0')

		avg.Packets++

		if len(line) < 13 {
			avg.Dropps++
			continue
		}

		if line[13] == 'A' {

			if len(line) < 50 {
				avg.Dropps++
				continue
			}

			o := 14
			for i := 0; i < 6; i++ {
				accs[i] = readAcceleration(line[o:])
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

			// wazihub.PostValues(device.Id, sensor, accs)
		} else if line[13] == 'P' {

			if len(line) < 38 {
				avg.Dropps++
				continue
			}

			force = readForce(line[14:])
			avg.PValues++
			for i := 0; i < 6; i++ {
				avg.Force[i] += force[i]
			}
		} else if line[13] == 'p' {

			if len(line) < 38 {
				avg.Dropps++
				continue
			}

			force = readForce(line[14:])
			avg.P2Values++
			for i := 0; i < 6; i++ {
				avg.Force2[i] += force[i]
			}
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
