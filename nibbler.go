package main

type defaultNibbler struct{}

var defNibbler = &defaultNibbler{}

func (d *defaultNibbler) SetNibble(index int, value byte) {

}

func (d *defaultNibbler) GetNibble(index int) byte {
	return 0
}
