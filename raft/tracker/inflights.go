package tracker

type Inflights struct {
	start  int
	count  int
	size   int
	buffer []uint64
}

func NewInflights(size int) *Inflights {
	return &Inflights{
		size: size,
	}
}

func (in *Inflights) Clone() *Inflights {
	ins := *in
	ins.buffer = append([]uint64(nil), in.buffer...)
	return &ins
}
func (in *Inflights) Add(inflight uint64) {
	if in.Full() {
		panic("cannot add into a Full inflights")
	}
	next := in.start + in.count
	size := in.size
	if next >= size {
		next -= size
	}
	if next >= len(in.buffer) {
		in.grow()
	}
	in.buffer[next] = inflight
	in.count++
}
func (in *Inflights) grow() {
	newSize := len(in.buffer) * 2
	if newSize == 0 {
		newSize = 1
	} else if newSize > in.size {
		newSize = in.size
	}
	newBuffer := make([]uint64, newSize)
	copy(newBuffer, in.buffer)
	in.buffer = newBuffer
}

// FreeLE 释放小于或等于to的flight数据
func (in *Inflights) FreeLE(to uint64) {
	if in.count == 0 || to < in.buffer[in.start] {
		return
	}
	idx := in.start
	var i int
	for i = 0; i < in.count; i++ {
		if to < in.buffer[idx] {
			break
		}
		size := in.size
		if idx++; idx >= size {
			idx -= size
		}
	}
	in.count -= i
	in.start = idx
	if in.count == 0 {
		in.start = 0
	}
}
func (in *Inflights) FreeFirstOne() {
	in.FreeLE(in.buffer[in.start])
}
func (in *Inflights) Full() bool {
	return in.count == in.size
}
func (in *Inflights) Count() int {
	return in.count
}
func (in *Inflights) reset() {
	in.count = 0
	in.start = 0
}
