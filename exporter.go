package main

import (
	"encoding/json"
	"strings"
	"sync"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
)

var mutex sync.RWMutex

type mqttExporter struct {
	client         mqtt.Client
	versionDesc    *prometheus.Desc
	connectDesc    *prometheus.Desc
	metrics        map[string]*prometheus.GaugeVec   // hold the metrics collected
	counterMetrics map[string]*prometheus.CounterVec // hold the metrics collected
	metricsLabels  map[string][]string               // holds the labels set for each metric to be able to invalidate them
}

func newMQTTExporter() *mqttExporter {
	// create a MQTT client
	options := mqtt.NewClientOptions()
	log.Infof("Connecting to %v", *brokerAddress)
	options.AddBroker(*brokerAddress)
	if *username != "" {
		options.SetUsername(*username)
	}
	if *password != "" {
		options.SetPassword(*password)
	}
	if *clientID != "" {
		options.SetClientID(*clientID)
	}
	m := mqtt.NewClient(options)
	if token := m.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal(token.Error())
	}

	// create an exporter
	c := &mqttExporter{
		client: m,
		versionDesc: prometheus.NewDesc(
			prometheus.BuildFQName(progname, "build", "info"),
			"Build info of this instance",
			nil,
			prometheus.Labels{"version": version}),
		connectDesc: prometheus.NewDesc(
			prometheus.BuildFQName(progname, "mqtt", "connected"),
			"Is the exporter connected to mqtt broker",
			nil,
			nil),
	}

	c.metrics = make(map[string]*prometheus.GaugeVec)
	c.counterMetrics = make(map[string]*prometheus.CounterVec)
	c.metricsLabels = make(map[string][]string)

	c.metrics["temperature"] = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "temperature",
			Help: "Temperature",
		},
		[]string{"room"},
	)

	c.metrics["humidity"] = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "humidity",
			Help: "Humidity",
		},
		[]string{"room"},
	)

	c.metrics["power"] = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "power",
			Help: "power",
		},
		[]string{"room"},
	)

	var topics = strings.Split(*topic, " ")

	for _, top := range topics {
		m.Subscribe(top, 2, c.receiveMessage())
	}

	return c
}

func (c *mqttExporter) Describe(ch chan<- *prometheus.Desc) {
	mutex.RLock()
	defer mutex.RUnlock()
	ch <- c.versionDesc
	ch <- c.connectDesc
	for _, m := range c.counterMetrics {
		m.Describe(ch)
	}
	for _, m := range c.metrics {
		m.Describe(ch)
	}
}

func (c *mqttExporter) Collect(ch chan<- prometheus.Metric) {
	mutex.RLock()
	defer mutex.RUnlock()
	ch <- prometheus.MustNewConstMetric(
		c.versionDesc,
		prometheus.GaugeValue,
		1,
	)
	connected := 0.
	if c.client.IsConnected() {
		connected = 1.
	}
	ch <- prometheus.MustNewConstMetric(
		c.connectDesc,
		prometheus.GaugeValue,
		connected,
	)
	for _, m := range c.counterMetrics {
		m.Collect(ch)
	}
	for _, m := range c.metrics {
		m.Collect(ch)
	}
}

// {"Time":"2019-11-22T17:03:05","SI7021":{"Temperature":20.6,"Humidity":56.7},"TempUnit":"C"}
type SensorMessage struct {
	Time     string
	SI7021   Reading
	TempUnit string
}

type Reading struct {
	Temperature float64
	Humidity    float64
}

type ResultMessage struct {
	POWER string
}

// {"Time":"2019-11-20T21:35:48","Uptime":"3T13:21:53","Heap":16,"SleepMode":"Dynamic","Sleep":50,"LoadAvg":19,"POWER1":"OFF",
// "Wifi":{"AP":1,"SSId":"xxx","BSSId":"xx:xx:xx:xx:xx:xx","Channel":11,"RSSI":100,"LinkCount":1,"Downtime":"0T00:00:04"}}
type StateMessage struct {
	Time      string
	Uptime    string
	Heap      string
	SleepMode string
	Sleep     int
	LoadAvg   int
	POWER1    string
}

func (e *mqttExporter) receiveMessage() func(mqtt.Client, mqtt.Message) {
	return func(c mqtt.Client, m mqtt.Message) {
		mutex.Lock()
		defer mutex.Unlock()

		labelValues := prometheus.Labels{}
		labelValues["room"] = strings.Split(m.Topic(), "/")[1]

		var messageType = strings.Split(m.Topic(), "/")[2]
		switch messageType {
		case "STATE":
			e.logStateMessage(m, labelValues)
		case "SENSOR":
			e.logSensorMessage(m, labelValues)
		case "RESULT":
			e.logResultMessage(m, labelValues)
		case "POWER":
			// e.logPowerMessage(m, labelValues)
		case "POWER1":
			e.logPowerMessage(m, labelValues)
		case "LWT":
		case "UPTIME":
		default:
			log.Warnf("Invalid topic: %s ends with unknown message type!", m.Topic())
		}
	}
}

func (e *mqttExporter) logStateMessage(m mqtt.Message, labelValues prometheus.Labels) {
	var message StateMessage
	_ = json.Unmarshal(m.Payload(), &message)
	e.metrics["power"].With(labelValues).Set(onOffToFloat(message.POWER1))
}

func (e *mqttExporter) logSensorMessage(m mqtt.Message, labelValues prometheus.Labels) {
	var message SensorMessage
	_ = json.Unmarshal(m.Payload(), &message)

	e.metrics["temperature"].With(labelValues).Set(message.SI7021.Temperature)
	e.metrics["humidity"].With(labelValues).Set(message.SI7021.Humidity)
}

func (e *mqttExporter) logPowerMessage(m mqtt.Message, labelValues prometheus.Labels) {
	var message = string(m.Payload())
	e.metrics["power"].With(labelValues).Set(onOffToFloat(message))
}

func (e *mqttExporter) logResultMessage(m mqtt.Message, labelValues prometheus.Labels) {
	var message ResultMessage
	_ = json.Unmarshal(m.Payload(), &message)
	e.metrics["power"].With(labelValues).Set(onOffToFloat(message.POWER))
}

func onOffToFloat(onOff string) float64 {
	var power = 0
	if onOff == "ON" {
		power = 1
	}
	return float64(power)
}
