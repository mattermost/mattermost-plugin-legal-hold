package legalhold

type Hash struct {
	Path string `csv:"path"`
	Hash string `csv:"hash"`
}

type HashList []Hash

func (h HashList) Exists(path string) bool {
	for _, hash := range h {
		if hash.Path == path {
			return true
		}
	}
	return false
}

func (h HashList) Replace(path, hash string) HashList {
	for i, v := range h {
		if v.Path == path {
			h[i].Hash = hash
			return h
		}
	}
	return h
}

func (h *HashList) Add(path, hash string) HashList {
	return append(*h, Hash{Path: path, Hash: hash})
}
