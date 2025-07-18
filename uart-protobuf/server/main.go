package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"time"

	"github.com/tarm/serial"
	"google.golang.org/protobuf/proto"
	whisperpb "pico-serial-protobuf/proto"
)

func main() {
	// Update the serial port name for your system
	// Linux/macOS: "/dev/ttyACM0", "/dev/ttyUSB0"
	// Windows: "COM3", "COM5", etc.
	portName := "/dev/cu.usbmodem11101"

	config := &serial.Config{
		Name:        portName,
		Baud:        115200,
		ReadTimeout: time.Second * 2,
	}
	port, err := serial.OpenPort(config)
	if err != nil {
		log.Fatalf("failed to open serial port: %v", err)
	}
	defer port.Close()

	go func() {
		for {
			buf := make([]byte, 10)
			port.Read(buf)
			fmt.Println(string(buf))
		}

	}()

	for {
		text := fmt.Sprintf("Hello from pc via serial %s", time.Now().Format(time.TimeOnly))

		err = sendDisplayMessage(port, text)
		if err != nil {
			log.Printf("failed to send message: %v", err)
		}
		time.Sleep(10 * time.Second)
	}
}

func sendDisplayMessage(writer *serial.Port, text string) error {
	msg := &whisperpb.ServerEvent{
		Event: &whisperpb.ServerEvent_DisplayEvent{
			DisplayEvent: &whisperpb.DisplayEvent{
				Text: text,
			},
		},
	}

	data, err := proto.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal proto: %w", err)
	}

	// Write length prefix
	length := uint32(len(data))
	err = binary.Write(writer, binary.LittleEndian, length)
	if err != nil {
		return fmt.Errorf("failed to write length: %w", err)
	}

	// Write the actual protobuf message
	_, err = writer.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	fmt.Printf("Sent message: %s\n", text)
	return nil
}
