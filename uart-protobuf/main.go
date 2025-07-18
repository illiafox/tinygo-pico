package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image/color"
	"io"
	"log"
	"machine"
	"machine/usb/cdc"
	whisperpb "pico-serial-protobuf/proto"
	"strings"
	"time"
	"tinygo.org/x/drivers"
	"tinygo.org/x/drivers/st7735"
	"tinygo.org/x/tinyfont"
	"tinygo.org/x/tinyfont/freemono"
)

const (
	sampleRate = 8000 // 8 kSa/s is enough for envelope
	bufSize    = 256  // ~32 ms of audio
)

func main() {
	err := machine.UART0.Configure(machine.UARTConfig{
		BaudRate: 115200,
	})
	if err != nil {
		log.Fatalf("failed to configure machine: %v", err)
	}

	textSend := make(chan string, 10)

	go startDisplay(textSend)

	machine.Serial.Configure(machine.UARTConfig{
		BaudRate: 115200,
	})

	err = readData(&serialReaderAdapter{machine.Serial}, textSend)
	if err != nil {
		log.Fatalf("failed to read data: %v", err)
	}
}

func readData(reader io.Reader, textSend chan<- string) error {
	var buf bytes.Buffer

	var message = new(whisperpb.ServerEvent)

	for {
		var length uint32
		err := binary.Read(reader, binary.LittleEndian, &length)
		if err == io.EOF {
			break // End of stream
		}
		if err == cdc.ErrBufferEmpty {
			time.Sleep(3 * time.Second)
			continue
		}

		if err != nil {
			fmt.Printf("failed to read length: %w\n", err)
			continue
		}

		buf.Reset()
		n, err := io.CopyN(&buf, reader, int64(length))
		if err != nil {
			return fmt.Errorf("failed to read message: %w", err)
		}
		if n != int64(length) {
			log.Printf("incomplete message: expected %d bytes, got %d\n", length, n)
			continue
		}

		err = message.UnmarshalVT(buf.Bytes())
		if err != nil {
			textSend <- err.Error()
			log.Println("failed to unmarshal message: ", err)
		}

		switch event := message.GetEvent().(type) {
		case *whisperpb.ServerEvent_DisplayEvent:
			fmt.Printf("DisplayEvent: %s \n", event.DisplayEvent.Text)
			textSend <- event.DisplayEvent.Text
		default:
			fmt.Printf("Unknown event: %T\n", message)
		}

	}

	return nil
}

const (
	resetPin = machine.GP17
	dcPin    = machine.GP16
	csPin    = machine.GP18

	blPin = machine.GP14 // not used, write any pin
)

func startDisplay(text <-chan string) {
	err := machine.SPI1.Configure(machine.SPIConfig{
		Frequency: 24_000_000,
	})
	if err != nil {
		log.Fatal(err)
	}

	display := st7735.New(machine.SPI1, resetPin, dcPin, csPin, blPin)
	display.Configure(st7735.Config{
		Rotation: drivers.Rotation90,
	})

	white := color.RGBA{255, 255, 255, 255}
	black := color.RGBA{0, 0, 0, 255}

	width, height := display.Size()
	fmt.Printf("Display size: %dx%d\n", width, height)

	display.FillScreen(black)

	font := freemono.Bold9pt7b

	for {
		displayText := <-text
		// display.FillScreen(black)

		display.FillScreen(black)

		wrapText(display, font, displayText, 10, 20, 120, 15, white)
	}
}

func wrapText(display st7735.Device, font tinyfont.Font, text string, x, y, maxWidth, lineHeight int16, color color.RGBA) {
	words := strings.Split(text, " ")
	var line string
	currentY := y

	for _, word := range words {
		testLine := line
		if testLine != "" {
			testLine += " "
		}
		testLine += word

		width, _ := tinyfont.LineWidth(&font, testLine)
		if int16(width) > maxWidth {
			// draw the current line
			tinyfont.WriteLine(&display, &font, x, currentY, line, color)
			currentY += lineHeight
			line = word
		} else {
			line = testLine
		}
	}

	// draw the last line
	if line != "" {
		tinyfont.WriteLine(&display, &font, x, currentY, line, color)
	}
}
