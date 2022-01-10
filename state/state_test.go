package state

import (
	"fmt"
	"testing"
	"time"
)

func TestStateMachine_SwitchTo(t *testing.T) {
	machine := NewMachine()
	machine.register(StateLeader, &LeaderState{})
	machine.register(StateCandidate, &CandidateState{})
	machine.register(StateFollower, &FollowerState{})

	go func() {
		for {
			select {
			case rd := <-machine.Ready():
				time.Sleep(time.Second)
				fmt.Println("get msg " + rd.msg)
				if rd.msg != "" {
					fmt.Println("get msg " + rd.msg)
				}
			}
		}
	}()

	machine.StartAt(StateFollower)

	time.Sleep(10 * time.Second)
	machine.Stop()
}
