package feign

import (
	"errors"
	"fmt"
	"math/rand"
)

var (
	mgr = BalanceManager{
		balances: make(map[string]Balance),
	}
)

func init() {
	mgr.balances["random"] = &RandomBalance{}
	mgr.balances["roundrobin"] = &RandomBalance{}
}

type Balance interface {
	DoBalance(urls []string) (string, error)
}

type BalanceManager struct {
	balances map[string]Balance
}

func DoBalance(balanceType string, urls []string) (string, error) {
	balance := mgr.balances[balanceType]
	return balance.DoBalance(urls)
}

type RandomBalance struct {
}

func (b *RandomBalance) DoBalance(urls []string) (string, error) {
	lens := len(urls)
	if lens == 0 {
		return "", errors.New("url列表为空")
	}

	index := rand.Intn(lens)
	inst := urls[index]

	return inst, nil
}

type RoundRobinBalance struct {
	curIndex int
}

func (b *RoundRobinBalance) DoBalance(urls []string) (string, error) {
	lens := len(urls)
	if lens == 0 {
		return "", fmt.Errorf("url列表为空")
	}

	if b.curIndex >= lens {
		b.curIndex = 0
	}
	inst := urls[b.curIndex]

	b.curIndex = (b.curIndex + 1) % lens
	return inst, nil
}
