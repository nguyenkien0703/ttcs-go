package lib_db_fields

type CustomType interface {
	Select(columnName string) string
}
