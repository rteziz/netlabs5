package protocol

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

func ValidatePort(port string) error {
	n, err := strconv.ParseUint(port, 10, 16)
	if err != nil || n == 0 {
		return fmt.Errorf("invalid port %q: expected a number from 1 to 65535", port)
	}
	return nil
}

func ValidateHost(host string) error {
	if net.ParseIP(host) != nil {
		return nil
	}
	name := strings.TrimSuffix(host, ".")
	if name == "" || len(name) > 253 {
		return fmt.Errorf("invalid host %q: expected an IP address or DNS name", host)
	}

	if strings.Contains(name, ".") && strings.Trim(name, "0123456789.") == "" {
		return fmt.Errorf("invalid IP address %q", host)
	}
	for _, label := range strings.Split(name, ".") {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return fmt.Errorf("invalid DNS name %q", host)
		}
		for _, c := range label {
			if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-') {
				return fmt.Errorf("invalid host %q: expected an IP address or DNS name without a port", host)
			}
		}
	}
	return nil
}
