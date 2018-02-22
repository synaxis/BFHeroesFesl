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
	acctNuLogin          = "NuLogin"
	acctNuLoginPersona   = "NuLoginPersona"
	acctNuLookupUserInfo = "NuLookupUserInfo"
)

type ansNuLookupUserInfo struct {
	Taxon    string     `fesl:"TXN"`
	UserInfo []userInfo `fesl:"userInfo"`
}

type userInfo struct {
	Namespace    string `fesl:"namespace"`
	XUID         string `fesl:"xuid"`
	MasterUserID string `fesl:"masterUserId"`
	UserID       string `fesl:"userId"`
	UserName     string `fesl:"userName"`
}

// NuLookupUserInfo - Gets basic information about a game user
func (fm *FeslManager) NuLookupUserInfo(event network.EventClientCommand) {

	if event.Client.HashState.Get("clientType") == "server" && event.Command.Msg["userInfo.0.userName"] == "Test-Server" {
		fm.NuLookupUserInfoServer(event)
		return
	}

	ans := ansNuLookupUserInfo{Taxon: acctNuLookupUserInfo, UserInfo: []userInfo{}}

	logrus.Println("LookupUserInfo CLIENT" + event.Command.Msg["userInfo.0.userName"])

	keys, _ := strconv.Atoi(event.Command.Msg["userInfo.[]"])
	for i := 0; i < keys; i++ {
		heroNamePkt := event.Command.Msg["userInfo."+strconv.Itoa(i)+".userName"]

		var id, userID, heroName, online string
		err := fm.db.stmtGetHeroeByName.QueryRow(heroNamePkt).Scan(&id, &userID, &heroName, &online)
		if err != nil {
			return
		}

		ans.UserInfo = append(ans.UserInfo, userInfo{
			UserName:     heroName,
			UserID:       id,
			MasterUserID: id,
			Namespace:    "MAIN",
			XUID:         "24",
		})
	}

	event.Client.Answer(&codec.Pkt{
		Content: ans,
		Send:    event.Command.HEX,
		Type:    acct,
	})

}

// NuLookupUserInfoServer - Gets basic information about a game user
func (fm *FeslManager) NuLookupUserInfoServer(event network.EventClientCommand) {
	var err error

	var id, userID, servername, secretKey, username string
	err = fm.db.stmtGetServerByID.QueryRow(event.Client.HashState.Get("sID")).Scan(&id, &userID, &servername, &secretKey, &username)
	if err != nil {
		logrus.Errorln(err)
		return
	}

	event.Client.Answer(&codec.Pkt{
		Content: ansNuLookupUserInfo{
			Taxon: acctNuLookupUserInfo,
			UserInfo: []userInfo{
				{
					Namespace:    "MAIN",
					XUID:         "24",
					MasterUserID: "1",
					UserID:       "1",
					UserName:     servername,
				},
			},
		},
		Send: event.Command.HEX,
		Type: acct,
	})
}

type ansNuLoginPersona struct {
	Taxon     string `fesl:"TXN"`
	ProfileID string `fesl:"profileId"`
	UserID    string `fesl:"userId"`
	Lkey      string `fesl:"lkey"`
}

// NuLoginPersona - soldier login command
func (fm *FeslManager) NuLoginPersona(event network.EventClientCommand) {
	if !event.Client.IsActive {
		logrus.Println("C Left")
		return
	}

	if event.Client.HashState.Get("clientType") == "server" {
		// Server login
		fm.NuLoginPersonaServer(event)
		return
	}

	var id, userID, heroName, online string
	err := fm.db.stmtGetHeroeByName.QueryRow(event.Command.Msg["name"]).Scan(&id, &userID, &heroName, &online)
	if err != nil {
		logrus.Println("Wrong Login")
		return
	}

	// Setup a new key for our persona
	lkey := BF2RandomUnsafe(24)
	lkeyRedis := fm.level.NewObject("lkeys", lkey)
	lkeyRedis.Set("id", id)
	lkeyRedis.Set("userID", userID)
	lkeyRedis.Set("name", heroName)

	saveRedis := make(map[string]interface{})
	saveRedis["heroID"] = id
	event.Client.HashState.SetM(saveRedis)

	event.Client.HashState.Set("lkeys", event.Client.HashState.Get("lkeys")+";"+lkey)
	event.Client.Answer(&codec.Pkt{
		Content: ansNuLogin{
			Taxon:     acctNuLoginPersona,
			ProfileID: userID,
			UserID:    userID,
			Lkey:      lkey,
		},
		Send: event.Command.HEX,
		Type: acct,
	})
}

