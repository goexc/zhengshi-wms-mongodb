package urlx

//判断uri是否在现有列表中
func In(urls []string, url string) bool {
	if len(urls) == 0 {
		return false
	}

	for _, u := range urls {
		if u == url {
			return true
		}
	}
	return false
}
