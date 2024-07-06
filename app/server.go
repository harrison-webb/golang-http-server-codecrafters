package main

import (
	"bytes"
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
)

type request struct {
	verb    []byte
	path    []byte
	headers map[string][]byte // cursed
	body    []byte
}

type response struct {
	code    []byte
	headers map[string][]byte
	body    []byte
}

// func responseToString(response response) string {
// 	var sb strings.Builder
//
// 	var statusLine string
// 	switch response.code {
// 	case "200":
// 		statusLine = "HTTP/1.1 200 OK"
// 	case "201":
// 		statusLine = "HTTP/1.1 201 Created"
// 	case "404":
// 		statusLine = "HTTP/1.1 404 Not Found"
// 	}
//
// 	sb.WriteString(statusLine)
// 	sb.WriteString("\r\n")
//
// 	// convert headers to string
// 	for k, v := range response.headers {
// 		sb.WriteString(k)
// 		sb.WriteString(": ")
// 		sb.WriteString(v)
// 		sb.WriteString("\r\n")
// 	}
//
// 	sb.WriteString("\r\n")
//
// 	sb.WriteString(response.body)
//
// 	return sb.String()
// }

func printResponseBytes(response response) []byte {
	result := make([]byte, 0)

	var statusLine []byte
	if bytes.Equal(response.code, []byte("200")) {
		statusLine = []byte("HTTP/1.1 200 OK")
	} else if bytes.Equal(response.code, []byte("201")) {
		statusLine = []byte("HTTP/1.1 201 Created")
	} else if bytes.Equal(response.code, []byte("404")) {
		statusLine = []byte("HTTP/1.1 404 Not Found")
	}
	result = append(result, statusLine...)
	result = append(result, []byte("\r\n")...)

	// convert headers to string
	for k, v := range response.headers {
		result = append(result, []byte(k)...)
		result = append(result, []byte(": ")...)
		result = append(result, v...)
		result = append(result, []byte("\r\n")...)
	}

	result = append(result, []byte("\r\n")...)

	result = append(result, response.body...)

	return result
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

	buf := make([]byte, 4096)
	_, err := connection.Read(buf)
	if err != nil {
		fmt.Println("Error reading from connection: ", err.Error())
		os.Exit(1)
	}

	request := parseRequest(buf)

	if bytes.Equal(request.verb, []byte("GET")) {
		processGet(request, connection, filepath)
	} else if bytes.Equal(request.verb, []byte("POST")) {
		processPost(request, connection, filepath)
	} else {
		fmt.Printf("error: %v\n", "verb not implemented")
	}
}

func processGet(request request, connection net.Conn, filepath string) {
	response := response{
		code:    make([]byte, 0),
		headers: make(map[string][]byte),
		body:    make([]byte, 0),
	}

	val, ok := request.headers["Accept-Encoding"]
	if ok {
		acceptedEncodings := bytes.Split(val, []byte(", "))
		for _, element := range acceptedEncodings {
			if bytes.Equal(element, []byte("gzip")) {
				response.headers["Content-Encoding"] = []byte("gzip")
			}
		}
	}

	path := request.path

	if bytes.Equal(path, []byte("/")) {
		response.code = []byte("200")
		// connection.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
	} else if bytes.HasPrefix(path, []byte("/echo")) {
		response.code = []byte("200")
		echoValue := path[6:]
		response.headers["Content-Type"] = []byte("text/plain")

		if bytes.Equal(response.headers["Content-Encoding"], []byte("gzip")) {
			// gzip compress "echoValue" then set Content-Length to len(gzip(echoValue))
			fmt.Println("Compressing")
			// gzipEncodedBytes := base64.StdEncoding.EncodeToString(b.Bytes())
			gzipEncodedBytes, length := Compress(echoValue)
			response.headers["Content-Length"] = []byte(strconv.Itoa(length))
			response.body = gzipEncodedBytes
		} else {
			response.headers["Content-Length"] = []byte(strconv.Itoa(len(echoValue)))
			response.body = echoValue
		}
		// payload := fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(echoValue), echoValue)
		// connection.Write([]byte(payload))
	} else if bytes.Equal(path, []byte("/user-agent")) {
		response.code = []byte("200")
		headerVal := request.headers["User-Agent"]
		response.headers["Content-Type"] = []byte("text/plain")
		response.headers["Content-Length"] = []byte(strconv.Itoa(len(headerVal)))
		response.body = headerVal
		// payload := fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(headerVal), headerVal)
		// connection.Write([]byte(payload))
	} else if bytes.HasPrefix(path, []byte("/files")) {
		filename := bytes.Split(path, []byte("/"))[2]
		fileLocation := fmt.Sprintf("%s%s", filepath, filename)
		fileContent, err := os.ReadFile(fileLocation)
		if err != nil {
			// file does not exist
			// connection.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
			response.code = []byte("404")
		} else {
			// payload := fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: application/octet-stream\r\nContent-Length: %d\r\n\r\n%s", len(fileContent), fileContent)
			// connection.Write([]byte(payload))
			response.code = []byte("200")
			response.headers["Content-Type"] = []byte("application/octet-stream")
			response.headers["Content-Length"] = []byte(strconv.Itoa(len(fileContent)))
			response.body = fileContent
		}
	} else {
		response.code = []byte("404")
		// connection.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
	}

	// fmt.Println(responseToString(response))

	connection.Write(printResponseBytes(response))
}

