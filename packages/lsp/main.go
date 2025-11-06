package main

import (
	"bufio"
	"encoding/json"
	"houdini_lsp/compiler"
	"houdini_lsp/lsp"
	"houdini_lsp/rpc"
	"io"
	"log"
	"os"
)

func main() {
	logger := get_logger("/home/d2du/code/oss/houdini/houdini-lib/packages/lsp/houdini_lsp.log")
	logger.Println("starting server")

	state := compiler.NewState()

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Split(rpc.Split)

	writer := os.Stdout

	for scanner.Scan() {
		msg := scanner.Bytes()
		method, content, err := rpc.DecodeMessage(msg)
		if err != nil {
			logger.Printf("Error decoding header: %s", err)
			continue
		}

		handleMessage(writer, logger, &state, method, content)
	}
}

func handleMessage(writer io.Writer, logger *log.Logger, state *compiler.State, method string, msg []byte) {
	logger.Printf("Handling method: %s", method)

	switch method {
	case "initialize":
		var req lsp.InitializeRequest
		if err := json.Unmarshal(msg, &req); err != nil {
			logger.Printf("initialize Error unmarshalling message: %s", err)
		}

		logger.Printf("Connected to client: %s version: %s", req.Params.ClientInfo.Name, req.Params.ClientInfo.Version)
		msg := lsp.NewInitializeResponse(req.ID)
		write_response(writer, msg)

	case "textDocument/didOpen":
		var req lsp.DidOpenTextDocumetNotification
		if err := json.Unmarshal(msg, &req); err != nil {
			logger.Printf("textDocument/didOpen Error unmarshalling message: %s", err)
		}

		state.AddDocument(req.Params.TextDocument.URI, req.Params.TextDocument.Text)

	case "textDocument/didChange":
		var req lsp.TextDocumentDidChangeNotification
		if err := json.Unmarshal(msg, &req); err != nil {
			logger.Printf("textDocument/didChange Error unmarshalling message: %s", err)
		}

		for _, change := range req.Params.ContentChanges {
			state.UpdateDocument(req.Params.TextDocument.URI, change.Text)
			// current := state.GetDocument(req.Params.TextDocument.URI)
			// logger.Printf("cccrrent: %s", req.Params.TextDocument.URI)

		}

	case "textDocument/hover":
		var req lsp.HoverRequest
		if err := json.Unmarshal(msg, &req); err != nil {
			logger.Printf("textDocument/hover Error unmarshalling message: %s", err)
		}

		msg := lsp.NewHoverResponse(req.ID)
		write_response(writer, msg)

	 case "textDocument/completion":
		var req lsp.CompletionRequest
		if err := json.Unmarshal(msg, &req); err != nil {
			logger.Printf("textDocument/completion Error unmarshalling message: %s", err)
		}

		logger.Printf("Completion triggered at URI: %s, Line: %d, Character: %d",
			req.Params.TextDocument.URI,
			req.Params.Position.Line,
			req.Params.Position.Character)

		msg := lsp.NewCompletionResponse(req.ID)
		write_response(writer, msg)
	}
}

func get_logger(filename string) *log.Logger {
	logfile, err := os.OpenFile(filename, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)
	if err != nil {
		panic("hey, you didnt give me a good file")
	}

	return log.New(logfile, "[houdini_lsp] > ", log.Ldate|log.Ltime|log.Lshortfile)
}

func write_response(writer io.Writer, msg any) {
	reply := rpc.EncodeMessage(msg)
	writer.Write([]byte(reply))
}
