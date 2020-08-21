package request

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/valyala/fasthttp"
	"net"
	"time"
)

var BlankAddr = &net.TCPAddr{}
var tcpDialer = fasthttp.TCPDialer{
	Concurrency: 0,
}

type WrapTCPDialer struct {
	*fasthttp.TCPDialer
}

func (d *WrapTCPDialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	ded, _ := ctx.Deadline()
	t := ded.Sub(time.Now())
	return d.TCPDialer.DialTimeout(addr, t)
}

func DialerForLocalAddr(localaddr net.Addr) *WrapTCPDialer {
	/*d := &net.Dialer{
		Timeout:   15 * time.Second,
		KeepAlive: 5 * time.Second,
		LocalAddr: localaddr,
		Control: func(network, address string, c syscall.RawConn) error {
			return c.Control(func(fdPtr uintptr) {
				// got socket file descriptor to set parameters.
				fd := int(fdPtr)
				_ = syscall.SetsockoptLinger(int(fd), syscall.SOL_SOCKET, syscall.SO_LINGER, &syscall.Linger{})
				_ = syscall.SetsockoptInt(fd, syscall.SOL_TCP, syscall.TCP_QUICKACK, 0)
				_ = syscall.SetsockoptInt(int(fd), syscall.IPPROTO_TCP, unix.TCP_FASTOPEN_CONNECT, 1)
				_ = syscall.SetsockoptInt(int(fd), syscall.IPPROTO_TCP, syscall.TCP_KEEPIDLE, 5)
				_ = syscall.SetsockoptInt(int(fd), syscall.IPPROTO_TCP, syscall.TCP_KEEPCNT, 5)
				_ = syscall.SetsockoptInt(int(fd), syscall.IPPROTO_TCP, syscall.TCP_KEEPINTVL, 2)
				//_ = syscall.SetsockoptInt(int(fd), syscall.IPPROTO_TCP, unix.TCP_USER_TIMEOUT, 15)
			})
		},
	}*/
	return &WrapTCPDialer{
		TCPDialer: &fasthttp.TCPDialer{
			Concurrency: 0,
			//LocalAddr:   localaddr.(*net.TCPAddr),
		},
	}
}

//var lock = &sync.Mutex{}
var rr = GetRR()
var anyDialer = DialerForLocalAddr(BlankAddr)

func DialFunc(addr string) (net.Conn, error) {
	var ctx, _ = context.WithTimeout(context.TODO(), time.Second * 5)
	//defer cancel()
	var ip net.IP
	h, _, _ := net.SplitHostPort(addr)
	if ip = net.ParseIP(h); ip.To4() != nil {
		if ip.To4() == nil {
			anyDialer.DialContext(ctx, "tcp", addr)
		}
		return rr.Next().DialContext(ctx, "tcp" ,addr)
	}
	return anyDialer.DialContext(ctx, "tcp", addr)
}

func GetRR() *RoundRobin {
	rr := NewRR()
	//lock.Lock()
	oip, err := GetOutboundIP()
	if err != nil {
		logrus.WithError(err).Error()
	}
	_, ipnet, err := net.ParseCIDR(oip.String() + "/24")
	if err != nil {
		logrus.WithError(err).Error()
	}
	if err != nil {
		logrus.WithError(err).Error()
	}

	for _, listFunc := range []func() (map[string][]string, error){ListIPv4} {
		res, err := listFunc()
		if err != nil {
			continue
		}
		for _, v := range res {
			for _, ip := range v {
				if i := net.ParseIP(ip); i == nil || i.To4() == nil || !ipnet.Contains(i) {
					continue
				} else {
					rr.Add(DialerForLocalAddr(&net.TCPAddr{
						IP:   i,
						Port: 0,
						Zone: "",
					}))
				}
			}
		}
	}
	//lock.Unlock()
	return rr
}

// GetOutboundIP preferred outbound ip of this machine
func GetOutboundIP() (net.IP, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return nil, err
	}
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	if err = conn.Close(); err != nil {
		return nil, err
	}
	return localAddr.IP, nil
}

func GetMacAddrs() ([]string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	addrs := make([]string, len(ifaces))
	for i, ifa := range ifaces {
		addrs[i] = ifa.HardwareAddr.String()
	}

	return addrs, nil
}

func ListIPv4() (map[string][]string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("get net interfaces: %w", err)
	}
	res := make(map[string][]string)
	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			return nil, fmt.Errorf("get addrs %s: %w", i.Name, err)
		}
		for _, addr := range addrs {
			//var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				if ip4 := v.IP.To4(); ip4 != nil {
					res[i.Name] = append(res[i.Name], ip4.String())
				}
			case *net.IPAddr:
				//	if ip4 := v.IP.To4(); ip4 != nil {
				//		res[i.Name] = append(res[i.Name], ip4.String())
				//	}
			}
		}
	}
	return res, nil
}

func ListIPv6() (map[string][]string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("get net interfaces: %w", err)
	}
	res := make(map[string][]string)
	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			return nil, fmt.Errorf("get addrs %s: %w", i.Name, err)
		}
		for _, addr := range addrs {
			switch v := addr.(type) {
			case *net.IPNet:
				if ip6 := v.IP.To16(); ip6 != nil {
					res[i.Name] = append(res[i.Name], ip6.String())
				}
			case *net.IPAddr:
				//	if ip4 := v.IP.To4(); ip4 != nil {
				//		res[i.Name] = append(res[i.Name], ip4.String())
				//	}
			}
		}
	}
	return res, nil
}
