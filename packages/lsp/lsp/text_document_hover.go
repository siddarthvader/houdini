package lsp

type HoverRequest struct {
	Request
	HoverParams
}

type HoverParams struct {
	TextDocumentPositionParams
}

type HoverResponse struct {
	Response
	Result HoverResult `json:"result"`
}

type HoverResult struct {
	Contents string `json:"contents"`
}

func NewHoverResponse(id int) HoverResponse {
	return HoverResponse{
		Response: Response{
			ID: id,
		},
		Result: HoverResult{
			Contents: "Hello, from houdini_lsp",
		},
	}
}
