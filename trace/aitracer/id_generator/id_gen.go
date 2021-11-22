package id_generator

import (
	cryptorand "crypto/rand"
	"math"
	"math/big"
	"math/rand"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type IdGenerator struct {
	lock   sync.Mutex
	rand   *rand.Rand
	prefix atomic.Value
}

func New() *IdGenerator {
	var seed int64
	seedN, err := cryptorand.Int(cryptorand.Reader, big.NewInt(math.MaxInt64))
	if err == nil {
		seed = seedN.Int64()
	} else {
		seed = time.Now().UnixNano()
	}
	return &IdGenerator{
		rand: rand.New(rand.NewSource(seed)),
	}
}

func (g *IdGenerator) genUint64() uint64 {
	g.lock.Lock()
	defer g.lock.Unlock()
	return g.rand.Uint64()
}

func (g *IdGenerator) Start() {
	g.prefix.Store(g.genPrefix())
	go func() {
		for range time.Tick(time.Second) {
			g.prefix.Store(g.genPrefix())
		}
	}()
}

var (
	flags = "0123456789abcdef"
)

// 00000000000000000000000000000000
// YYYYMMDDHHMMSSRRRRRRRRRRIIIIIIII
// 202101271500590066145D5A84D5E6FE
func (g *IdGenerator) genPrefix() interface{} {
	sb := strings.Builder{}
	sb.WriteString(time.Now().Format("20060102150405"))
	sb.WriteString("00")

	randv := g.genUint64()
	for i := 0; i < 8; i++ {
		flagv := randv & 0xf
		sb.WriteByte(flags[flagv&0xf])
		randv = randv >> 8
	}
	return sb.String()
}

func (g *IdGenerator) GenId() string {
	sb := strings.Builder{}
	prefixV, _ := g.prefix.Load().(string)
	sb.WriteString(prefixV)

	randv := g.genUint64()
	for i := 0; i < 8; i++ {
		flagv := randv & 0xf
		sb.WriteByte(flags[flagv&0xf])
		randv = randv >> 8
	}
	return sb.String()
}
