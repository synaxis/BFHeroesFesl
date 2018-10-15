package theater

import (
	"time"

	"github.com/Synaxis/bfheroesFesl/inter/network"
	"github.com/Synaxis/bfheroesFesl/inter/network/codec"

	"github.com/sirupsen/logrus"
)

type reqCONN struct {
	TID int `fesl:"TID"`
	Locale string `fesl:"LOCALE"`
	Platform string `fesl:"PLAT"`
	Prod string `fesl:"PROD"`
	Protocol int `fesl:"PROT"`
	SdkVersion string `fesl:"SDKVERSION"`
	Version string `fesl:"VERS"`
}

type ansCONN struct {
	TID         string `fesl:"TID"`
	TIME 		int64  `fesl:"TIME"`
	ConnTTL     int    `fesl:"activityTimeoutSecs"`
	Protocol    string `fesl:"PROT"`
}

// CONN - Enters Theater
func (tm *Theater) CONN(event network.EvProcess) {

	logrus.Println("======CONN=========")
	event.Client.Answer(&codec.Packet{
		Message: "CONN",
		Content: ansCONN{
			//sendPacket->SetVar("ATIME", "NuLoginPersona");
			TID:         event.Process.Msg["TID"],
			TIME: time.Now().UTC().Unix(),
			ConnTTL:     3600,
			Protocol:    event.Process.Msg["PROT"],
		},
	})
}
