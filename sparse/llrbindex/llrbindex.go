package llrbindex

import (
	"github.com/biogo/store/llrb"
)

type chunk struct {
	start     int64
	diskStart int64
	length    int64
}

func (c *chunk) Compare(than llrb.Comparable) int {
	d := c.start - than.(*chunk).start
	if d < 0 {
		return -1
	} else if d == 0 {
		return 0
	} else {
		return 1
	}
}

type Index struct {
	data llrb.Tree
}

func (i *Index) Read(start int64) (diskStart, sliceLength, gap int64) {
	it1 := i.data.Floor(&chunk{start, 0, 0})
	if it1 != nil {
		ch1 := it1.(*chunk)
		offset := start - ch1.start
		if offset < ch1.length {
			diskStart = ch1.diskStart + offset
			sliceLength = ch1.length - offset
		}
	}
	it2 := i.data.Ceil(&chunk{start + 1, 0, 0})
	if it2 == nil {
		gap = -1
	} else {
		ch2 := it2.(*chunk)
		gap = ch2.start - (start + sliceLength)
	}
	return
}

type guide chunk

func (c guide) Compare(than llrb.Comparable) int {
	ch := than.(*chunk)
	if ch.start+ch.length <= c.start {
		return 1
	} else if c.start+c.length <= ch.start {
		return -1
	} else {
		return 0
	}
}

func (i *Index) Write(start, diskStart, sliceLength int64) {
	var overlaps []*chunk
	i.data.DoMatching(func(elem llrb.Comparable) bool {
		overlaps = append(overlaps, elem.(*chunk))
		return false
	}, guide{start, 0, sliceLength})
	sliceBegin := start
	sliceEnd := start + sliceLength
	for _, ch := range overlaps {
		chunkBegin := ch.start
		chunkEnd := ch.start + ch.length
		if sliceEnd <= chunkBegin || chunkEnd <= sliceBegin {
			panic("No overlap")
		}
		leftOutside := chunkBegin < sliceBegin
		rightOutside := sliceEnd < chunkEnd
		if !leftOutside && !rightOutside {
			// slice: ******
			// chunk:  ----
			i.data.Delete(ch)
		} else if leftOutside && rightOutside {
			// slice:  ****
			// chunk: ------
			// Reuse ch as new left as they have the same start.
			ch.length = sliceBegin - chunkBegin
			deltaToRight := sliceEnd - chunkBegin
			i.data.Insert(&chunk{
				start:     sliceEnd,
				diskStart: ch.diskStart + deltaToRight,
				length:    chunkEnd - sliceEnd,
			})
		} else if leftOutside && !rightOutside {
			// slice:  ****
			// chunk: ---
			//        A
			ch.length = sliceBegin - chunkBegin
		} else if !leftOutside && rightOutside {
			// slice: *****
			// chunk:   ----
			//             C
			delta := sliceEnd - chunkBegin
			i.data.Insert(&chunk{
				start:     ch.start + delta,
				diskStart: ch.diskStart + delta,
				length:    ch.length - delta,
			})
			i.data.Delete(ch)
		}
	}
	// Put the slice into the map.
	i.data.Insert(&chunk{
		start:     start,
		diskStart: diskStart,
		length:    sliceLength,
	})
	// TODO: merge with the left neighbour even if possible.
}
