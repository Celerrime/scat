package slots

type Slots chan slot
type slot struct{}

func New(n int) Slots {
	slots := make(Slots, n)
	for i, n := 0, cap(slots); i < n; i++ {
		slots <- slot{}
	}
	return slots
}

func (s Slots) Take() {
	<-s
}

func (s Slots) Release() {
	s <- slot{}
}
