// +build linux,ignore

package store

import "time"

func ServerTime() time.Time {
	return time.Now()
}
