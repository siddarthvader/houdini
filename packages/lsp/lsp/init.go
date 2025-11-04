package lsp


type InitializeRequest struct {
	Request
	Params InitializeRequestParams `json:"params"`
}

type InitializeRequestParams struct {
	ClientInfo *ClientInfo `json:"clientInfo"`
}

type ClientInfo struct {
	Name string `json:"name"`
	Version string `json:"version"`
}

type InitializeResponse struct {
	Response
	Result InitializeResult `json:"result"`
}

type InitializeResult struct {
	ServerInfo *ServerInfo `json:"serverInfo"`
	Capabilities *ServerCapabilities `json:"capabilities"`
}

type ServerInfo struct {
	Name string `json:"name"`
	Version string `json:"version"`
}

type ServerCapabilities struct {

}


func NewInitializeResponse(id int) InitializeResponse {
	return InitializeResponse{
		Response: Response{
			ID: id,
			RPC:"2.0",
		},
		Result: InitializeResult{
			ServerInfo: &ServerInfo{
				Name: "houdini-lsp",
				Version: "0.0.1",
			},
			Capabilities: &ServerCapabilities{},
		},
	}
}
