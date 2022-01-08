package state

import (
	"MyRaft/logger"
	"fmt"
)

type StateEnum int

const (
	StateFollower     StateEnum = iota // 正常工作状态
	StateCandidate                     // 选举中
	StatePreCandidate                  // 选举中
	StateLeader                        // leader状态
)

func (se StateEnum) String() string {
	switch se {
	case StateFollower:
		return "FOLLOWER"
	case StateCandidate:
		return "CANDIDATE"
	case StateLeader:
		return "LEADER"
	default:
		logger.Sugar().Errorf("unknown node state:%d", se)
		return "INVALID_STATE"
	}
}

type Machine interface {
	Do()
	SetStateMachine(sm *StateMachine)
	SwitchTo(enum StateEnum)
}
type Switch struct {
	state StateEnum
}
type StateMachine struct {
	switchC chan Switch
	close   chan struct{}
	stMap   map[StateEnum]Machine
	state   StateEnum
}
type StateBase struct {
	machine *StateMachine
}

func (sb *StateBase) SetStateMachine(sm *StateMachine) {
	sb.machine = sm
}
func (sb *StateBase) SwitchTo(enum StateEnum) {
	fmt.Printf("Machine from %s to %s", sb.machine.state, enum)
	fmt.Println()
	sb.machine.switchC <- Switch{state: enum}
}
func (sm *StateMachine) register(enum StateEnum, machine Machine) {
	sm.stMap[enum] = machine
	machine.SetStateMachine(sm)
}
func (sm *StateMachine) run() {
	for {
		select {
		case m := <-sm.switchC:
			if handler, ok := sm.stMap[m.state]; ok {
				sm.state = m.state
				handler.Do()
			} else {
				fmt.Printf("no this state handler: %s", m.state)
			}
		case <-sm.close:
			break
		}
		//time.Sleep(time.Millisecond)
	}
}

func NewMachine() *StateMachine {
	ins := &StateMachine{
		switchC: make(chan Switch, 1),
		close:   make(chan struct{}),
		stMap:   map[StateEnum]Machine{},
	}
	go ins.run()
	return ins
}

func (sm *StateMachine) Stop() {
	close(sm.close)
	sm.stMap = map[StateEnum]Machine{}
}
func (sm *StateMachine) StartAt(enum StateEnum) {
	if handler, ok := sm.stMap[enum]; ok {
		fmt.Printf("Machine start at %s", enum)
		fmt.Println()
		handler.Do()
	}
}
