package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jacobsa/go-serial/serial"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Please provide a filename")
		os.Exit(1)
	}

	filename := os.Args[1]

	file, err := os.Open(filename)
	if err != nil {
		fmt.Printf("Error opening file %s: %v\n", filename, err)
		os.Exit(1)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	var programBytes []byte

	for scanner.Scan() {
		line := scanner.Text()

		colonIndex := strings.Index(line, ":")
		if colonIndex == -1 {
			fmt.Println("Invalid line format: no colon found")
			continue
		}

		addressStr := line[:colonIndex]
		address, err := strconv.ParseUint(addressStr, 16, 16)
		if err != nil {
			fmt.Printf("Error parsing address: %v\n", err)
			continue
		}
		fmt.Printf("Address: %04X\n", address)

		remainder := line[colonIndex+1:]

		remainderParts := strings.SplitN(remainder, "--", 2)
		hexPart := remainderParts[0]

		hexStrings := strings.Fields(hexPart)

		for _, hexStr := range hexStrings {
			hexStr = strings.TrimSpace(hexStr)
			if hexStr == "" {
				continue
			}
			byteValue, err := strconv.ParseUint(hexStr, 16, 8)
			if err != nil {
				fmt.Printf("Error converting hex to byte: %v\n", err)
				continue
			}
			programBytes = append(programBytes, byte(byteValue))
		}

		index := strings.IndexFunc(line, func(r rune) bool {
			return r == ' ' || r == '\t'
		})

		if index != -1 {
			binaryStr := line[:index]
			b, err := strconv.ParseUint(binaryStr, 2, 8)
			if err != nil {
				fmt.Printf("Error converting binary to byte: %v\n", err)
				continue
			}
			programBytes = append(programBytes, byte(b))
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("Error reading from file %s: %v\n", filename, err)
	}

	fmt.Println("Program Bytes is %d bytes long\n", len(programBytes))
	fmt.Println("Bytes from the first column:")
	for _, b := range programBytes {
		fmt.Printf("%08b\n", b)
	}

	options := serial.OpenOptions{
		PortName:        "/dev/ttyUSB1",
		BaudRate:        115200,
		DataBits:        8,
		StopBits:        1,
		MinimumReadSize: 1,
	}

	port, err := serial.Open(options)
	if err != nil {
		fmt.Printf("Error opening serial port: %v\n", err)
		os.Exit(1)
	}
	defer port.Close()

	loadBytes := []byte("LOAD")
	for _, b := range loadBytes {
		fmt.Printf("%02X ", b)
	}
	fmt.Println()

	for _, b := range loadBytes {
		_, err = port.Write([]byte{b})
		if err != nil {
			fmt.Printf("Error write byte to serial port: %v\n", err)
		}
		time.Sleep(100 * time.Millisecond)
	}

	//_, err = port.Write([]byte("LOAD"))

	// if err != nil {
	// 	fmt.Printf("Error sending 'LOAD' to serial port: %v\n", err)
	// 	return
	// }

	response := make([]byte, 5)
	_, err = port.Read(response)
	if err != nil {
		if err != io.EOF {
			fmt.Printf("Error reading from serial port: %v\n", err)
			return
		}
	}

	if string(response) != "READY" {
		fmt.Printf("Did not receive 'READY' response; received:", string(response))
	}

	fmt.Printf("Ready Received proceeding ...")

	checksumCalculated := byte(0)

	messageLength := len(programBytes) + 4
	fmt.Printf("messageLength: %v\n", messageLength)
	messageLengthLE := []byte{byte(messageLength & 0xFF), byte(messageLength >> 8)}
	for _, b := range messageLengthLE {
		fmt.Printf("%02X ", b)
	}
	fmt.Println()

	for _, b := range messageLengthLE {
		checksumCalculated ^= b
		_, err = port.Write([]byte{b})
		if err != nil {
			fmt.Printf("Error write byte to serial port: %v\n", err)
		}
		time.Sleep(100 * time.Millisecond)
	}

	startAddress := 0x0000
	fmt.Printf("startAddress: %v\n", startAddress)
	startAddressLE := []byte{byte(startAddress & 0xFF), byte(startAddress >> 8)}
	for _, b := range startAddressLE {
		fmt.Printf("%02X ", b)
	}
	fmt.Println()

	fmt.Printf("Sending Starting Address")
	for _, b := range startAddressLE {
		checksumCalculated ^= b
		_, err = port.Write([]byte{b})
		if err != nil {
			fmt.Printf("Error write byte to serial port: %v\n", err)
		}
		time.Sleep(100 * time.Millisecond)
	}
	fmt.Println()

	fmt.Printf("Sending Program Bytes\n")
	for _, b := range programBytes {
		fmt.Printf("%02X ", b)
	}
	fmt.Println()

	for _, b := range programBytes {
		checksumCalculated ^= b
		_, err = port.Write([]byte{b})
		if err != nil {
			fmt.Printf("Error write byte to serial port: %v\n", err)
		}
		time.Sleep(100 * time.Millisecond)
	}
	fmt.Println()

	fmt.Printf("Calculated Checksum: %X\n", checksumCalculated)

	fmt.Printf("Waiting to Receive checksum\n")
	checksumReceived := make([]byte, 1)
	_, err = port.Read(checksumReceived)
	if err != nil {
		if err != io.EOF {
			fmt.Printf("Error Reading checksum from serial port: %v\n", err)
		}
	}

	fmt.Println()
	fmt.Printf("Received checksum: %02X\n", checksumReceived[0])
	fmt.Println("Data Transmission complete.")

}
