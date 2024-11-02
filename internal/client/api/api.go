package api

import (
	"errors"
	"fmt"
	"log"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/kyren223/eko/internal/client/gateway"
	"github.com/kyren223/eko/internal/data"
	"github.com/kyren223/eko/internal/packet"
	"github.com/kyren223/eko/pkg/assert"
)

type AppendMessage data.Message

func SendMessage(message string) tea.Cmd {
	return func() tea.Msg {
		log.Println("request SendMessage sent")
		request := packet.SendMessage{Content: message}
		response, ok := <-gateway.Send(&request)
		if !ok {
			log.Println()
			return errors.New("request timeout")
		}
		log.Println("request SendMessage received response")

		switch response := response.(type) {
		case *packet.ErrorMessage:
			return errors.New(response.Error)
		case *packet.Messages:
			assert.Assert(len(response.Messages) == 1, "server must return only the one message that was sent")
			return AppendMessage(response.Messages[0])
		}
		return fmt.Errorf("received invalid response from server: %v", response.Type())
	}
}
