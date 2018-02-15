package theater

import (
	"github.com/Synaxis/bfheroesFesl/inter/network"
	"github.com/Synaxis/bfheroesFesl/inter/network/codec"
)

type ansPING struct {
	TheaterID string `fesl:"TID"`
}

func (tm *Theater) PING(event *network.Client) {
	event.Client.WriteEncode(&codec.Packet{
		Type:    thtrPING,
		Payload: ansPING{"0"},
	})
}
