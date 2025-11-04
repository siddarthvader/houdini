package lsp

type Request struct {
	RPC    string `json:"jsonrpc"`
	ID     int    `json:"id"`
	Method string `json:"method"`
	Params any    `json:"params"`
}

type Response struct {
	RPC    string `json:"jsonrpc"`
	ID     int    `json:"id,omitempty"`
	// Result any    `json:"result"`
	// Error  any    `json:"error"`
}

type Notification struct {
	RPC    string `json:"jsonrpc"`
	Method string `json:"method"`
	Params any    `json:"params"`
}