// NuLoginPersonaServer Pre-Server Login (out of order ?)
func (fm *FeslManager) NuLoginPersonaServer(event network.EventClientCommand) {
	var id, userID, servername, secretKey, username string
	err := fm.db.stmtGetServerByName.QueryRow(event.Command.Msg["name"]).Scan(&id, &userID, &servername, &secretKey, &username)
	if err != nil {
		logrus.Println("Wrong Server Login")
		return
	}

	// Setup a new key for our persona
	lkey := BF2RandomUnsafe(24)
	lkeyRedis := fm.level.NewObject("lkeys", lkey)
	lkeyRedis.Set("id", userID)
	lkeyRedis.Set("userID", userID)
	lkeyRedis.Set("name", servername)

	event.Client.HashState.Set("lkeys", event.Client.HashState.Get("lkeys")+";"+lkey)
	event.Client.Answer(&codec.Pkt{
		Content: ansNuLogin{
			Taxon:     acctNuLoginPersona,
			ProfileID: id,
			UserID:    id,
			Lkey:      lkey,
		},
		Send: event.Command.HEX,
		Type: acct,
	})
}

type ansNuGetPersonas struct {
	Taxon    string   `fesl:"TXN"`
	Personas []string `fesl:"personas"`
}

// NuGetPersonas - Soldier data lookup call
func (fm *FeslManager) NuGetPersonas(event network.EventClientCommand) {
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

	ans := ansNuGetPersonas{Taxon: acctNuGetPersonas, Personas: []string{}}

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

	event.Client.Answer(&codec.Pkt{
		Send:    event.Command.HEX,
		Type:    acct,
		Content: ans,
	})
}

// NuGetPersonasServer - Soldier data lookup call for servers
func (fm *FeslManager) NuGetPersonasServer(event network.EventClientCommand) {
	logrus.Println("SERVER CONNECT")

	// Server login
	rows, err := fm.db.stmtGetServerByID.Query(event.Client.HashState.Get("uID"))
	if err != nil {
		return
	}

	ans := ansNuGetPersonas{Taxon: acctNuGetPersonas, Personas: []string{}}

	for rows.Next() {
		var id, userID, servername, secretKey, username string
		err := rows.Scan(&id, &userID, &servername, &secretKey, &username)
		if err != nil {
			logrus.Errorln(err)
			return
		}

		ans.Personas = append(ans.Personas, servername)
		event.Client.HashState.Set("ownerId."+strconv.Itoa(len(ans.Personas)), id)
	}

	event.Client.Answer(&codec.Pkt{
		Send:    event.Command.HEX,
		Type:    acct,
		Content: ans,
	})
}

// NuGetAccount - General account information retrieved, based on parameters sent
func (fm *FeslManager) NuGetAccount(event network.EventClientCommand) {
	if !event.Client.IsActive {
		logrus.Println("Client Left")
		return
	}

	fm.acctNuGetAccount(&event)
}

type ansNuGetAccount struct {
	Taxon          string `fesl:"TXN"`
	NucleusID      string `fesl:"nuid"`
	UserID         string `fesl:"userId"`
	HeroName       string `fesl:"heroName"`
	DobDay         int    `fesl:"DOBDay"`
	DobMonth       int    `fesl:"DOBMonth"`
	DobYear        int    `fesl:"DOBYear"`
	Country        string `fesl:"country"`
	Language       string `fesl:"language"`
	GlobalOptIn    bool   `fesl:"globalOptin"`
	ThidPartyOptIn bool   `fesl:"thidPartyOptin"`
}

func (fm *FeslManager) acctNuGetAccount(event *network.EventClientCommand) {
	event.Client.Answer(&codec.Pkt{
		Type: acct,
		Content: ansNuGetAccount{
			Taxon:          acctNuGetAccount,
			Country:        "US",
			Language:       "en_US",
			DobDay:         1,
			DobMonth:       1,
			DobYear:        2018,
			GlobalOptIn:    false,
			ThidPartyOptIn: false,
			NucleusID:      event.Client.HashState.Get("email"),
			HeroName:       event.Client.HashState.Get("username"),
			UserID:         event.Client.HashState.Get("uID"),
		},
		Send: event.Command.HEX,
	})
}

