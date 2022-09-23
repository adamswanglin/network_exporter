package target

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/syepes/network_exporter/pkg/common"
	"github.com/syepes/network_exporter/pkg/ping"
)

// PING Object
type PING struct {
	logger   log.Logger
	icmpID   *common.IcmpID
	name     string
	host     string
	ip       string
	srcAddr  string
	interval time.Duration
	timeout  time.Duration
	count    int
	labels   map[string]string
	result   *ping.PingResult
	stop     chan struct{}
	wg       sync.WaitGroup
	sync.RWMutex
}

// NewPing starts a new monitoring goroutine
func NewPing(logger log.Logger, icmpID *common.IcmpID, startupDelay time.Duration, name string, host string, ip string, srcAddr string, interval time.Duration, timeout time.Duration, count int, labels map[string]string) (*PING, error) {
	if logger == nil {
		logger = log.NewNopLogger()
	}
	t := &PING{
		logger:   logger,
		icmpID:   icmpID,
		name:     name,
		host:     host,
		ip:       ip,
		srcAddr:  srcAddr,
		interval: interval,
		timeout:  timeout,
		count:    count,
		labels:   labels,
		stop:     make(chan struct{}),
		result:   &ping.PingResult{},
	}
	t.wg.Add(1)
	go t.run(startupDelay)
	return t, nil
}

func (t *PING) run(startupDelay time.Duration) {
	if startupDelay > 0 {
		select {
		case <-time.After(startupDelay):
		case <-t.stop:
		}
	}

	waitChan := make(chan struct{}, MaxConcurrentJobs)

	tick := time.NewTicker(t.interval)
	for {
		select {
		case <-t.stop:
			tick.Stop()
			t.wg.Done()
			return
		case <-tick.C:
			waitChan <- struct{}{}
			go func() {
				t.ping()
				<-waitChan
			}()
		}
	}
}

// Stop gracefully stops the monitoring
func (t *PING) Stop() {
	close(t.stop)
	t.wg.Wait()
}

func (t *PING) ping() {
	icmpID := int(t.icmpID.Get())
	data, err := ping.Ping(t.host, t.ip, t.srcAddr, t.count, t.interval, t.timeout, icmpID)
	if err != nil {
		level.Error(t.logger).Log("type", "ICMP", "func", "ping", "msg", fmt.Sprintf("%s", err))
	}

	bytes, err2 := json.Marshal(data)
	if err2 != nil {
		level.Error(t.logger).Log("type", "ICMP", "func", "ping", "msg", fmt.Sprintf("%s", err2))
	}
	level.Debug(t.logger).Log("type", "ICMP", "func", "ping", "msg", fmt.Sprintf("%s", string(bytes)))

	t.Lock()
	defer t.Unlock()
	t.result.SntSummary += data.SntSummary
	t.result.SntFailSummary += data.SntFailSummary
	t.result.SntTimeSummary += data.SntTimeSummary
	t.result = data
}

// Compute returns the results of the Ping metrics
func (t *PING) Compute() *ping.PingResult {
	t.RLock()
	defer t.RUnlock()

	if t.result == nil {
		return nil
	}
	return t.result
}

// Name returns name
func (t *PING) Name() string {
	t.RLock()
	defer t.RUnlock()
	return t.name
}

// Host returns host
func (t *PING) Host() string {
	t.RLock()
	defer t.RUnlock()
	return t.host
}

// Ip returns ip
func (t *PING) Ip() string {
	t.RLock()
	defer t.RUnlock()
	return t.ip
}

// Labels returns labels
func (t *PING) Labels() map[string]string {
	t.RLock()
	defer t.RUnlock()
	return t.labels
}
