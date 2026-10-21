package protocol

import (
	"fmt"
	"strconv"
)

func ValidatePort(port string) error {
	n, err := strconv.ParseUint(port, 10, 16)
	if err != nil || n == 0 {
		return fmt.Errorf("invalid port %q: expected a number from 1 to 65535", port)
	}
	return nil
}
