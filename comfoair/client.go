package comfoair

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"go.bug.st/serial"
)

// TODO preset enums?

type Client struct {
	serialPort        string
	mutex             sync.Mutex
	previousFanPreset string
}

func NewClient(serialPort string) (*Client, error) {
	return &Client{
		serialPort: serialPort,
	}, nil
}

func (c *Client) GetDeviceInfo() (*DeviceInfo, error) {
	if response, err := c.write([]byte{0x00, 0x69}, []byte{}); err != nil {
		return nil, err
	} else {
		return &DeviceInfo{
			MinorVersion: int(response[1]),
			MajorVersion: int(response[0]),
			DeviceName:   string(response[3:]),
		}, nil
	}
}

func (c *Client) GetStatus() (*Status, error) {
	if temp, err := c.GetTemperatureStatus(); err != nil {
		return nil, err
	} else if fan, err := c.GetFanStatus(); err != nil {
		return nil, err
	} else if valve, err := c.GetValveStatus(); err != nil {
		return nil, err
	} else {
		return &Status{
			Temperature: temp,
			Fan:         fan,
			Valve:       valve,
		}, nil
	}
}

func (c *Client) GetFanStatus() (*FanStatus, error) {
	if response, err := c.write([]byte{0x00, 0x0b}, []byte{}); err != nil {
		return nil, err
	} else {
		return &FanStatus{
			Preset:       parseFanPreset(int(response[0])),
			Supply:       int(response[0]),
			Exhaust:      int(response[1]),
			SupplySpeed:  parseFanSpeed(response[2:4]),
			ExhaustSpeed: parseFanSpeed(response[4:6]),
		}, nil
	}
}

func parseFanSpeed(speed []byte) int {
	return int(1875000.0 / float32(binary.BigEndian.Uint16(speed)))
}

func parseFanPreset(speed int) string {
	var preset string
	if speed == 15 {
		preset = "off"
	} else if speed == 35 {
		preset = "low"
	} else if speed == 50 {
		preset = "mid"
	} else if speed == 70 {
		preset = "high"
	} else {
		log.Printf("Unexpected fan speed for preset conversion: %v", speed)
		preset = "low"
	}

	return preset
}

func (c *Client) GetValveStatus() (*ValveStatus, error) {
	if response, err := c.write([]byte{0x00, 0x0d}, []byte{}); err != nil {
		return nil, err
	} else {
		var bypass int
		// Value 0xff is undefined, so filter it out
		if response[0] != 0xff {
			bypass = int(response[0])
		}

		var preHeating bool
		// 0x02 is undefined, so assume it is closed
		if int(response[1]) == 1 {
			preHeating = true
		}

		return &ValveStatus{
			Bypass:                 bypass,
			PreHeating:             preHeating,
			BypassMotorCurrent:     int(response[2]),
			PreheatingMotorCurrent: int(response[3]),
		}, nil
	}
}

func (c *Client) GetTemperatureStatus() (*TemperatureStatus, error) {
	if response, err := c.write([]byte{0x00, 0xd1}, []byte{}); err != nil {
		return nil, err
	} else {
		return &TemperatureStatus{
			Comfort: parseTemperature(response[0]),
			Outside: parseTemperature(response[1]),
			Supply:  parseTemperature(response[2]),
			Exhaust: parseTemperature(response[3]),
			Return:  parseTemperature(response[4]),
		}, nil
	}
}

func parseTemperature(temp byte) float32 {
	return float32(temp)/2.0 - 20
}

func (c *Client) GetOperatingTime() (*OperatingTime, error) {
	if response, err := c.write([]byte{0x00, 0xdd}, []byte{}); err != nil {
		return nil, err
	} else {
		return &OperatingTime{
			LowHours:    convertThreeBytesToInteger(response[3:6]),
			MediumHours: convertThreeBytesToInteger(response[6:9]),
			HighHours:   convertThreeBytesToInteger(response[17:20]),
			FilterHours: int(binary.BigEndian.Uint16(response[15:17])),
		}, nil
	}
}