type ansNuLogin struct {
	Taxon     string `fesl:"TXN"`
	ProfileID string `fesl:"profileId"`
	UserID    string `fesl:"userId"`
	NucleusID string `fesl:"nuid"`
	Lkey      string `fesl:"lkey"`
}

type ansNuLoginErr struct {
	Taxon   string                `fesl:"TXN"`
	Message string                `fesl:"localizedMessage"`
	Errors  []nuLoginContainerErr `fesl:"errorContainer"`
	Code    int                   `fesl:"errorCode"`
}

type nuLoginContainerErr struct {
	Value      string `fesl:"value"`
	FieldError string `fesl:"fieldError"`
	FieldName  string `fesl:"fieldName"`
}

// NuLogin - master login command
// TODO: Here we can implement a banlist/permission check if player is allowed to play/join
func (fm *FeslManager) NuLogin(event network.EventClientCommand) {

	if event.Client.HashState.Get("clientType") == "server" {
		// Server login
		fm.NuLoginServer(event)
		return
	}

	var id, username, email, birthday, language, country, gameToken string

	err := fm.db.stmtGetUserByGameToken.QueryRow(event.Command.Msg["encryptedInfo"]).Scan(&id, &username, &email, &birthday, &language, &country, &gameToken)
	if err != nil {
		event.Client.Answer(&codec.Pkt{
			Content: ansNuLoginErr{
				Taxon:   acctNuLogin,
				Message: `"Wrong Login/Spoof"`,
				Code:    120,
			},
			Send: event.Command.HEX,
			Type: event.Command.Query,
		})
		return
	}

	saveRedis := map[string]interface{}{
		"uID":       id,
		"username":  username,
		"sessionID": gameToken,
		"email":     email,
		"keyHash":   event.Command.Msg["encryptedInfo"],
	}
	event.Client.HashState.SetM(saveRedis)

	// Setup a new key for our persona
	lkey := BF2RandomUnsafe(24)
	lkeyRedis := fm.level.NewObject("lkeys", lkey)
	lkeyRedis.Set("id", id)
	lkeyRedis.Set("userID", id)
	lkeyRedis.Set("name", username)

	event.Client.HashState.Set("lkeys", event.Client.HashState.Get("lkeys")+";"+lkey)
	event.Client.Answer(&codec.Pkt{
		Content: ansNuLogin{
			Taxon:     acctNuLogin,
			ProfileID: id,
			UserID:    id,
			NucleusID: username,
			Lkey:      lkey,
		},
		Send: event.Command.HEX,
		Type: acct,
	})
}

// NuLoginServer - login command for servers
func (fm *FeslManager) NuLoginServer(event network.EventClientCommand) {
	var id, userID, servername, secretKey, username string

	err := fm.db.stmtGetServerBySecret.QueryRow(event.Command.Msg["password"]).Scan(&id, &userID, &servername, &secretKey, &username)
	if err != nil {
		event.Client.Answer(&codec.Pkt{
			Content: ansNuLoginErr{
				Taxon:   acctNuLogin,
				Message: `"Wrong Server "`,
				Code:    122,
			},
			Send: event.Command.HEX,
			Type: acct,
		})
		return
	}

	saveRedis := make(map[string]interface{})
	saveRedis["uID"] = userID
	saveRedis["sID"] = id
	saveRedis["username"] = username
	saveRedis["apikey"] = event.Command.Msg["encryptedInfo"]
	saveRedis["keyHash"] = event.Command.Msg["password"]
	event.Client.HashState.SetM(saveRedis)

	// Setup a new key for new persona
	lkey := BF2RandomUnsafe(24)
	lkeyRedis := fm.level.NewObject("lkeys", lkey)
	lkeyRedis.Set("id", id)
	lkeyRedis.Set("userID", userID)
	lkeyRedis.Set("name", username)

	event.Client.HashState.Set("lkeys", event.Client.HashState.Get("lkeys")+";"+lkey)
	event.Client.Answer(&codec.Pkt{
		Content: ansNuLogin{
			Taxon:     acctNuLogin,
			ProfileID: userID,
			UserID:    userID,
			NucleusID: username,
			Lkey:      lkey,
		},
		Send: event.Command.HEX,
		Type: acct,
	})
}
