package main

import (
	"fmt"
	"net"
	"strings"
)

func networkKind(isIPv6 bool) string {
	if isIPv6 {
		return "udp6"
	}
	return "udp4"
}

func resolveInterface(name string, isIPv6 bool) (*net.Interface, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("не удалось получить список сетевых интерфейсов: %w", err)
	}

	if name != "" {
		for i := range ifaces {
			iface := ifaces[i]
			if iface.Name != name {
				continue
			}
			if iface.Flags&net.FlagMulticast == 0 {
				return nil, fmt.Errorf("интерфейс %q не поддерживает multicast; подходящие интерфейсы: %s", name, describeInterfaces(ifaces, isIPv6))
			}
			if !hasAddressFamily(&iface, isIPv6) {
				return nil, fmt.Errorf(
					"интерфейс %q не имеет ни одного %s-адреса (похоже, это не тот интерфейс); подходящие интерфейсы: %s",
					name, familyLabel(isIPv6), describeInterfaces(ifaces, isIPv6),
				)
			}
			return &iface, nil
		}
		return nil, fmt.Errorf("сетевой интерфейс %q не найден; доступные интерфейсы: %s", name, describeInterfaces(ifaces, isIPv6))
	}

	for i := range ifaces {
		iface := ifaces[i]
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		if iface.Flags&net.FlagMulticast == 0 {
			continue
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		if !hasAddressFamily(&iface, isIPv6) {
			continue
		}
		return &iface, nil
	}

	return nil, fmt.Errorf(
		"не найден подходящий сетевой интерфейс с multicast и %s-адресом; укажите его явно через флаг -iface. подходящие интерфейсы: %s",
		familyLabel(isIPv6), describeInterfaces(ifaces, isIPv6),
	)
}

func hasAddressFamily(iface *net.Interface, isIPv6 bool) bool {
	addrs, err := iface.Addrs()
	if err != nil {
		return false
	}
	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok {
			continue
		}
		isV4 := ipNet.IP.To4() != nil
		if isV4 && !isIPv6 {
			return true
		}
		if !isV4 && isIPv6 {
			return true
		}
	}
	return false
}

func familyLabel(isIPv6 bool) string {
	if isIPv6 {
		return "IPv6"
	}
	return "IPv4"
}

func describeInterfaces(ifaces []net.Interface, isIPv6 bool) string {
	var parts []string
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagMulticast == 0 {
			continue
		}
		mark := ""
		if hasAddressFamily(&iface, isIPv6) {
			mark = " [подходит]"
		}
		parts = append(parts, iface.Name+mark)
	}
	if len(parts) == 0 {
		return "(не найдено ни одного включённого multicast-интерфейса)"
	}
	return strings.Join(parts, ", ")
}
