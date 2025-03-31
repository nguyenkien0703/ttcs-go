package config

import (
	"fmt"
	"io/ioutil"
	"os"
)

var serverEnvName string = os.Getenv("SERVER_ENV_NAME")

const (
	ServerTypeNameLocal       = "local"
	ServerTypeNameDevelop     = "develop"
	ServerTypeNameClosedAlpha = "closed_alpha"
	ServerTypeNameStaging     = "staging"
	ServerTypeNameStagingMgr  = "staging-mgr"
	ServerTypeNameRelease     = "release"
	ServerTypeNameReleaseMgr  = "release-mgr"
)

const LogDir = "go/src/golang-ttcs-server/log/"
const AppCodeName = "ttcs"

const AmountFirstClaim = 1

// First claim reward
var CharacterCounts = map[uint32]uint64{
	110: 1,
	111: 1,
	112: 1,
	113: 1,
	114: 1,
	115: 1,
	116: 1,
	117: 1,
	118: 1,
	119: 1,
	120: 1,
}
var MiningItemCounts = map[uint32]uint64{
	1103: 2, // Iron Pickaxe id 1103
	1204: 2, // Iron Vest id 1204
	1304: 2, // Iron Helmet id 1304
}

var MasterMiningItemIDsClaim = []uint32{1103, 1204, 1304}
var MasterCharacterIDsClaim = []uint32{110}

var credentialJson []byte = nil
var credentialJsonErr error = nil

func init() {
	filepath := "go/src/golang-ttcs-server/apiaccess.json"
	f, err := os.Open(filepath)
	if err != nil {
		credentialJsonErr = err
		return
	}
	defer f.Close()
	credentialJson, err = ioutil.ReadAll(f)
	if err != nil {
		credentialJsonErr = err
		return
	}
}

func GetServerEnvName() string {
	return serverEnvName
}

func GetIsLocal() bool {
	switch GetServerEnvName() {
	case ServerTypeNameLocal:
		return true
	}
	return false
}

func GetIsDevelop() bool {
	switch GetServerEnvName() {
	case ServerTypeNameDevelop:
		return true
	}
	return false
}

func GetIsClosedAlpha() bool {
	return GetServerEnvName() == ServerTypeNameClosedAlpha
}

func GetIsDebugMode() bool {
	name := GetServerEnvName()
	switch name {
	case ServerTypeNameRelease, ServerTypeNameReleaseMgr:
		return false
	}
	return true
}

func RouteManagementPage() bool {
	switch GetServerEnvName() {
	case ServerTypeNameStaging, ServerTypeNameRelease:
		return false
	}
	return true
}

func GetIsManagement() bool {
	switch GetServerEnvName() {
	case ServerTypeNameStagingMgr, ServerTypeNameReleaseMgr:
		return true
	}
	return false
}

func GetServerRootUrl() string {
	return ""
}

func GetServerApiRootUrl() string {
	switch GetServerEnvName() {
	case ServerTypeNameLocal:
		return "http://localhost:8080/api/"
	}
	return ""
}
func getBlockchainApiURL() string {
	switch GetServerEnvName() {
	case ServerTypeNameLocal:
		return "http://194.233.82.172:3001/"
	case ServerTypeNameClosedAlpha:
		return "http://194.233.82.172:3001/"
	case ServerTypeNameDevelop:
		return "http://194.233.82.172:3001/"
	default:
		return "http://194.233.82.172:3001/"
	}
}

func GetAppUrl() string {
	switch GetServerEnvName() {
	case ServerTypeNameLocal:
		return "http://localhost:8080/"
	}
	return ""
}

func GetOAuthGoogleSettings() map[string]string {
	redirectURL := fmt.Sprintf("%s%s", GetServerApiRootUrl(), "auth/account/google/login")
	switch serverEnvName {
	default:
		return map[string]string{
			"ClientID":     "",
			"ClientSecret": "",
			"RedirectURL":  redirectURL,
		}
	}
}
func GetApiBuyTicketFirstLoginUrl() string {
	return fmt.Sprintf("%s%s", getBlockchainApiURL(), "bonus/add-draw")
}

func GetApiClaimLoginBonusUrl() string {
	return fmt.Sprintf("%s%s", getBlockchainApiURL(), "bonus/login-bonus")

}

func GetApiClaimFirstLoginUrl() string {
	return fmt.Sprintf("%s%s", getBlockchainApiURL(), "bonus/first-login")
}

func GetApiActivePassBlcUrl() string {
	return fmt.Sprintf("%s%s", getBlockchainApiURL(), "game/active-pass")
}

func GetApiTotalClaimRemainingUrl() string {
	return fmt.Sprintf("%s%s", getBlockchainApiURL(), "bonus/total-claim-remaining")
}

func GetApiTotalMaxClaimUrl() string {
	return fmt.Sprintf("%s%s", getBlockchainApiURL(), "bonus/max-claim")
}

func GetApiTokenApproveBlcUrl() string {
	return fmt.Sprintf("%s%s", getBlockchainApiURL(), "token/approve")
}

func GetApiIsFirstClaimUrl() string {
	return fmt.Sprintf("%s%s", getBlockchainApiURL(), "bonus/is-first-login")
}

func GetApiKeyCoinGeckoSettings() string {
	switch GetServerEnvName() {
	// you can write your own api key here.
	case ServerTypeNameLocal:
		return ""
	case ServerTypeNameClosedAlpha:
		return ""
	case ServerTypeNameDevelop:
		return ""
	default:
		return ""
	}
}

func GetGameContractAddress() string {
	switch GetServerEnvName() {
	// you can write your own contract address here.
	case ServerTypeNameLocal:
		return ""
	case ServerTypeNameClosedAlpha:
		return ""
	case ServerTypeNameDevelop:
		return ""
	default:
		return ""
	}
}

func GetAlchemyPayURL() string {
	// you can write your own alchemy pay url here.
	if GetIsDebugMode() {
		return "https://api-nft-sbx.alchemytech.cc"
	}

	return "https://openapi-nft.alchemypay.org/"
}
func GetAlchemyPayURLPage() string {
	if GetIsDebugMode() {
		return "https://nft-sbx.alchemytech.cc"
	}

	return "https://nftcheckout.alchemypay.org/"
}
