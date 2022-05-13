package id

import (
	"github.com/google/uuid"
	"github.com/nats-io/nuid"
)

var (
	UUID ID = &uuidGen{}
	NUID ID = &nuidGen{}
)

// Gen is an interface for generating unique random identifiers.
type ID interface {
	New() string
}

// uuidGen implements IDGen to generate UUIDs.
type uuidGen struct{}

func (i *uuidGen) New() string {
	return uuid.New().String()
}

// nuidGen implements IDGen to generate NUIDs.
type nuidGen struct{}

func (i *nuidGen) New() string {
	return nuid.Next()
}
