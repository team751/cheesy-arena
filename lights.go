// Copyright 2014 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)
//
// Methods for controlling the field LED lighting.

package main

import (
	"log"
	"net"
	"time"
)

type LightPacket [32]byte

type Lights struct {
	connections    map[string]*net.Conn
	packets        map[string]*LightPacket
	oldPackets     map[string]*LightPacket
	hotGoal        string
	newConnections bool
	currentMode    string
	animationCount int
}

// Sets the color by name and transition time for the given LED channel.
func (lightPacket *LightPacket) setColorFade(channel int, color string, fade byte) {
	switch color {
	case "off":
		lightPacket.setRgbFade(channel, 0, 0, 0, fade)
	case "white":
		lightPacket.setRgbFade(channel, 15, 15, 15, fade)
	case "red":
		lightPacket.setRgbFade(channel, 15, 0, 0, fade)
	case "blue":
		lightPacket.setRgbFade(channel, 0, 0, 15, fade)
	case "green":
		lightPacket.setRgbFade(channel, 0, 15, 0, fade)
	case "yellow":
		lightPacket.setRgbFade(channel, 15, 11, 0, fade)
	case "darkred":
		lightPacket.setRgbFade(channel, 1, 0, 0, fade)
	case "darkblue":
		lightPacket.setRgbFade(channel, 0, 0, 1, fade)
	}
}

// Sets the color by name with instant transition for the given LED channel.
func (lightPacket *LightPacket) setColor(channel int, color string) {
	lightPacket.setColorFade(channel, color, 0)
}

// Sets the color by RGB values and transition time for the given LED channel.
func (lightPacket *LightPacket) setRgbFade(channel int, red byte, green byte, blue byte, fade byte) {
	lightPacket[channel*4] = red
	lightPacket[channel*4+1] = green
	lightPacket[channel*4+2] = blue
	lightPacket[channel*4+3] = fade
}

// Sets the color by name with instant transition for all LED channels.
func (lightPacket *LightPacket) setAllColor(color string) {
	lightPacket.setAllColorFade(color, 0)
}

// Sets the color by name and transition time for all LED channels.
func (lightPacket *LightPacket) setAllColorFade(color string, fade byte) {
	for i := 0; i < 8; i++ {
		lightPacket.setColorFade(i, color, fade)
	}
}

func (lights *Lights) Setup() error {
	lights.currentMode = "off"

	err := lights.SetupConnections()
	if err != nil {
		return err
	}

	lights.packets = make(map[string]*LightPacket)
	lights.packets["red"] = &LightPacket{}
	lights.packets["blue"] = &LightPacket{}
	lights.oldPackets = make(map[string]*LightPacket)
	lights.oldPackets["red"] = &LightPacket{}
	lights.oldPackets["blue"] = &LightPacket{}

	lights.sendLights()

	// Set up a goroutine to animate the lights when necessary.
	ticker := time.NewTicker(time.Millisecond * 50)
	go func() {
		for _ = range ticker.C {
			lights.animate()
		}
	}()
	return nil
}

func (lights *Lights) SetupConnections() error {
	lights.connections = make(map[string]*net.Conn)

	// Don't enable lights for a side if the controller address is not configured.
	if len(eventSettings.RedGoalLightsAddress) != 0 {
		conn, err := net.Dial("udp4", eventSettings.RedGoalLightsAddress)
		lights.connections["red"] = &conn
		if err != nil {
			return err
		}
	} else {
		lights.connections["red"] = nil
	}
	if len(eventSettings.BlueGoalLightsAddress) != 0 {
		conn, err := net.Dial("udp4", eventSettings.BlueGoalLightsAddress)
		lights.connections["blue"] = &conn
		if err != nil {
			return err
		}
	} else {
		lights.connections["blue"] = nil
	}
	lights.newConnections = true
	return nil
}

// Makes a goal for the given alliance hot.
func (lights *Lights) SetHotGoal(alliance string, leftSide bool) {
	if leftSide {
		lights.packets[alliance].setColor(0, "off")
		lights.packets[alliance].setColor(1, "off")
		lights.packets[alliance].setColor(2, "off")
		lights.packets[alliance].setColor(3, "yellow")
		lights.packets[alliance].setColor(4, "yellow")
		lights.packets[alliance].setColor(5, "yellow")
		lights.hotGoal = "left"
	} else {
		lights.packets[alliance].setColor(0, "yellow")
		lights.packets[alliance].setColor(1, "yellow")
		lights.packets[alliance].setColor(2, "yellow")
		lights.packets[alliance].setColor(3, "off")
		lights.packets[alliance].setColor(4, "off")
		lights.packets[alliance].setColor(5, "off")
		lights.hotGoal = "right"
	}
	lights.sendLights()
}

