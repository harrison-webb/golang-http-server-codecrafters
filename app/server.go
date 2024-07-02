package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
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
	filepathPtr := flag.String("directory", "", "")
	flag.Parse()

	filepath := *filepathPtr

	l, err := net.Listen("tcp", "0.0.0.0:4221")
	fmt.Println("Server listening on port 4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}

	for {
		connection, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}

		go handleConnection(connection, filepath)
	}
}

func handleConnection(connection net.Conn, filepath string) {
	defer connection.Close()

	buf := make([]byte, 2048)
	_, err := connection.Read(buf)
	if err != nil {
		fmt.Println("Error reading from connection: ", err.Error())
		os.Exit(1)
	}

	request := parseRequest(buf)

	if request.verb == "GET" {
		processGet(request, connection, filepath)
	} else if request.verb == "POST" {
		processPost(request, connection, filepath)
	} else {
		fmt.Printf("error: %v\n", "verb not implemented")
	}
}

func processGet(request request, connection net.Conn, filepath string) {
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
	} else if strings.HasPrefix(path, "/files") {
		filename := strings.Split(path, "/")[2]
		fileLocation := fmt.Sprintf("%s%s", filepath, filename)
		fileContent, err := os.ReadFile(fileLocation)
		if err != nil {
			// file does not exist
			connection.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
		} else {
			payload := fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: application/octet-stream\r\nContent-Length: %d\r\n\r\n%s", len(fileContent), fileContent)
			connection.Write([]byte(payload))
		}
	} else {
		connection.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
	}

}

func processPost(request request, connection net.Conn, filepath string) {
	path := request.path // e.x. /files/myfile

	if strings.HasPrefix(path, "/files") {
		filename := strings.Split(path, "/")[2]
		fileLocation := fmt.Sprintf("%s%s", filepath, filename)
		// create and write request body to file
		f, err := os.Create(fileLocation)
		if err != nil {
			panic(err)
		}
		_, err = f.WriteString(strings.TrimSpace(request.body))
		if err != nil {
			panic(err)
		}

		connection.Write([]byte("HTTP/1.1 201 Created\r\n\r\n"))
	}
}

func parseRequest(buf []byte) request {
	requestString := string(buf[:])
	requestChunks := strings.Split(requestString, " ")
	verb := requestChunks[0]
	path := requestChunks[1]
	var body string

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

	val, ok := headers["Content-Length"]
	if ok {
		len, err := strconv.Atoi(val)
		if err != nil {
			panic(err)
		}
		body = strings.Split(requestString, "\r\n\r\n")[1][:len]
	} else {
		body = strings.Split(requestString, "\r\n\r\n")[1]
	}

	return request{verb: verb, path: path, headers: headers, body: body}
}
