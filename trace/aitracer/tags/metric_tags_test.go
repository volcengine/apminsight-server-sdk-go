package tags

import (
	"fmt"
	"testing"
)

func TestMerge(t *testing.T) {
	from := NewMetricTagKeysRegister([]string{"foo", "bar", "bar"}, []string{"baz", "qux", "http.status_code"})

	builtin := GetBuiltinTagKeysRegister()
	builtin.MergeTagKeysRegister(from)

	fmt.Println(builtin.GetServerTagKeys())
	fmt.Println(builtin.GetClientTagKeys())
}
