package strings

func Or(args ...string) string {
	for _, str := range args {
		if str != "" {
			return str
		}
	}
	return ""
}
