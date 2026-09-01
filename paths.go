package boltdb

// Path represents a nested bucket path.
type Path []string

type path [][]byte

func toDBPath(p Path) path {
	bPath := make(path, 0, len(p))
	for _, p := range p {
		bPath = append(bPath, []byte(p))
	}
	return bPath
}

func (p Path) last() string {
	return p[len(p)-1]
}

func (p path) last() []byte {
	return p[len(p)-1]
}
