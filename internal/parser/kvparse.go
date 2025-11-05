package parser

import (
	"fmt"
)

type KeyValue struct {
	Key   string
	Value interface{}
}

func NewKeyValue() *KeyValue {
	return &KeyValue{}
}

func Parse(data []byte) error {
	return fmt.Errorf("the first byte is: '%s' ", string(data[0]))
	// ind := bytes.Index(data, []byte("\r\n"))
	// if ind != -1 {
	// 	return fmt.Errorf(" '\r\n' exists as the new line")
	// }
	// ind = bytes.Index(data, []byte("\n"))
	// if ind != -1 {
	// 	return fmt.Errorf(" '\n' exists as the new line")
	// }
	// ind = bytes.Index(data, []byte(", "))
	// if ind != 1 {
	// 	return fmt.Errorf(" ', ' exists as the separator")
	// }
	// return fmt.Errorf("line is not being detected")
}
