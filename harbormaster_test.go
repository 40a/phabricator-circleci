package main

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"testing"
)

var examplePost = `{
    "allParamsJson": {},
    "formparams": {}
}
`

func TestDecodeHM(t *testing.T) {
	hm := harbormasterMessage{}
	assert.Nil(t, json.Unmarshal([]byte(examplePost), &hm))
}
