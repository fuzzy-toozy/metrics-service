package err_check_test_data

import "runtime"

type Kek struct {
}

func (k *Kek) Lol() error {
	return nil
}

func mulfunc(i int) (int, error) {
	return i * 2, nil
}

func errCheckFunc() {

	mulfunc(5)           // want "expression returns unchecked error"
	res, _ := mulfunc(5) // want "assignment with unchecked error"
	runtime.KeepAlive(res)

	k := Kek{}
	k.Lol() // want "expression returns unchecked error"

	defer mulfunc(5) // want "defer statement with unchecked error"
	go mulfunc(5)    // want "go statement with unchecked error"

	defer k.Lol() // want "defer statement with unchecked error"
	go k.Lol()    // want "go statement with unchecked error"
}
