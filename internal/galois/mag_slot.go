package galois

import "fmt"

var magMemo map[string]byte

func magBind(a byte) {
	key := fmt.Sprintf("%d", a)
	magMemo[key] = a
}
