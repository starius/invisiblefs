package sparse

import "fmt"

type chainElem struct {
	inOffsets int64 // Start in s.offsets.
	inData    int64 // Its &(this.parent_ptr).
}

type chain struct {
	// Each element is latest entry in offsets whose 1-based index is
	// divisible by 2 ^ (its_0_based_index_in_chain + 1).
	chain   []chainElem
	nPushed int
}

func countZeroBits(i int) int {
	if i == 0 {
		panic("countZeroBits(0)")
	}
	n := 0
	for i&1 == 0 {
		n++
		i >>= 1
	}
	return n
}

func (c *chain) push(e chainElem) {
	c.nPushed++
	affected := countZeroBits(c.nPushed)
	if len(c.chain) < affected {
		if len(c.chain) != affected-1 {
			panic("c.chain too short")
		}
		c.chain = append(c.chain, chainElem{})
	}
	for i := 0; i < affected; i++ {
		c.chain[i] = e
	}
}

func (c *chain) parent() chainElem {
	if c.nPushed == 0 {
		return chainElem{}
	}
	i := countZeroBits(c.nPushed + 1)
	if i > len(c.chain)+1 {
		panic("too large i")
	} else if i == len(c.chain)+1 {
		return chainElem{}
	} else if i == len(c.chain) {
		panic("i == len(c.chain)")
	} else {
		return c.chain[i]
	}
}

func newChain() *chain {
	return &chain{}
}

func restore(elements []chainElem, sizes []int) (*chain, error) {
	// Items in elements and sizes are ordered by size decreasing.
	if len(elements) != len(sizes) {
		return nil, fmt.Errorf("sizes mismatch")
	}
	var c []chainElem
	i := len(sizes) - 1
	if sizes[i] == 1 {
		i--
	}
	size := 2
	for i >= 0 {
		for size <= sizes[i] {
			size *= 2
			c = append(c, elements[i])
		}
		if size != sizes[i]*2 {
			return nil, fmt.Errorf("size is not a power of 2")
		}
		i--
	}
	nPushed := 0
	for _, size := range sizes {
		nPushed += size
	}
	return &chain{
		chain:   c,
		nPushed: nPushed,
	}, nil
}
