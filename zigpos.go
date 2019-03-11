package main

import (
	"encoding/json"
	"log"
	"time"

	wazihub "github.com/j-forster/Wazihub-API"
	"github.com/j-forster/Wazihub-API/mqtt"
)

type zigposData []struct {
	Timestamp         string `json:"timestamp"`
	FilteredPositions []struct {
		Timestamp      string  `json:"timestamp"`
		AccuracyRadius float32 `json:"accuracyRadius"`
		X              float32 `json:"x"`
		Y              float32 `json:"y"`
		Z              float32 `json:"z"`
	} `json:"filteredPositions`
}

func Zigpos() {

	for {
		log.Println("[Zigpos] Connecting to mqtt://192.168.0.101 ...")
		client, err := mqtt.Dial("192.168.0.101:1883", "mekob", true, nil, nil)
		if err != nil {
			log.Println("[Zigpos] Error:", err)
			time.Sleep(time.Second * 5)
			continue
		}

		log.Println("[Zigpos] Connected.")
		client.Subscribe("zp/rnhd/v2/pan/23176/server/messages/COMPOSED_POSITIONS", 0x00)

		var data zigposData
		for msg := range client.Message() {
			err := json.Unmarshal(msg.Data, &data)
			if err != nil {
				log.Println("[Zigpos] Unmarshal Error:", err)
				continue
			}
			if len(data) != 0 && len(data[0].FilteredPositions) != 0 {
				pos := data[0].FilteredPositions[0]
				log.Println("[Zigpos] Data:", &pos)
				wazihub.PostValue(device.Id, "position", &pos)
			}
		}

		log.Println("[Zigpos] Closed:", client.Error)
		time.Sleep(time.Second * 2)
	}
}
