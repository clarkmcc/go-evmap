package eventual

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

type Target interface {
	Insert(key int, value *int)
	Get(key int) (*int, bool)
}

var _ Target = &Map[int, int]{}
var _ Target = &targetMap{}

type targetMap struct {
	lock sync.RWMutex
	m    map[int]*int
}

func (t *targetMap) Insert(key int, value *int) {
	t.lock.Lock()
	defer t.lock.Unlock()
	t.m[key] = value
}

func (t *targetMap) Get(key int) (*int, bool) {
	t.lock.RLock()
	defer t.lock.RUnlock()
	v, ok := t.m[key]
	return v, ok
}

func BenchmarkMap(b *testing.B) {
	// We want to determine given a number of readers and writers, how many reads
	// are we able to perform per second.
	var testCases = []struct {
		writers      int
		readers      int
		keys         int
		refreshEvery int
		duration     time.Duration
	}{
		{1, 10, 10000, 10000, 5 * time.Second},
		{1, 20, 100000, 10000, 5 * time.Second},
		{1, 30, 1000000, 10000, 5 * time.Second},
		{1, 1000, 1000000, 10000, 10 * time.Second},
	}

	for _, c := range testCases {
		for _, impl := range []string{"std", "eventual"} {
			b.Run(fmt.Sprintf("%s/%v/%v/%v/%v", impl, c.writers, c.readers, c.refreshEvery, c.duration.String()), func(b *testing.B) {
				var m Target
				switch impl {
				case "std":
					m = &targetMap{m: map[int]*int{}}
				case "eventual":
					m = NewMap[int, int](WithMaxReplicationWriteLag(c.refreshEvery))
				}
				readsPerSecond, writesPerSecond := Drive(b, BenchParams{
					Writers:  c.writers,
					Readers:  c.readers,
					Keys:     c.keys,
					Duration: c.duration,
				}, m)
				b.ReportMetric(readsPerSecond, "rps")
				b.ReportMetric(writesPerSecond, "wps")
			})
		}
	}
}

type BenchParams struct {
	Writers  int
	Readers  int
	Keys     int
	Duration time.Duration
}

func Drive(b *testing.B, params BenchParams, target Target) (float64, float64) {
	start := time.Now()
	wg := sync.WaitGroup{}
	writesChan := make(chan int, params.Writers)
	for i := 0; i < params.Writers; i++ {
		wg.Add(1)
		go func() {
			writes := 0
			defer wg.Done()
			defer func() {
				writesChan <- writes
			}()
			for {
				if start.Add(params.Duration).Before(time.Now()) {
					break
				}
				k := rand.Intn(params.Keys)
				target.Insert(k, &k)
				writes++
			}
		}()
	}

	readsChan := make(chan int, params.Readers)
	for i := 0; i < params.Readers; i++ {
		wg.Add(1)
		go func() {
			reads := 0
			defer wg.Done()
			defer func() {
				readsChan <- reads
			}()
			for {
				if start.Add(params.Duration).Before(time.Now()) {
					break
				}
				k := rand.Intn(params.Keys)
				target.Get(k)
				reads++
			}
		}()
	}
	wg.Wait()
	close(writesChan)
	close(readsChan)
	var totalReads float64 = 0
	for reads := range readsChan {
		totalReads += float64(reads)
	}
	var totalWrites float64 = 0
	for writes := range writesChan {
		totalWrites += float64(writes)
	}
	return totalReads / params.Duration.Seconds(), totalWrites / params.Duration.Seconds()
}
