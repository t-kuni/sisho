package ksuid

import (
	"github.com/segmentio/ksuid"
	domainKsuid "github.com/t-kuni/sisho/domain/system/ksuid"
)

type KsuidGenerator struct{}

func NewKsuidGenerator() domainKsuid.IKsuid {
	return &KsuidGenerator{}
}

func (k *KsuidGenerator) New() string {
	return ksuid.New().String()
}
