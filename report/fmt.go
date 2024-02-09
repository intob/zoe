package report

import "fmt"

func fmtCount(count uint32) string {
	const unit = 1000
	if count < unit {
		return fmt.Sprintf("%d", count)
	}
	div, exp := uint32(unit), 0
	for n := count / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%c",
		float32(count)/float32(div), "KMGTPE"[exp])
}
