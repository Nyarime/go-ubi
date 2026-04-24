package ubi

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

// WearLevelInfo holds wear leveling statistics for a UBI image
type WearLevelInfo struct {
	TotalPEBs    int
	UsedPEBs     int
	FreePEBs     int
	BadPEBs      int
	MinEC        uint64
	MaxEC        uint64
	AvgEC        float64
	MedianEC     uint64
	ECDistribution map[uint64]int // erase count -> number of PEBs
}

// AnalyzeWearLeveling scans all PEBs and collects erase counter statistics
func (r *Reader) AnalyzeWearLeveling() (*WearLevelInfo, error) {
	if r.image == nil {
		return nil, fmt.Errorf("call Parse() first")
	}

	info := &WearLevelInfo{
		MinEC:          math.MaxUint64,
		ECDistribution: make(map[uint64]int),
	}

	var ecValues []uint64
	numPEBs := int(r.size) / r.image.PEBSize

	for i := 0; i < numPEBs; i++ {
		offset := int64(i) * int64(r.image.PEBSize)
		ec, err := r.readECHeader(offset)
		if err != nil {
			info.BadPEBs++
			continue
		}

		info.TotalPEBs++
		ecValues = append(ecValues, ec.EC)
		info.ECDistribution[ec.EC]++

		if ec.EC < info.MinEC { info.MinEC = ec.EC }
		if ec.EC > info.MaxEC { info.MaxEC = ec.EC }

		// Check if PEB has data (VID header present)
		vid, err := r.readVIDHeader(offset + int64(ec.VIDHdrOff))
		if err == nil && vid.Magic == UBI_VID_HDR_MAGIC {
			info.UsedPEBs++
		} else {
			info.FreePEBs++
		}
	}

	// Calculate average
	if len(ecValues) > 0 {
		var sum uint64
		for _, ec := range ecValues {
			sum += ec
		}
		info.AvgEC = float64(sum) / float64(len(ecValues))

		// Median
		sort.Slice(ecValues, func(i, j int) bool { return ecValues[i] < ecValues[j] })
		info.MedianEC = ecValues[len(ecValues)/2]
	}

	return info, nil
}

// String formats wear leveling info
func (w *WearLevelInfo) String() string {
	var sb strings.Builder
	sb.WriteString("📊 Wear Leveling Analysis\n")
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	sb.WriteString(fmt.Sprintf("  Total PEBs:  %d\n", w.TotalPEBs))
	sb.WriteString(fmt.Sprintf("  Used:        %d (%.1f%%)\n", w.UsedPEBs, float64(w.UsedPEBs)/float64(w.TotalPEBs)*100))
	sb.WriteString(fmt.Sprintf("  Free:        %d (%.1f%%)\n", w.FreePEBs, float64(w.FreePEBs)/float64(w.TotalPEBs)*100))
	if w.BadPEBs > 0 {
		sb.WriteString(fmt.Sprintf("  Bad:         %d ⚠️\n", w.BadPEBs))
	}
	sb.WriteString(fmt.Sprintf("\n  Erase Counts:\n"))
	sb.WriteString(fmt.Sprintf("    Min:    %d\n", w.MinEC))
	sb.WriteString(fmt.Sprintf("    Max:    %d\n", w.MaxEC))
	sb.WriteString(fmt.Sprintf("    Avg:    %.1f\n", w.AvgEC))
	sb.WriteString(fmt.Sprintf("    Median: %d\n", w.MedianEC))
	sb.WriteString(fmt.Sprintf("    Spread: %d (max-min)\n", w.MaxEC-w.MinEC))

	// Health assessment
	health := "🟢 Good"
	if w.MaxEC-w.MinEC > 1000 {
		health = "🟡 Moderate wear imbalance"
	}
	if w.MaxEC-w.MinEC > 5000 || w.BadPEBs > 0 {
		health = "🔴 Significant wear / bad blocks"
	}
	sb.WriteString(fmt.Sprintf("\n  Health: %s\n", health))
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	return sb.String()
}
