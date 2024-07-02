package main

import (
	"fmt"
	"net"
	"os"
	"strings"
)

type request struct {
	verb    string
	path    string
	headers map[string]string
	body    string
}

func main() {
	// You can use print statements as follows for debugging, they'll be visible when running tests.
	fmt.Println("Logs from your program will appear here!")

	// Uncomment this block to pass the first stage
	l, err := net.Listen("tcp", "0.0.0.0:4221")
	fmt.Println("Server listening on port 4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}

	connection, err := l.Accept()
	if err != nil {
		fmt.Println("Error accepting connection: ", err.Error())
		os.Exit(1)
	}

	buf := make([]byte, 2048)
	_, err = connection.Read(buf)
	if err != nil {
		fmt.Println("Error reading from connection: ", err.Error())
		os.Exit(1)
	}

	request := parseRequest(buf)

	path := request.path

	if path == "/" {
		connection.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
	} else if strings.HasPrefix(path, "/echo") {
		echoValue := path[6:]
		fmt.Println(echoValue)
		payload := fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(echoValue), echoValue)
		fmt.Println(payload)
		connection.Write([]byte(payload))
	} else if path == "/user-agent" {
		headerVal := request.headers["User-Agent"]
		payload := fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(headerVal), headerVal)
		connection.Write([]byte(payload))
	} else {
		connection.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
	}

}

func parseRequest(buf []byte) request {
	requestString := string(buf[:])
	requestChunks := strings.Split(requestString, " ")
	verb := requestChunks[0]
	path := requestChunks[1]
	body := strings.Split(requestString, "\r\n\r\n")[1]

	headersStart := strings.Index(requestString, "\r\n") + 4 // headers start just after the first \r\n
	headersEnd := strings.Index(requestString, "\r\n\r\n")

	// header format:
	//		header1: value1\r\n
	//		header2: value2\r\n
	//		...
	headersRaw := requestString[headersStart:headersEnd]
	headers := make(map[string]string)
	for _, pair := range strings.Split(headersRaw, "\r\n") { // split "header section" string on \r\n and iterate over each line
		key := strings.Split(pair, ":")[0]
		value := strings.Split(pair, ":")[1]
		value = strings.TrimSpace(value)
		fmt.Printf("Key: %s, Value: %s", key, value)
		headers[key] = value
	}

	return request{verb: verb, path: path, headers: headers, body: body}
}
