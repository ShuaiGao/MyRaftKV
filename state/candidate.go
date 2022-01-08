package state

import "fmt"

type CandidateState struct {
	StateBase
	State StateEnum
}

func (c *CandidateState) Do() {
	fmt.Printf("Candidate DO")
	fmt.Println()
	c.SwitchTo(StateLeader)
}
