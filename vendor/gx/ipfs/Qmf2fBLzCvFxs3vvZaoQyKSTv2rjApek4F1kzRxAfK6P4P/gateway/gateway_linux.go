package gateway

import (
	"net"
	"os/exec"
	"fmt"
)

func DiscoverGateway() (ip net.IP, err error) {
	ip, err = discoverGatewayUsingNetstat()
	fmt.Println("DiscoverGateway :::> ",ip,err)
	if err != nil {
		ip, err = discoverGatewayUsingRoute()
		if err!=nil {
			ip, err = discoverGatewayUsingIp()
		}
	}
	return
}

//busybox netstat -rn
func discoverGatewayUsingNetstat() (net.IP, error) {
	routeCmd := exec.Command( "/bin/busybox", "netstat", "-rn")
	output, err := routeCmd.CombinedOutput()
	fmt.Println("discoverGatewayUsingNetstat ::> ",err)
	if err != nil {
		return nil, err
	}
	return parseLinuxRoute(output)
}
func discoverGatewayUsingIp() (net.IP, error) {
	routeCmd := exec.Command("/usr/bin/ip", "route", "show")
	output, err := routeCmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	return parseLinuxIPRoute(output)
}

func discoverGatewayUsingRoute() (net.IP, error) {
	routeCmd := exec.Command("/usr/bin/route", "-n")
	output, err := routeCmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	return parseLinuxRoute(output)
}
