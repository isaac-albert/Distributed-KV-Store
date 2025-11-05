package parser

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
)

type KeyValue struct {
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
}

type KeyValueArr []KeyValue

func NewKeyValue() *KeyValue {
	return &KeyValue{}
}

func NewKeyValueArray() *KeyValueArr {
	return &KeyValueArr{}
}

func Parse(r io.Reader) error {

	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}

	kvArr := NewKeyValueArray()
	kv := NewKeyValue()

	if err := json.Unmarshal(data, kvArr); err == nil {
		for _, val := range *kvArr {
			log.Printf("kv pair in array: '%s'\n", val)
		}
	} else if err = json.Unmarshal(data, kv); err == nil {
		log.Printf("kv pair single item: '%s'\n", *kv)
	} else {
		log.Printf("kvArr: '%s', kv: '%s', data: '%s'\n", kvArr, *kv, data)
		return fmt.Errorf("bad Request")
	}

	return nil
}

