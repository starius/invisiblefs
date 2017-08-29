package sparse

import "testing"

func TestChain(t *testing.T) {
	c := newChain()
	steps := []struct {
		push, parent int64
	}{
		{1, 0},
		{2, 0},
		{3, 2},
		{4, 0},
		{5, 4},
		{6, 4},
		{7, 6},
		{8, 0},
		{9, 8},
		{10, 8},
		{11, 10},
		{12, 8},
		{13, 12},
		{14, 12},
		{15, 14},
		{16, 0},
		{17, 16},
		{18, 16},
		{19, 18},
		{20, 16},
		{21, 20},
	}
	for _, step := range steps {
		parent := c.parent()
		if parent.inOffsets != step.parent {
			t.Errorf("before pushing %d c.parent(): %#v, want %d", step.push, parent, step.parent)
		}
		c.push(chainElem{step.push, step.push})
	}
}
