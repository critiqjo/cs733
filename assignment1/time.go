// +build linux,ignore

package main

import "time"

func ServerTime() time.Time {
	return time.Now()
}
