package server

import (
	"fmt"
	"net"
	"sync"

	"github.com/kyren223/eko/internal/utils/log"
	"github.com/kyren223/eko/internal/utils/packets"
)

func handleConnection(conn net.Conn, wg *sync.WaitGroup) {
	log.Info("Accepted client: %v", conn.RemoteAddr().String())
	defer log.Info("Disconnecting client: %v", conn.RemoteAddr().String())
	defer conn.Close()
	defer wg.Done()

	// packet, err := packets.ReadPacket(conn)
	// if err != nil {
	// 	log.Info("Received bad packet: %v", err)
	// 	err = respondWithError(conn, err, 0)
	// 	if err != nil {
	// 		log.Error("Failed to respond to bad packet: %v", err)
	// 	}
	// 	return
	// }
	//
	// if ok, reason := isPacketOk(packet); !ok {
	// 	err = respondWithError(conn, fmt.Errorf("unsupported packet: %v", reason), 0)
	// 	if err != nil {
	// 		log.Error("Failed to respond to unsupported packet: %v", err)
	// 	}
	// 	return
	// }

	buffer := make([]byte, 1024)
	bytesRead, err := conn.Read(buffer)
	if err != nil {
		log.Error("Failed reading: %v", err)
		return
	}
	request := string(buffer[:bytesRead])
	log.Info("Read %v bytes: %v", bytesRead, request)

	response := []byte(fmt.Sprintf("Eko \"%v\"", request))
	bytesWritten, err := conn.Write(response)
	if err != nil {
		log.Error("Failed writing response: %v", err)
		return
	}
	log.Info("Written %v bytes", bytesWritten)
}

func isPacketOk(packet packets.Packet) (bool, string) {
	if packet.Version() != packets.V1 {
		return false, "versions other than 1 are not supported"
	}
	if packet.HasFlag(packets.FlagError) {
		return false, "error flag should only be used by the server"
	}
	return true, ""
}

func respondWithError(conn net.Conn, err error, extraFlag byte) error {
	flag := packets.FlagError | extraFlag
	packet := packets.NewPacket(packets.V1, flag, []byte(err.Error()))
	return packet.Write(conn)
}

func processPacket(packet packets.Packet) (sessId uint16, data []byte, err error) {
	if packet.HasFlag(packets.FlagHandshake) {
	}
	return
}
