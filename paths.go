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

// Last returns the last element of a Path.
func (p Path) Last() string {
	return p[len(p)-1]
}

// Parent returns a Path with last element removed.
func (p Path) Parent() Path {
	return p[:len(p)-1]
}

func (p path) last() []byte {
	return p[len(p)-1]
}
