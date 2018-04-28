package fesl

import (
	"strconv"
	
	"github.com/Synaxis/bfheroesFesl/inter/network"
	"github.com/Synaxis/bfheroesFesl/inter/network/codec"
	"github.com/sirupsen/logrus"
)

const (
	acct                 = "acct"
	acctNuGetAccount     = "NuGetAccount"
	acctNuGetPersonas    = "NuGetPersonas"
)

type ansNuGetPersonas struct {
	TXN      string   `fesl:"TXN"`
	Personas []string `fesl:"personas"`
}

// NuGetPersonas . Display all Personas/Heroes
func (fm *Fesl) NuGetPersonas(event network.EvProcess) {
	if !event.Client.IsActive {
		logrus.Println("Client Left")
		return
	}

	if event.Client.HashState.Get("clientType") == "server" {
		fm.NuGetPersonasServer(event)
		return
	}

	rows, err := fm.db.stmtGetHeroesByUserID.Query(event.Client.HashState.Get("uID"))
	if err != nil {
		return
	}

	ans := ansNuGetPersonas{
		TXN: acctNuGetPersonas,
		Personas: []string{},
		}

	for rows.Next() {
		var id, userID, heroName, online string
		err := rows.Scan(&id, &userID, &heroName, &online)
		if err != nil {
			logrus.Errorln(err)
			return
		}

		ans.Personas = append(ans.Personas, heroName)
		event.Client.HashState.Set("ownerId."+strconv.Itoa(len(ans.Personas)), id)
	}

	event.Client.HashState.Set("numOfHeroes", strconv.Itoa(len(ans.Personas)))

	event.Client.Answer(&codec.Packet{
		Send:    event.Process.HEX,
		Message: acct,
		Content: ans,
	})
}

type ansNuGetAccount struct {
	TXN             string `fesl:"TXN"`
	NucleusID       string `fesl:"nuid"`
	UserID          string `fesl:"userId"`
	HeroName        string `fesl:"heroName"`
	DobDay          int    `fesl:"DOBDay"`
	DobMonth        int    `fesl:"DOBMonth"`
	DobYear         int    `fesl:"DOBYear"`
	Country         string `fesl:"country"`
	Language        string `fesl:"language"`
	GlobalOptIn     bool   `fesl:"globalOptin"`
	ThirdPartyOptIn bool   `fesl:"thirdPartyOptin"`
}

// NuGetAccount - General account information retrieved, based on parameters sent
func (fm *Fesl) NuGetAccount(event network.EvProcess) {
	if !event.Client.IsActive {
		logrus.Println("Client Left")
		return
	}
	fm.acctNuGetAccount(&event)
}

func (fm *Fesl) acctNuGetAccount(event *network.EvProcess) {
	event.Client.Answer(&codec.Packet{
		Message: acct,
		Content: ansNuGetAccount{
			TXN:           		acctNuGetAccount,
			Country:        	"US",
			Language:       	"en_US",
			DobDay:         	1,
			DobMonth:       	1,
			DobYear:        	1992,
			GlobalOptIn:    	false,
			ThirdPartyOptIn:	false,
			NucleusID:      	event.Client.HashState.Get("email"),
			HeroName:       	event.Client.HashState.Get("username"),
			UserID:         	event.Client.HashState.Get("uID"),
		},
		Send: event.Process.HEX,
	})
}
