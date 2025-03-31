package config

const DbMain string = "main"
const DbSub0 string = "sub0"

func GetdbConnectionSettings() map[string]map[string]string {
	return map[string]map[string]string{
		DbMain: {
			"Host":          "db",
			"Port":          "3306",
			"Database":      "ttcs",
			"User":          "root",
			"Password":      "AQiX3tm9J0@!#",
			"MaxConnection": "10",
		},
		DbSub0: {
			"Host":          "db",
			"Port":          "3306",
			"Database":      "ttcs",
			"User":          "root",
			"Password":      "AQiX3tm9J0@!#",
			"MaxConnection": "10",
		},
	}
}

func TablePriority() []string {
	return []string{
		// you can write priority table name here.
		"",
	}
}

func TablePriorityMap() map[string]int {
	l := TablePriority()
	length := len(l)

	m := make(map[string]int, length)
	for i, v := range l {
		m[v] = length - i
	}
	return m
}
