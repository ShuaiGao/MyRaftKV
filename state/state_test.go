package state

import (
	"testing"
	"time"
)

func TestStateMachine_SwitchTo(t *testing.T) {
	machine := NewMachine()
	machine.register(StateLeader, &LeaderState{})
	machine.register(StateCandidate, &CandidateState{})
	machine.register(StateFollower, &FollowerState{})
	machine.StartAt(StateFollower)

	time.Sleep(2 * time.Second)
	machine.Stop()
}
