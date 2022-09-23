package ping

import (
	"bytes"
	"fmt"
	"time"

	"github.com/syepes/network_exporter/pkg/common"
	"github.com/syepes/network_exporter/pkg/icmp"
)

// Ping ICMP Operation
func Ping(addr string, ip string, srcAddr string, count int, interval time.Duration, timeout time.Duration, icmpID int) (*PingResult, error) {
	var out PingResult

	pingOptions := &PingOptions{}
	pingOptions.SetCount(count)
	pingOptions.SetTimeout(timeout)
	pingOptions.SetInterval(interval)

	out, err := runPing(addr, ip, srcAddr, icmpID, pingOptions)
	if err != nil {
		return &out, err
	}
	return &out, nil
}

// PingString ICMP Operation
func PingString(addr string, ip string, srcAddr string, count int, timeout time.Duration, interval time.Duration, icmpID int) (result string, err error) {
	pingOptions := &PingOptions{}
	pingOptions.SetCount(count)
	pingOptions.SetTimeout(timeout)
	pingOptions.SetInterval(interval)

	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf("Start %v, PING %v (%v)\n", time.Now().Format("2006-01-02 15:04:05"), addr, addr))
	begin := time.Now().UnixNano() / 1e6
	pingResult, err := runPing(addr, ip, srcAddr, icmpID, pingOptions)
	end := time.Now().UnixNano() / 1e6

	buffer.WriteString(fmt.Sprintf("%v packets transmitted, %v packet loss, time %vms\n", count, pingResult.DropRate, end-begin))
	buffer.WriteString(fmt.Sprintf("rtt min/avg/max = %v/%v/%v ms\n", common.Time2Float(pingResult.WorstTime), common.Time2Float(pingResult.AvgTime), common.Time2Float(pingResult.BestTime)))

	result = buffer.String()

	if err != nil {
		return result, err
	}

	return result, nil
}

func runPing(ipAddr string, ip string, srcAddr string, icmpID int, option *PingOptions) (pingResult PingResult, err error) {
	pingResult.DestAddr = ipAddr
	pingResult.DestIp = ip

	// Avoid collisions/interference caused by multiple coroutines initiating mtr
	pid := icmpID
	timeout := option.Timeout()
	interval := option.Interval()
	ttl := defaultTTL
	pingReturn := PingReturn{}

	seq := 0
	for cnt := 0; cnt < option.Count(); cnt++ {
		icmpReturn, err := icmp.Icmp(ip, srcAddr, ttl, pid, timeout, seq)
		//if err != nil {
		//	pingResult.Success = false
		//	pingResult.DropRate = 1.0
		//	return pingResult, err
		//}

		if err != nil || !icmpReturn.Success || !common.IsEqualIP(ip, icmpReturn.Addr) {
			continue
		}

		pingReturn.allTime = append(pingReturn.allTime, icmpReturn.Elapsed)

		pingReturn.succSum++
		if pingReturn.worstTime == time.Duration(0) || icmpReturn.Elapsed > pingReturn.worstTime {
			pingReturn.worstTime = icmpReturn.Elapsed
		}
		if pingReturn.bestTime == time.Duration(0) || icmpReturn.Elapsed < pingReturn.bestTime {
			pingReturn.bestTime = icmpReturn.Elapsed
		}
		pingReturn.sumTime += icmpReturn.Elapsed
		pingReturn.avgTime = time.Duration((int64)(pingReturn.sumTime/time.Microsecond)/(int64)(pingReturn.succSum)) * time.Microsecond
		pingReturn.success = true

		seq++
		time.Sleep(interval)
	}

	pingResult.Success = pingReturn.success
	pingResult.DropRate = float64(option.Count()-pingReturn.succSum) / float64(option.Count())
	pingResult.SumTime = pingReturn.sumTime
	pingResult.AvgTime = pingReturn.avgTime
	pingResult.BestTime = pingReturn.bestTime
	pingResult.WorstTime = pingReturn.worstTime
	pingResult.SquaredDeviationTime = time.Duration(common.TimeSquaredDeviation(pingReturn.allTime))
	pingResult.UncorrectedSDTime = time.Duration(common.TimeUncorrectedDeviation(pingReturn.allTime))
	pingResult.CorrectedSDTime = time.Duration(common.TimeCorrectedDeviation(pingReturn.allTime))
	pingResult.RangeTime = time.Duration(common.TimeRange(pingReturn.allTime))
	pingResult.SntSummary = option.Count()
	pingResult.SntFailSummary = option.Count() - pingReturn.succSum
	pingResult.SntTimeSummary = time.Duration(common.TimeRange(pingReturn.allTime))

	return pingResult, nil
}
