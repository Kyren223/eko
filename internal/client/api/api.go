package api

import (
	"log"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/kyren223/eko/internal/client/gateway"
	"github.com/kyren223/eko/internal/packet"
	"github.com/kyren223/eko/pkg/assert"
)

func SendMessage(message string) tea.Cmd {
	return func() tea.Msg {
		log.Println("request SendMessage sent")
		request := packet.SendMessage{Content: message}
		responsePayload, ok := <-gateway.Send(&request)
		if !ok {
			log.Println("request timeout")
			return ""
		}
		log.Println("request SendMessage received response")

		response := responsePayload.(*packet.ErrorMessage)
		assert.Assert(ok, "server should return an ErrorMessage response for a SendMessage request")
		assert.Assert(response.IsOk(), "server should always return an OK if we didn't mess up")
		return ""
	}
}
