package main

import (
	"bufio"
	"encoding/json"
	"houdini_lsp/lsp"
	"houdini_lsp/rpc"
	"log"
	"os"
)

func main() {
	logger := get_logger("/home/d2du/code/oss/houdini/houdini-lib/packages/lsp/houdini_lsp.log")
	logger.Println("starting server")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Split(rpc.Split)
	for scanner.Scan() {
		msg := scanner.Bytes()
		method, content, err := rpc.DecodeMessage(msg)
		if err != nil {
			logger.Printf("Error decoding header: %s", err)
			continue
		}
		handleMessage(logger, method, content)
	}
}

func handleMessage(logger *log.Logger, method string, msg []byte) {
	logger.Printf("Handling method: %s", method)
	switch method {
	case "initialize":
		var req lsp.InitializeRequest
		if err := json.Unmarshal(msg, &req); err != nil {
			logger.Printf("Error unmarshalling message: %s", err)
		}
		logger.Printf("Connected to client: %s version: %s", req.Params.ClientInfo.Name, req.Params.ClientInfo.Version)
		msg := lsp.NewInitializeResponse(req.ID)
		reply := rpc.EncodeMessage(msg)

		writer := os.Stdout
		writer.Write([]byte(reply))
		logger.Println("Replied to client")
	}
}

func get_logger(filename string) *log.Logger {
	logfile, err := os.OpenFile(filename, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)
	if err != nil {
		panic("hey, you didnt give me a good file")
	}

	return log.New(logfile, "[houdini_lsp] > ", log.Ldate|log.Ltime|log.Lshortfile)
}
