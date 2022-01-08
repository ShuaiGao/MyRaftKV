package state

import "fmt"

type LeaderState struct {
	StateBase
	State StateEnum
}

func (l *LeaderState) Do() {
	fmt.Printf("Leader DO")
	fmt.Println()
	//l.SwitchTo(StateFollower)
}
