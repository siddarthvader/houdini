package compiler

type State struct {
	Document map[string]string
}


func NewState() State {
	return State{
		Document: map[string]string{},
	}
}

func (s *State) AddDocument(uri string, content string) {
	s.Document[uri] = content
}

func (s *State) UpdateDocument(uri string, content string) {
	s.Document[uri] = content
}

func (s *State) GetDocument(uri string) string {
	return s.Document[uri]
}
