package state

import (
	"MyRaft/logger"
	"fmt"
	"time"
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
	ready   chan Ready
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

type Ready struct {
	msg string
}

func (sm *StateMachine) run() {
	//var ready chan struct{} = make(chan struct{})
	//var advance chan struct{} = make(chan struct{}, 1)
	var rd Ready
	//var rd Ready = Ready{msg: time.Second.String()}
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
		case sm.ready <- rd:
			fmt.Println("ready ")
			rd = Ready{}
		//case <-advance:
		//	fmt.Println("ready " + rd.msg)
		//	rd = Ready{}
		//	//advance = nil
		case <-time.After(time.Second):
			sm.empty()
			rd = Ready{msg: time.Now().String()}
		}
		//fmt.Println("loop")
		//time.Sleep(time.Millisecond)
	}
}

func (sm *StateMachine) Ready() <-chan Ready {
	return sm.ready
}

func (sm *StateMachine) empty() {
	fmt.Println("state machine empty ")
}

func NewMachine() *StateMachine {
	ins := &StateMachine{
		switchC: make(chan Switch, 1),
		close:   make(chan struct{}),
		ready:   make(chan Ready),
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
