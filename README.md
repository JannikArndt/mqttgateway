# MQTT to Prometheus (Tasmota Style)

A project that subscribes to MQTT queues and published prometheus metrics.

Forked from https://github.com/inuits/mqttgateway

Adapted for devices that run https://github.com/arendst/Tasmota

## Installation

Requires go > 1.9

```
go get -u github.com/JannikArndt/mqtttoprom
```

Build for Raspberry Pi:

```
GOOS=linux GOARCH=arm GOARM=5 go build
```

You probably want the service to run around the clock. `systemd` can help you. 
You can find a service-definition in [mqtttoprom.service](mqtttoprom.service) and an
installation script that creates a dedicated user and enables the service in [install_mqtttoprom.sh](install_mqtttoprom.sh).
Unfortunately, it needs to run as root, so you should take a look into what it does before executing it 😉

## How does it work?

mqttgateway will connect to the MQTT broker at `--mqtt.broker-address` and
listen to the topics specified by `--mqtt.topic`.

By default, it will listen to `#`.

It expects the topics to follow the Tasmota-scheme 

* `tele/<room>/SENSOR` for temperature and humidity data
* `tele/<room>/STATE` for power ON/OFF

and exports them as

```
power{room="<room>"} 0
temperature{room="<room>"} 20.5
humidity{room="<room>"} 63.9
```

on http://localhost:9337/metrics in a format that Prometheus can read. 