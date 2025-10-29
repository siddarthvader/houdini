package main

import (
	"bufio"
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
		msg := scanner.Text()
		logger.Println("Received message:", msg)
		handleMessage(msg, logger)
	}
}

func handleMessage(msg string, logger *log.Logger) {
	// TODO: handle message
	logger.Println(msg)
}

func get_logger(filename string) *log.Logger {
	logfile, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		panic("hey, you didnt give me a good file")
	}

	return log.New(logfile, "[houdiuni] > ", log.Ldate|log.Ltime|log.Lshortfile)
}
