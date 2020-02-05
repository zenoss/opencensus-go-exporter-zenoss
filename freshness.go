package zenoss

import (
	"sync"
	"time"

	"github.com/mitchellh/hashstructure"

	zenoss "github.com/zenoss/zenoss-protobufs/go/cloud/data_receiver"
)

type freshnessChecker struct {
	m  map[uint64]*expiringUint64
	mu sync.Mutex
}

type expiringUint64 struct {
	v uint64
	t int64
}

func newFreshnessChecker(seconds int) *freshnessChecker {
	f := &freshnessChecker{m: make(map[uint64]*expiringUint64)}

	go func() {
		for now := range time.Tick(time.Second) {
			f.mu.Lock()
			for k, v := range f.m {
				if now.Unix()-v.t > int64(seconds) {
					delete(f.m, k)
				}
			}
			f.mu.Unlock()
		}
	}()

	return f
}

func (f *freshnessChecker) isFresh(model *zenoss.Model) bool {
	keyHash, err := hashstructure.Hash(model.Dimensions, nil)
	if err != nil {
		return false
	}

	valueHash, err := hashstructure.Hash(model.MetadataFields, nil)
	if err != nil {
		return false
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	if old, ok := f.m[keyHash]; ok {
		if valueHash == old.v {
			return true
		}
	}

	f.m[keyHash] = &expiringUint64{v: valueHash, t: time.Now().Unix()}

	return false
}
