package target

import (
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/syepes/network_exporter/pkg/common"
	"github.com/syepes/network_exporter/pkg/mtr"
)

// MTR Object
type MTR struct {
	logger   log.Logger
	icmpID   *common.IcmpID
	name     string
	host     string
	srcAddr  string
	interval time.Duration
	timeout  time.Duration
	maxHops  int
	count    int
	labels   map[string]string
	result   *mtr.MtrResult
	stop     chan struct{}
	wg       sync.WaitGroup
	sync.RWMutex
}

// NewMTR starts a new monitoring goroutine
func NewMTR(logger log.Logger, icmpID *common.IcmpID, startupDelay time.Duration, name string, host string, srcAddr string, interval time.Duration, timeout time.Duration, maxHops int, count int, labels map[string]string) (*MTR, error) {
	if logger == nil {
		logger = log.NewNopLogger()
	}
	t := &MTR{
		logger:   logger,
		icmpID:   icmpID,
		name:     name,
		host:     host,
		srcAddr:  srcAddr,
		interval: interval,
		timeout:  timeout,
		maxHops:  maxHops,
		count:    count,
		labels:   labels,
		result:   &mtr.MtrResult{HopSummaryMap: map[string]*common.IcmpSummary{}},
		stop:     make(chan struct{}),
	}
	t.wg.Add(1)
	go t.run(startupDelay)
	return t, nil
}

func (t *MTR) run(startupDelay time.Duration) {
	if startupDelay > 0 {
		select {
		case <-time.After(startupDelay):
		case <-t.stop:
		}
	}

	tick := time.NewTicker(t.interval)
	for {
		select {
		case <-t.stop:
			tick.Stop()
			t.wg.Done()
			return
		case <-tick.C:
			go t.mtr()
		}
	}
}

// Stop gracefully stops the monitoring
func (t *MTR) Stop() {
	close(t.stop)
	t.wg.Wait()
}

func (t *MTR) mtr() {
	icmpID := int(t.icmpID.Get())
	data, err := mtr.Mtr(t.host, t.srcAddr, t.maxHops, t.count, t.timeout, icmpID)
	if err != nil {
		level.Error(t.logger).Log("type", "MTR", "func", "mtr", "msg", fmt.Sprintf("%s", err))
	}

	bytes, err2 := json.Marshal(data)
	if err2 != nil {
		level.Error(t.logger).Log("type", "MTR", "func", "mtr", "msg", fmt.Sprintf("%s", err2))
	}
	level.Debug(t.logger).Log("type", "MTR", "func", "mtr", "msg", fmt.Sprintf("%s", string(bytes)))

	t.Lock()
	summaryMap := t.result.HopSummaryMap
	t.result = data
	for _, hop := range data.Hops {
		summary := summaryMap[strconv.Itoa(hop.TTL)+"_"+hop.AddressTo]
		if summary == nil {
			summary = &common.IcmpSummary{}
			summaryMap[strconv.Itoa(hop.TTL)+"_"+hop.AddressTo] = summary
		}
		summary.AddressFrom = hop.AddressFrom
		summary.AddressTo = hop.AddressTo
		summary.Snt += hop.Snt
		summary.SntTime += hop.SumTime
		summary.SntFail += hop.SntFail
	}
	t.result.HopSummaryMap = summaryMap

	t.Unlock()
}

// Compute returns the results of the MTR metrics
func (t *MTR) Compute() *mtr.MtrResult {
	t.RLock()
	defer t.RUnlock()

	if t.result == nil {
		return nil
	}
	return t.result
}

// Name returns name
func (t *MTR) Name() string {
	t.RLock()
	defer t.RUnlock()
	return t.name
}

// Host returns host
func (t *MTR) Host() string {
	t.RLock()
	defer t.RUnlock()
	return t.host
}

// Labels returns labels
func (t *MTR) Labels() map[string]string {
	t.RLock()
	defer t.RUnlock()
	return t.labels
}
