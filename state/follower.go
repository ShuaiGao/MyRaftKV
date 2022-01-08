package state

import "fmt"

type FollowerState struct {
	StateBase
	State StateEnum
}

func (f *FollowerState) Do() {
	fmt.Printf("Follower DO")
	fmt.Println()
	f.SwitchTo(StateCandidate)
}
