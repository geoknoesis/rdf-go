package rdf

func isQNameLocal(value string) bool {
	if value == "" {
		return false
	}
	for i := 0; i < len(value); i++ {
		ch := value[i]
		if i == 0 {
			if !isNameStartChar(ch) {
				return false
			}
		} else if !isNameChar(ch) {
			return false
		}
	}
	return true
}

func isNameStartChar(ch byte) bool {
	return (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') || ch == '_'
}

func isNameChar(ch byte) bool {
	return isNameStartChar(ch) || (ch >= '0' && ch <= '9') || ch == '-' || ch == '.'
}