func (c *Client) SetFanPreset(preset string) error {
	var fanSpeed int
	if preset == "off" {
		fanSpeed = 1
	} else if preset == "low" || preset == "" {
		fanSpeed = 2
	} else if preset == "mid" {
		fanSpeed = 3
	} else if preset == "high" {
		fanSpeed = 4
	} else {
		return fmt.Errorf("received unexpected preset: %v", preset)
	}

	log.Printf("Setting fan speed to %v", preset)

	if err := c.setFanSpeed(fanSpeed); err != nil {
		return fmt.Errorf("error setting fan speed: %v", err)
	}

	return nil
}

func (c *Client) ToggleFan(toggle bool) error {
	log.Printf("Toggling fan %v", toggle)

	if toggle {
		return c.SetFanPreset(c.previousFanPreset)
	} else {
		var currentPreset string

		if currentFanSpeed, err := c.GetFanStatus(); err != nil {
			currentPreset = "low"
		} else {
			currentPreset = parseFanPreset(currentFanSpeed.Supply)
		}

		if currentPreset != "off" {
			c.previousFanPreset = currentPreset
		}

		return c.setFanSpeed(1)
	}
}

func (c *Client) setFanSpeed(speed int) error {
	if speed < 0 || speed > 4 {
		return fmt.Errorf("invalid fan speed, tried to set %v", speed)
	}

	_, err := c.write([]byte{0x00, 0x99}, []byte{byte(speed)})

	if err != nil {
		return err
	}

	return nil
}

func convertThreeBytesToInteger(data []byte) int {
	arr := [4]byte{}
	copy(arr[1:], data)
	return int(binary.BigEndian.Uint32(arr[:]))
}

func (c *Client) write(cmd []byte, data []byte) ([]byte, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	port, err := serial.Open(c.serialPort, &serial.Mode{})
	if err != nil {
		return nil, err
	}
	defer port.Close()

	packed, _ := packWrite(cmd, data)

	n, err := port.Write(packed)

	if n == 0 {
		return nil, errors.New("nothing written")
	}

	if err != nil {
		return nil, err
	}

	time.Sleep(100 * time.Millisecond)

	buff := make([]byte, 250)
	n, err = port.Read(buff)

	if err != nil {
		return nil, err
	}

	if n == 0 {
		return nil, errors.New("no response")
	}

	if !bytes.Equal(buff[0:2], []byte{0x07, 0xf3}) {
		return nil, fmt.Errorf("didn't receive ACK, received %x instead", buff[0:2])
	}

	// If the response is exactly 2 bytes, no response command was sent
	if n == 2 {
		return []byte{}, nil
	}

	expectedResponseCommand := []byte{0x00, cmd[1] + 1}

	if !bytes.Equal(buff[4:6], expectedResponseCommand) {
		return nil, fmt.Errorf("unexpected response command. Expected %v, got %v", expectedResponseCommand, buff[4:6])
	}

	dataLength := int(buff[6])

	return buff[7 : 7+dataLength], nil
}

func packWrite(cmd []byte, data []byte) ([]byte, error) {
	header := []byte{0x07, 0xf0}
	trailer := []byte{0x07, 0x0f}

	dataLength := byte(len(data))

	packedCmdAndData := append(cmd, dataLength)
	packedCmdAndData = append(packedCmdAndData, data...)

	checksum := 173
	sevenEncounteredInChecksum := false

	for _, byte := range packedCmdAndData {
		if byte == 0x07 && sevenEncounteredInChecksum {
			sevenEncounteredInChecksum = true
			continue
		}

		checksum += int(byte)
	}

	checksum &= 0xff

	packed := append(header, packedCmdAndData...)
	packed = append(packed, byte(checksum))
	packed = append(packed, trailer...)

	return packed, nil
}
