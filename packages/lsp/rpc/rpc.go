package rpc

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
)



// to encode any messdage into lsp jsonrpc format
func EncodeMessage(msg any) string {
	content, err := json.Marshal(msg)
	if err != nil {
		return err.Error()
	}
	return fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(content), content)
}

type BaseMessage struct {
	Method string `json:"method"`
}

func DecodeMessage(msg []byte) (string, []byte, error) {
	// first get the header that is before \r\n\r\n
	header, content, found := bytes.Cut(msg, []byte{'\r', '\n', '\r', '\n'})

	if !found {
		return "", nil, errors.New("no header found")
	}

	// get the content length
	contentLengthBytes := header[len("Content-Length: "):]
	contentLength, err := strconv.Atoi(string(contentLengthBytes))
	if err != nil {
		return "", nil, err
	}

	var baseMessage BaseMessage
	err = json.Unmarshal(content[:contentLength], &baseMessage)
	if err != nil {
		return "", nil, err
	}
	return baseMessage.Method, content[:contentLength], nil
}


// type SplitFunc func(data []byte, atEOF bool) (advance int, token []byte, err error)
func Split(data []byte, atEOF bool) (advance int, token []byte, err error) {
	// first get the header that is before \r\n\r\n
	header, content, found := bytes.Cut(data, []byte{'\r', '\n', '\r', '\n'})

	if !found {
		return 0, nil, nil
	}

	// get the content length
	contentLengthBytes := header[len("Content-Length: "):]
	contentLength, err := strconv.Atoi(string(contentLengthBytes))
	if err != nil {
		return 0, nil, err
	}

	if len(content) < contentLength {
		return 0, nil, nil
	}

	totalLength := len(content) + 4 + len(header)
	return totalLength, data[:totalLength], nil

}

