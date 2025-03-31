package app_defines

const (
	ExternalAccountServiceGoogle = iota + 1
)

// ExternalAccounts : 外部アカウント一覧.
func ExternalAccountServices() []int {
	return []int{
		ExternalAccountServiceGoogle,
	}
}

// ExternalAccountName : 外部アカウント名.
func ExternalAccountServiceName(id int) string {
	switch id {
	case ExternalAccountServiceGoogle:
		return "google"
	}
	return ""
}

// ExternalAccountName : 外部アカウントID.
func ExternalAccountServiceId(name string) int {
	switch name {
	case "google":
		return ExternalAccountServiceGoogle
	}
	return 0
}
