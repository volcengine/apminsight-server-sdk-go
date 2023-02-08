package utils

import (
	"fmt"
	"testing"
)

func TestVersion(t *testing.T) {
	fmt.Println(CompareVersion("1.0.27-rc-1", "1.0.28-rc-4"))
	fmt.Println(CompareVersion("1.0.27-rc-1", "1.0.27-rc-4"))
	fmt.Println(CompareVersion("1.0.27-rc-4", "1.0.27-rc-1"))
	fmt.Println(CompareVersion("1.0.27-rc-1", "1.0.27-rc-1"))
	fmt.Println(CompareVersion("1.0.27-rc.1", "1.0.27-rc-2"))
	fmt.Println(CompareVersion("1.11.27-rc.1", "1.1.100-rc-2"))
	fmt.Println(CompareVersion("1.0.27-rc.1", "1.0.27"))
	fmt.Println(CompareVersion("1.0.27-rc.1", "1.0.28"))
	fmt.Println(CompareVersion("", "1.0.28"))
	fmt.Println(CompareVersion("", ""))
	fmt.Println(CompareVersion("1.0.28", ""))
}
