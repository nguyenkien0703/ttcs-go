package app_defines

import (
	lib_error "application/src/lib/error"
)

var StatusCode = struct {
	OK                  int
	Unknown             int
	InvalidSession      int
	ServerBusy          int
	Restart             int
	Maintenance         int
	BinaryUpdated       int
	AssetUpdated        int
	MasterDataUpdated   int
	IllegalArgs         int
	NotFound            int
	PermissionError     int
	NotReady            int
	UnderLimit          int
	OverLimit           int
	NotEnough           int
	NgWord              int
	Unauthorized        int
	IllegalMasterData   int
	OutOfTerm           int
	AlreadyReceived     int
	Duplicated          int
	NotEntered          int
	Finished            int
	BAN                 int
	UnusableAccounts    int
	PendingApplication  int
	EthInvalidSignature int
	EthInvalidAddress   int
	Payment             int
	TermsUpdated        int
}{
	OK: 0,

	Unknown:             lib_error.DefaultErrorCode,
	InvalidSession:      257,
	ServerBusy:          258,
	Restart:             259,
	Maintenance:         260,
	BinaryUpdated:       261,
	AssetUpdated:        262,
	MasterDataUpdated:   263,
	IllegalArgs:         264,
	NotFound:            265,
	PermissionError:     266,
	NotReady:            267,
	UnderLimit:          268,
	OverLimit:           269,
	NotEnough:           270,
	NgWord:              271,
	Unauthorized:        272,
	IllegalMasterData:   273,
	OutOfTerm:           274,
	AlreadyReceived:     275,
	Duplicated:          276,
	NotEntered:          277,
	Finished:            278,
	BAN:                 279,
	UnusableAccounts:    281,
	PendingApplication:  283,
	EthInvalidSignature: 284,
	EthInvalidAddress:   285,
	Payment:             290,
	TermsUpdated:        291,
}
