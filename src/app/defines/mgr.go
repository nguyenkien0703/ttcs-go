package app_defines

import "application/src/config"





var MgrPermissionLevel = struct {
	None  int
	Staff int
	Admin int
}{
	None:  0,
	Staff: 1,
	Admin: 2,
}

var MgrPermissionLevelText = map[int]string{
	MgrPermissionLevel.None:  "None",
	MgrPermissionLevel.Staff: "Staff",
	MgrPermissionLevel.Admin: "Admin",
}

var MgrAlertCode = struct {
	Warning string
	Error   string
	Success string
	Info    string
}{
	Warning: "warning",
	Error:   "error",
	Success: "success",
	Info:    "info",
}

const (
	MasterDataEditHistoryDataTypeEdit = iota
	MasterDataEditHistoryDataTypeDelete
	MasterDataEditHistoryDataTypeCsv
	MasterDataEditHistoryDataTypeSheet
)

func MgrUrl() string {
	switch config.GetServerEnvName() {
	case config.ServerTypeNameLocal:
		return "http://127.0.0.1:8080/mgr/"
	}
	return ""
}
