package responses

import "github.com/praveen001/uno/pkg/llm/constants"

func UserMessage(msg string) InputMessageUnion {
	return InputMessageUnion{
		OfInputMessage: &InputMessage{Role: constants.RoleUser, Content: InputContent{{OfInputText: &InputTextContent{Text: msg}}}},
	}
}
