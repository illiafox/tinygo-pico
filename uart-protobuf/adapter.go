package main

import "machine"

type serialReaderAdapter struct {
	s machine.Serialer
}

func (a *serialReaderAdapter) Read(p []byte) (int, error) {
	for i := range p {
		b, err := a.s.ReadByte()
		if err != nil {
			return i, err // i may be 0
		}
		p[i] = b
	}
	return len(p), nil
}
