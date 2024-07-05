package main

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
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

type response struct {
	code    string
	headers map[string]string
	body    string
}

func responseToString(response response) string {
	var sb strings.Builder

	var statusLine string
	switch response.code {
	case "200":
		statusLine = "HTTP/1.1 200 OK"
	case "201":
		statusLine = "HTTP/1.1 201 Created"
	case "404":
		statusLine = "HTTP/1.1 404 Not Found"
	}

	sb.WriteString(statusLine)
	sb.WriteString("\r\n")

	// convert headers to string
	for k, v := range response.headers {
		sb.WriteString(k)
		sb.WriteString(": ")
		sb.WriteString(v)
		sb.WriteString("\r\n")
	}

	sb.WriteString("\r\n")

	sb.WriteString(response.body)

	return sb.String()
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
	response := response{
		code:    "",
		headers: map[string]string{},
		body:    "",
	}

	val, ok := request.headers["Accept-Encoding"]
	if ok {
		encodings := strings.Split(val, ", ")
		for _, element := range encodings {
			if element == "gzip" {
				response.headers["Content-Encoding"] = "gzip"
			}
		}
	}

	path := request.path

	if path == "/" {
		response.code = "200"
		// connection.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
	} else if strings.HasPrefix(path, "/echo") {
		response.code = "200"
		echoValue := path[6:]
		response.headers["Content-Type"] = "text/plain"

		if response.headers["Content-Encoding"] == "gzip" {
			// gzip compress "echoValue" then set Content-Length to len(gzip(echoValue))
			var b bytes.Buffer
			gz := gzip.NewWriter(&b)
			if _, err := gz.Write([]byte(echoValue)); err != nil {
				panic(err)
			}
			if err := gz.Close(); err != nil {
				panic(err)
			}
			gzipEncodedString := base64.StdEncoding.EncodeToString(b.Bytes())
			response.headers["Content-Length"] = strconv.Itoa(len(gzipEncodedString))
			response.body = gzipEncodedString
		} else {
			response.headers["Content-Length"] = strconv.Itoa(len(echoValue))
			response.body = echoValue
		}
		// payload := fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(echoValue), echoValue)
		// connection.Write([]byte(payload))
	} else if path == "/user-agent" {
		response.code = "200"
		headerVal := request.headers["User-Agent"]
		response.headers["Content-Type"] = "text/plain"
		response.headers["Content-Length"] = strconv.Itoa(len(headerVal))
		response.body = headerVal
		// payload := fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(headerVal), headerVal)
		// connection.Write([]byte(payload))
	} else if strings.HasPrefix(path, "/files") {
		filename := strings.Split(path, "/")[2]
		fileLocation := fmt.Sprintf("%s%s", filepath, filename)
		fileContent, err := os.ReadFile(fileLocation)
		if err != nil {
			// file does not exist
			// connection.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
			response.code = "404"
		} else {
			// payload := fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: application/octet-stream\r\nContent-Length: %d\r\n\r\n%s", len(fileContent), fileContent)
			// connection.Write([]byte(payload))
			response.code = "200"
			response.headers["Content-Type"] = "application/octet-stream"
			response.headers["Content-Length"] = strconv.Itoa(len(fileContent))
			response.body = string(fileContent[:])
		}
	} else {
		response.code = "404"
		// connection.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
	}

	connection.Write([]byte(responseToString(response)))
}

func processPost(request request, connection net.Conn, filepath string) {
	response := response{
		code:    "",
		headers: map[string]string{},
		body:    "",
	}

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

		// connection.Write([]byte("HTTP/1.1 201 Created\r\n\r\n"))
		response.code = "201"
		connection.Write([]byte(responseToString(response)))
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
		// fmt.Printf("Key: %s, Value: %s", key, value)
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
