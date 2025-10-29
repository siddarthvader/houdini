package rpc_test

import (
	"houdini_lsp/rpc"
	"testing"
	"log"
)

type Message struct {
	Content string `json:"content"`
}

func TestEncodeMessage(t *testing.T) {
	expected := "Content-Length: 25\r\n\r\n{\"content\":\"hello world\"}"
	actual := rpc.EncodeMessage(Message{Content: "hello world"})
	log.Println(actual)
	if expected != actual {
		t.Errorf("expected %s, got %s", expected, actual)
	}
}

func TestDecodeMessage(t *testing.T) {
	incomingMessage := "Content-Length: 15\r\n\r\n{\"Method\":\"hi\"}"
	expectedLen := 15
	method, content, err := rpc.DecodeMessage([]byte(incomingMessage))
	log.Println(method, content, err)
	_ = method
	actualLen := len(content)
	if err != nil {
		t.Errorf("expected no error, got %s", err)
		return
	}
	if expectedLen != actualLen {
		t.Errorf("expected %d, got %d", expectedLen, actualLen)
	}
}