// Lights up the given alliance's goal for the given number of assists.
func (lights *Lights) SetAssistGoal(alliance string, numAssists int) {
	lights.packets[alliance].setColor(0, "off")
	lights.packets[alliance].setColor(1, "off")
	lights.packets[alliance].setColor(2, "off")
	lights.packets[alliance].setColor(3, "off")
	lights.packets[alliance].setColor(4, "off")
	lights.packets[alliance].setColor(5, "off")
	if numAssists > 0 {
		lights.packets[alliance].setColor(2, alliance)
		lights.packets[alliance].setColor(3, alliance)
	}
	if numAssists > 1 {
		lights.packets[alliance].setColor(1, alliance)
		lights.packets[alliance].setColor(4, alliance)
	}
	if numAssists > 2 {
		lights.packets[alliance].setColor(0, alliance)
		lights.packets[alliance].setColor(5, alliance)
	}
	lights.hotGoal = ""
	lights.sendLights()
}

// Turns off all lights for the given alliance's goal.
func (lights *Lights) ClearGoal(alliance string) {
	lights.packets[alliance].setColorFade(0, "off", 10)
	lights.packets[alliance].setColorFade(1, "off", 10)
	lights.packets[alliance].setColorFade(2, "off", 10)
	lights.packets[alliance].setColorFade(3, "off", 10)
	lights.packets[alliance].setColorFade(4, "off", 10)
	lights.packets[alliance].setColorFade(5, "off", 10)
}

// Turns on the given alliance's pedestal.
func (lights *Lights) SetPedestal(alliance string) {
	if alliance == "red" {
		lights.packets["blue"].setColor(6, alliance)
	} else {
		lights.packets["red"].setColor(6, alliance)
	}
	lights.sendLights()
}

// Turns off the given alliance's pedestal.
func (lights *Lights) ClearPedestal(alliance string) {
	if alliance == "red" {
		lights.packets["blue"].setColorFade(6, "off", 10)
	} else {
		lights.packets["red"].setColorFade(6, "off", 10)
	}
	lights.sendLights()
}

// Turns all lights green to signal that the field is safe to enter.
func (lights *Lights) SetFieldReset() {
	lights.packets["red"].setAllColor("green")
	lights.packets["blue"].setAllColor("green")
	lights.sendLights()
}

// Sets the lights to the given non-match mode for show or testing.
func (lights *Lights) SetMode(mode string) {
	lights.currentMode = mode
	lights.animationCount = 0

	switch mode {
	case "off":
		lights.packets["red"].setAllColor("off")
		lights.packets["blue"].setAllColor("off")
	case "all_white":
		lights.packets["red"].setAllColor("white")
		lights.packets["blue"].setAllColor("white")
	case "all_red":
		lights.packets["red"].setAllColor("red")
		lights.packets["blue"].setAllColor("red")
	case "all_green":
		lights.packets["red"].setAllColor("green")
		lights.packets["blue"].setAllColor("green")
	case "all_blue":
		lights.packets["red"].setAllColor("blue")
		lights.packets["blue"].setAllColor("blue")
	}
	lights.sendLights()
}

// Sends a control packet to the LED controllers only if their state needs to be updated.
func (lights *Lights) sendLights() {
	for alliance, connection := range lights.connections {
		if lights.newConnections || *lights.packets[alliance] != *lights.oldPackets[alliance] {
			if connection != nil {
				_, err := (*connection).Write(lights.packets[alliance][:])
				if err != nil {
					log.Printf("Failed to send %s light packet.", alliance)
				}
			}
			mainArena.hotGoalLightNotifier.Notify(lights.hotGoal)
		}
		*lights.oldPackets[alliance] = *lights.packets[alliance]
	}
	lights.newConnections = false
}

// State machine for controlling light sequences in the non-match modes.
func (lights *Lights) animate() {
	lights.animationCount += 1

	switch lights.currentMode {
	case "strobe":
		switch lights.animationCount {
		case 1:
			lights.packets["red"].setAllColor("white")
			lights.packets["blue"].setAllColor("off")
		case 2:
			lights.packets["red"].setAllColor("off")
			lights.packets["blue"].setAllColor("white")
			fallthrough
		default:
			lights.animationCount = 0
		}
		lights.sendLights()
	case "fade_red":
		if lights.animationCount == 1 {
			lights.packets["red"].setAllColorFade("red", 18)
			lights.packets["blue"].setAllColorFade("red", 18)
		} else if lights.animationCount == 61 {
			lights.packets["red"].setAllColorFade("darkred", 18)
			lights.packets["blue"].setAllColorFade("darkred", 18)
		} else if lights.animationCount > 120 {
			lights.animationCount = 0
		}
		lights.sendLights()
	case "fade_blue":
		if lights.animationCount == 1 {
			lights.packets["red"].setAllColorFade("blue", 18)
			lights.packets["blue"].setAllColorFade("blue", 18)
		} else if lights.animationCount == 61 {
			lights.packets["red"].setAllColorFade("darkblue", 18)
			lights.packets["blue"].setAllColorFade("darkblue", 18)
		} else if lights.animationCount > 120 {
			lights.animationCount = 0
		}
		lights.sendLights()
	case "fade_red_blue":
		if lights.animationCount == 1 {
			lights.packets["red"].setAllColorFade("blue", 18)
			lights.packets["blue"].setAllColorFade("darkred", 18)
		} else if lights.animationCount == 61 {
			lights.packets["red"].setAllColorFade("darkblue", 18)
			lights.packets["blue"].setAllColorFade("red", 18)
		} else if lights.animationCount > 120 {
			lights.animationCount = 0
		}
		lights.sendLights()
	}
}
