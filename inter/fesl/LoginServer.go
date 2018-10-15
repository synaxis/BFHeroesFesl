package fesl

import (
	"github.com/Synaxis/bfheroesFesl/inter/network"
	"github.com/Synaxis/bfheroesFesl/inter/network/codec"
	"github.com/sirupsen/logrus"
)

// NuLoginServer - NuLogin for gameServer.exe
func (fm *Fesl) NuLoginServer(event network.EvProcess) {
	ready := event.Client.IsActive
	if !ready {
		logrus.Println("C Left")
		return
	}

	logrus.Println("===NuLoginServer===")

	if event.Client.HashState.Get("clientType") != "server" {
		logrus.Println("===Possible Exploit==")
		return
	}

	var id, userID, servername, secretKey, username string
	err := fm.db.stmtGetServerBySecret.QueryRow(event.Process.Msg["password"]).Scan(&id,
		&userID, &servername, &secretKey, &username)

	if err != nil {
		logrus.Println("===NuLogin issue=")
		return
	}

	saveRedis := make(map[string]interface{})
	saveRedis["uID"] = userID
	saveRedis["sID"] = id
	saveRedis["username"] = username
	saveRedis["apikey"] = event.Process.Msg["encryptedInfo"]
	saveRedis["keyHash"] = event.Process.Msg["password"]
	event.Client.HashState.SetM(saveRedis)

	//Setup new key for our persona	
	tempKey, err := randomize()
	lkeyRedis := fm.level.NewObject("lkeys", tempKey)
	lkeyRedis.Set("id", id)
	lkeyRedis.Set("userID", userID)
	lkeyRedis.Set("name", username)

	if !ready {
		logrus.Println("AFK")
		return
	}

	event.Client.HashState.Set("lkeys", event.Client.HashState.Get("lkeys")+";"+tempKey)
	event.Client.Answer(&codec.Packet{
		Content: ansNuLogin{
			TXN:       acctNuLogin,
			ProfileID: userID,
			UserID:    userID,
			NucleusID: username,
			Lkey:      tempKey,
		},
		Send:    event.Process.HEX,
		Message: acct,
	})
}

//NuLoginPersonaServer The Login is based on the Name
//there's only 1 persona(hero) for the server, so it works like a password
func (fm *Fesl) NuLoginPersonaServer(event network.EvProcess) {
	ready := event.Client.IsActive
	if !ready {
		logrus.Println("AFK")
		return
	}

	logrus.Println("===LoginPersonaServer===")
	/////Checks///////

	if event.Client.HashState.Get("clientType") != "server" {
		logrus.Println("===Possible Exploit===")
		fm.Goodbye(event)
		return
	}

	var id, userID, servername, secretKey, username string
	err := fm.db.stmtGetServerByName.QueryRow(event.Process.Msg["name"]).Scan(&id, //continue
		&userID, &servername, &secretKey, &username)

	if event.Client.HashState.Get("clientType") != "server" || err != nil {
		logrus.Println("===Possible Exploit===")
		fm.Goodbye(event)
		return
	}

	// Setup a new key for our persona	
	tempKey, err := randomize()
	lkeyRedis := fm.level.NewObject("lkeys", tempKey)
	lkeyRedis.Set("id", userID)
	lkeyRedis.Set("userID", userID)
	lkeyRedis.Set("name", servername)

	event.Client.HashState.Set("lkeys", event.Client.HashState.Get("lkeys")+";"+tempKey)
	event.Client.Answer(&codec.Packet{
		Content: ansNuLogin{
			TXN:       acctNuLoginPersona,
			ProfileID: id,
			UserID:    id,
			Lkey:      tempKey,
		},
		Send:    event.Process.HEX,
		Message: acct,
	})

	logrus.Println("=== Server  Login OK===")
}
