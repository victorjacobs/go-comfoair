package main

import (
	"log"
	"net/http"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/julienschmidt/httprouter"
	"github.com/victorjacobs/go-comfoair/bridge"
	"github.com/victorjacobs/go-comfoair/config"
	"github.com/victorjacobs/go-comfoair/routes"
)

// TODO do availability
func main() {
	cfg, err := config.LoadConfiguration("comfoair.json")
	if err != nil {
		log.Fatalf("Error loading configuration: %v", err)
		return
	}

	bridge, err := bridge.New(cfg)
	if err != nil {
		log.Fatalf("Error setting up bridge: %v", err)
		return
	}

	mqttOpts := cfg.Mqtt.ClientOptions()
	// Configure MQTT subscriptions in the ConnectHandler to make sure they are set up after reconnect
	mqttOpts.SetOnConnectHandler(func(client mqtt.Client) {
		bridge.SubscribeToFanCommands(client)
	})

	mqttClient := mqtt.NewClient(mqttOpts)
	if t := mqttClient.Connect(); t.Wait() && t.Error() != nil {
		log.Printf("MQTT connection error: %v", t.Error())
		return
	}

	// Fan
	bridge.RegisterFan(mqttClient)
	go loopSafely(func() {
		bridge.PollFanState(mqttClient)

		time.Sleep(1 * time.Second)
	})

	// Sensors
	bridge.RegisterSensors(mqttClient)
	go loopSafely(func() {
		bridge.PollSensors(mqttClient)

		time.Sleep(time.Minute)
	})

	// Start httprouter
	router := httprouter.New()
	router.GET("/state", routes.State(bridge))

	go loopSafely(func() {
		http.ListenAndServe(":8080", router)
	})

	select {}
}
