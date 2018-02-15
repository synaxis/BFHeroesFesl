package network

import (
	"bytes"
	"encoding/binary"
	"errors"
	"net"
	"strings"

	"github.com/Synaxis/bfheroesFesl/inter/network/codec"

	"github.com/sirupsen/logrus"
)

type eventReadFesl func(outCommand *CommandFESL, payloadType string)

func (client *Client) readFESL(data []byte) {
	readFesl(data, func(cmd *CommandFESL, payloadType string) {
		client.eventChan <- ClientEvent{Name: "command." + payloadType, Data: cmd}
		client.eventChan <- ClientEvent{Name: "command", Data: cmd}
	})
}

func (client *Client) readTLSPacket(data []byte) {
	readFesl(data, func(cmd *CommandFESL, payloadType string) {
		client.eventChan <- ClientEvent{Name: "command." + cmd.Message["TXN"], Data: cmd}
		client.eventChan <- ClientEvent{Name: "command", Data: cmd}
	})
}

func (socket *SocketUDP) readFESL(data []byte, addr *net.UDPAddr) {
	p := bytes.NewBuffer(data)
	var payloadID uint32
	var payloadLen uint32

	payloadType := string(data[:4])
	p.Next(4)

	binary.Read(p, binary.BigEndian, &payloadID)
	binary.Read(p, binary.BigEndian, &payloadLen)

	payloadRaw := data[12:]
	payload := codec.DecodeFESL(payloadRaw)

	out := &CommandFESL{
		Query:     payloadType,
		PayloadID: payloadID,
		Message:   payload,
	}

	socket.EventChan <- SocketUDPEvent{Name: "command." + payloadType, Addr: addr, Data: out}
	socket.EventChan <- SocketUDPEvent{Name: "command", Addr: addr, Data: out}
}

func readFesl(data []byte, fireEvent eventReadFesl) {
	var (
		err            error
		payloadID      uint32
		payloadLen     uint32
		payloadTypeRaw = make([]byte, 4)
	)

	payload := bytes.NewBuffer(data)

	if _, err = payload.Read(payloadTypeRaw); err != nil {
		return
	}

	if err = binary.Read(payload, binary.BigEndian, &payloadID); err != nil {
		return
	}

	if err = binary.Read(payload, binary.BigEndian, &payloadLen); err != nil {
		return
	}

	if (payloadLen - 12) > uint32(len(payload.Bytes())) {
		logrus.Errorf("Packet not fully read, payload: %s", payload.Bytes())
	}

	msg := codec.DecodeFESL(payload.Bytes())

	out := &CommandFESL{
		Query:     string(payloadTypeRaw),
		PayloadID: payloadID,
		Message:   msg,
	}

	// logrus.
	// 	WithField("type", "request").
	// 	Debugf("%s", payloadRaw)
	fireEvent(out, string(payloadTypeRaw))
}

type CommandFESL struct {
	Message   map[string]string
	Query     string
	PayloadID uint32
}

// processCommand turns gamespy's command string to the
// command struct
func processCommand(msg string) (*CommandFESL, error) {
	outCommand := new(CommandFESL) // Command not a CommandFESL
	outCommand.Message = make(map[string]string)
	data := strings.Split(msg, `\`)

	// TODO:
	// Should maybe return an emtpy Command struct instead
	if len(data) < 1 {
		logrus.Errorln("Command message invalid")
		return nil, errors.New("Command message invalid")
	}

	// TODO:
	// Check if that makes any sense? Kinda just translated from the js-code
	//		if (data.length < 2) { return out; }
	if len(data) == 1 {
		outCommand.Message["__query"] = data[0]
		outCommand.Query = data[0]
		return outCommand, nil
	}

	outCommand.Query = data[1]
	outCommand.Message["__query"] = data[1]
	for i := 1; i < len(data)-1; i = i + 2 {
		outCommand.Message[strings.ToLower(data[i])] = data[i+1]
	}

	return outCommand, nil
}

func (client *Client) processCommand(command string) {
	gsPacket, err := processCommand(command)
	if err != nil {
		logrus.Errorf("%s: Error processing command %s.\n%v", client.name, command, err)
		//client.eventChan <- client.FireError(err)
		return
	}

	client.eventChan <- ClientEvent{Name: "command." + gsPacket.Query, Data: gsPacket}
	client.eventChan <- ClientEvent{Name: "command", Data: gsPacket}
}

func (socket *SocketUDP) processCommand(command string, addr *net.UDPAddr) {
	gsPacket, err := processCommand(command)
	if err != nil {
		logrus.Errorf("%s: Error processing command %s.\n%v", socket.name, command, err)
		socket.EventChan <- SocketUDPEvent{Name: "error", Addr: addr, Data: err}
		return
	}

	socket.EventChan <- SocketUDPEvent{Name: "command." + gsPacket.Query, Addr: addr, Data: gsPacket}
	socket.EventChan <- SocketUDPEvent{Name: "command", Addr: addr, Data: gsPacket}
}