func processPost(request request, connection net.Conn, filepath string) {
	response := response{
		code:    make([]byte, 0),
		headers: make(map[string][]byte),
		body:    make([]byte, 0),
	}

	path := request.path // e.x. /files/myfile

	if bytes.HasPrefix(path, []byte("/files")) {
		filename := bytes.Split(path, []byte("/"))[2]
		fileLocation := fmt.Sprintf("%s%s", filepath, filename)
		// create and write request body to file
		f, err := os.Create(fileLocation)
		if err != nil {
			panic(err)
		}
		_, err = f.WriteString(string(bytes.TrimSpace(request.body))[:])
		if err != nil {
			panic(err)
		}

		// connection.Write([]byte("HTTP/1.1 201 Created\r\n\r\n"))
		response.code = []byte("201")
		connection.Write(printResponseBytes(response))
	}
}

func parseRequest(req []byte) request {
	// requestString := string(buf[:])
	// requestChunks := strings.Split(requestString, " ")
	requestChunks := bytes.Split(req, []byte(" "))
	verb := requestChunks[0]
	path := requestChunks[1]

	var body []byte

	// headersStart := strings.Index(requestString, "\r\n") + 4 // headers start just after the first \r\n
	headersStart := bytes.Index(req, []byte("\r\n")) + 2
	headersEnd := bytes.Index(req, []byte("\r\n\r\n"))

	// header format:
	//		header1: value1\r\n
	//		header2: value2\r\n
	//		...
	headersRaw := req[headersStart:headersEnd] // this is the section of the req byte array with the headers in it
	fmt.Printf("DEBUG: headersRaw\n%s<DONE>", string(headersRaw[:]))
	headers := make(map[string][]byte)
	for _, pair := range bytes.Split(headersRaw, []byte("\r\n")) { // split "header section" string on \r\n and iterate over each line
		key := bytes.Split(pair, []byte(":"))[0]
		value := bytes.Split(pair, []byte(":"))[1]
		// value = strings.TrimSpace(value)
		value = bytes.TrimSpace(value)
		// fmt.Printf("Key: %s, Value: %s", key, value)
		keyToString := string(key[:])
		headers[keyToString] = value
	}

	val, ok := headers["Content-Length"] // if "Content-Length" header exists
	if ok {
		len, err := strconv.Atoi(string(val[:]))
		if err != nil {
			panic(err)
		}
		body = bytes.Split(req, []byte("\r\n\r\n"))[1][:len] // read <content-length> bytes after \r\n\r\n
	} else {
		body = bytes.Split(req, []byte("\r\n\r\n"))[1]
	}

	return request{verb: verb, path: path, headers: headers, body: body}
}

func Compress(payload []byte) (result []byte, resultLength int) {
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	if _, err := gz.Write(payload); err != nil {
		panic(err)
	}

	_ = gz.Close()

	return b.Bytes(), b.Len()
}

func Decompress(payload []byte) []byte {
	buf := bytes.NewReader(payload)
	r, _ := gzip.NewReader(buf)
	defer r.Close()

	var result bytes.Buffer

	_, _ = io.Copy(&result, r)

	return result.Bytes()
}
