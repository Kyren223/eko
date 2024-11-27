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

type (
	AppendMessage     data.Message
	UserProfileUpdate data.User
)

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
			return errors.New("request SendMessage timeout")
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
		return errors.New("request GetMessages timeout")
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

func GetUserById(id snowflake.ID) tea.Cmd {
	return func() tea.Msg {
		log.Println("request GetUserById for ID", id, "sent")
		request := packet.GetUserByID{UserID: id}
		response, ok := <-gateway.Send(&request)
		if !ok {
			return errors.New("request GetUserById timeout")
		}
		log.Println("request GetUserById received response")

		switch response := response.(type) {
		case *packet.ErrorMessage:
			return errors.New(response.Error)
		case *packet.Users:
			if len(response.Users) == 0 {
				return fmt.Errorf("requested user id %v not found", id)
			}
			assert.Assert(len(response.Users) == 1, "server must return only one user with the matching id")
			return UserProfileUpdate(response.Users[0])
		}
		return fmt.Errorf("received invalid response from server: %v", response.Type())
	}
}

func CreateServer(
	name string,
	icon string,
	bgHexColor string,
	fgHexColor string,
	isPublic bool,
) tea.Cmd {
	return func() tea.Msg {
		requestName := "CreateServer"
		log.Println("request", requestName, "sent")
		request := packet.CreateNetwork{
			Name:       name,
			Icon:       icon,
			BgHexColor: bgHexColor,
			FgHexColor: fgHexColor,
			IsPublic:   isPublic,
		}
		response, ok := <-gateway.Send(&request)
		if !ok {
			return fmt.Errorf("request %s timeout", requestName)
		}
		log.Println("request", requestName, "received response")

		switch response := response.(type) {
		case *packet.ErrorMessage:
			return errors.New(response.Error)
		case *packet.Users:
			if len(response.Users) == 0 {
				return fmt.Errorf("requested user id %v not found", id)
			}
			assert.Assert(len(response.Users) == 1, "server must return only one user with the matching id")
			return UserProfileUpdate(response.Users[0])
		}
		return fmt.Errorf("received invalid response from server: %v", response.Type())
	}
}
