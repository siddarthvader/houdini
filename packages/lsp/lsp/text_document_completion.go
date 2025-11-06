package lsp

type CompletionRequest struct {
	Request
	Params CompletionParams `json:"params"`
}

type CompletionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
	Context      *CompletionContext     `json:"context,omitempty"`
}

type CompletionContext struct {
	TriggerKind int `json:"triggerKind"`
	TriggerCharacter string `json:"triggerCharacter"`
}

type CompletionResponse struct {
	Response
	Result CompletionResult `json:"result"`
}

type CompletionResult struct {
	IsIncomplete bool `json:"isIncomplete"`
	Items []CompletionItem `json:"items"`
}

type CompletionItem struct {
	Label string `json:"label"`
	Kind int `json:"kind"`
	Detail string `json:"detail"`
	Documentation string `json:"documentation"`
	SortText string `json:"sortText"`
	FilterText string `json:"filterText"`
	InsertText string `json:"insertText"`
	InsertTextFormat int `json:"insertTextFormat"`
	CommitCharacters []string `json:"commitCharacters"`
	Command *Command `json:"command"`
	Data interface{} `json:"data"`
}

type Command struct {
	Title string `json:"title"`
	Command string `json:"command"`
	Arguments []interface{} `json:"arguments"`
}

func NewCompletionResponse(id int) CompletionResponse {
	return CompletionResponse{
		Response: Response{
			ID:  id,
			RPC: "2.0",
		},
		Result: CompletionResult{
			IsIncomplete: false,
			Items: []CompletionItem{
				{
					Label: "Hello",
					Kind: 1,
					Detail: "Hello",
					Documentation: "Hello",
					SortText: "Hello",
					FilterText: "Hello",
					InsertText: "Hello",
					InsertTextFormat: 1,
				},
			},
		},

	}
}
