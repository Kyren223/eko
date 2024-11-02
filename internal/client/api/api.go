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
	"github.com/kyren223/eko/pkg/snowflake"
)

type AppendMessage data.Message

func SendMessage(message string) tea.Cmd {
	return func() tea.Msg {
		log.Println("request SendMessage sent")
		frequencyId := snowflake.ID(1852771536100921344)
		request := packet.SendMessage{
			Content:     message,
			FrequencyID: &frequencyId,
		}
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

func GetMessages() tea.Msg {
	log.Println("request GetMessages sent")
	frequencyId := snowflake.ID(1852771536100921344)
	request := packet.GetMessagesRange{
		FrequencyID: &frequencyId,
		ReceiverID:  nil,
		From:        nil,
		To:          nil,
	}
	response, ok := <-gateway.Send(&request)
	if !ok {
		log.Println()
		return errors.New("request timeout")
	}
	log.Println("request GetMessages received response")

	switch response := response.(type) {
	case *packet.ErrorMessage:
		return errors.New(response.Error)
	case *packet.Messages:
		return response
	}
	return fmt.Errorf("received invalid response from server: %v", response.Type())
}
