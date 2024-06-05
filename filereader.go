package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jacobsa/go-serial/serial"
)

func parseExpectedResults(filename string) ([]byte, error) {
	re := regexp.MustCompile(`expect_([0-9A-Fa-f_]+)`)
	matches := re.FindStringSubmatch(filename)
	if len(matches) < 1 {
		return nil, fmt.Errorf("expected result not found in filename: %s", filename)
	}
	expectedHexStrings := strings.Split(matches[1], "_")
	expectedResults := make([]byte, len(expectedHexStrings))

	for i, hexStr := range expectedHexStrings {
		value, err := strconv.ParseUint(hexStr, 16, 8)
		if err != nil {
			return nil, fmt.Errorf("invalid hex value in filename: %s", hexStr)
		}
		expectedResults[i] = byte(value)
	}

	return expectedResults, nil
}

func loadProgramBytes(file *os.File, programBytes []byte, filename string) ([]byte, bool) {
	scanner := bufio.NewScanner(file)

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
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("Error reading from file %s: %v\n", filename, err)
	}
	return programBytes, true
}

func main() {

	READY_CMD := "READY\r\n"
	LOAD_CMD := "LOAD\r\n"
	RESET_CMD := "RESET\r\n"

	if len(os.Args) < 2 {
		fmt.Println("Please provide a filename")
		os.Exit(1)
	}

	filename := os.Args[1]

	expectedResults, err := parseExpectedResults(filename)
	if err != nil {
		fmt.Println("Error parsing expected results:", err)
		os.Exit(-1)
	}

	fmt.Printf("expectedResults: %v", expectedResults)

	file, err := os.Open(filename)
	if err != nil {
		fmt.Printf("Error opening file %s: %v\n", filename, err)
		os.Exit(1)
	}
	defer file.Close()

	var programBytes []byte

	programBytes, success := loadProgramBytes(file, programBytes, filename)
	if success {
		fmt.Println("Program bytes loaded successfully")
	} else {
		fmt.Println("Failed to load program bytes")
		os.Exit(1)
	}

	fmt.Println("Program Bytes is %v bytes long\n", len(programBytes))
	fmt.Println("Bytes from the first column:")
	for _, b := range programBytes {
		fmt.Printf("%02X\n", b)
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

	fmt.Println("Sending LOAD command")
	loadBytes := []byte(LOAD_CMD)
	fmt.Printf("Load Command is %v bytes long\n", len(loadBytes))
	for _, b := range loadBytes {
		fmt.Printf("%02X ", b)
	}
	fmt.Println()

	for _, b := range loadBytes {
		fmt.Printf("Sending %02X\n", b)
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

	response := make([]byte, 7)
	_, err = port.Read(response)
	if err != nil {
		if err != io.EOF {
			fmt.Printf("Error reading from serial port: %v\n", err)
			return
		}
	}

	if string(response) != READY_CMD {
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
		fmt.Printf("Sending %02X\n", b)
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
		fmt.Printf("Sending %02X\n", b)
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
		fmt.Printf("Sending %02X\n", b)
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

	resetBytes := []byte(RESET_CMD)
	for _, b := range resetBytes {
		fmt.Printf("%02X ", b)
	}
	fmt.Println()

	for _, b := range resetBytes {
		fmt.Printf("Sending %02X\n", b)
		_, err = port.Write([]byte{b})
		if err != nil {
			fmt.Printf("Error write byte to serial port: %v\n", err)
		}
		time.Sleep(100 * time.Millisecond)
	}

	programResults := make([]byte, 2)
	_, err = port.Read(programResults)
	if err != nil {
		if err != io.EOF {
			fmt.Printf("Error reading from serial port: %v\n", err)
			return
		}
	}

	fmt.Printf("Result-1: %02X\n", programResults[0])
	fmt.Printf("Result-2: %02X\n", programResults[1])

	if programResults[0] == expectedResults[0] {
		fmt.Println("test passed")
	}

	readyResponse := make([]byte, 7)
	_, err = port.Read(readyResponse)
	if err != nil {
		if err != io.EOF {
			fmt.Printf("Error reading from serial port: %v\n", err)
			return
		}
	}

	if string(readyResponse) != READY_CMD {
		fmt.Printf("Did not receive 'READY' response; received:", string(readyResponse))
	}

	fmt.Println("Data Transmission complete.")

}
