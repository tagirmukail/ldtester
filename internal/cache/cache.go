package cache

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/tagirmukail/ldtester/internal/tester"
)

const (
	cleanDuration = time.Minute
)

type Item struct {
	exp      time.Time
	confHash string // this is necessary to check whether the load testing configuration has changed
	data     tester.Item
}

func newItem(confHash string, item tester.Item, expiration time.Duration) Item {
	return Item{
		exp:      time.Now().Add(expiration),
		confHash: confHash,
		data:     item,
	}
}

func (i Item) GetTesterItem() tester.Item {
	return i.data
}

// Cache represent cache with expire values
type Cache struct {
	m sync.Map

	stopCh  chan struct{}
	stopped uint32
}

func New() *Cache {
	c := &Cache{
		m: sync.Map{},
	}

	go c.clean()

	return c
}

func (c *Cache) Get(confHash string, key tester.Key) (Item, bool) {
	i, ok := c.m.Load(key)
	if !ok {
		return Item{}, false
	}

	item, ok := i.(Item)
	if !ok {
		return Item{}, false
	}

	if item.confHash != confHash {
		c.m.Delete(key)
		return Item{}, false
	}

	return item, true
}

func (c *Cache) Set(confHash string, key tester.Key, i tester.Item, expiration time.Duration) {
	c.m.Store(key, newItem(confHash, i, expiration))
}

func (c *Cache) Close() {
	if atomic.LoadUint32(&c.stopped) > 0 {
		return
	}

	atomic.StoreUint32(&c.stopped, 1)

	close(c.stopCh)
}

func (c *Cache) clean() {
	t := time.NewTicker(cleanDuration)
	defer t.Stop()

	for {
		select {
		case <-t.C:
			c.m.Range(func(key, value interface{}) bool {
				item, ok := value.(Item)
				if !ok {
					c.m.Delete(key)
					return true
				}

				if item.exp.After(time.Now()) {
					c.m.Delete(key)
					return true
				}

				return true
			})
		case <-c.stopCh:
			return
		}
	}
}
