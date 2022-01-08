package raft

import "errors"

// Bootstrap 初始化RawNode，添加配置到其他工作节点，当存储数据非空时返回错误
// 建议应用程序手动引导它们的状态，而不是调用这个方法
// 方法是设置一个具有第一个索引的Storage，并将所需的ConfState存储为其InitialState
func (rn *RawNode) Bootstrap(peers []Peer) error {
	if len(peers) == 0 {
		return errors.New("must provide at least one peer to Bootstrap")
	}
	lastIndex, err := rn.raft.raftLog.storage.LastIndex()
	if err != nil {
		return err
	}
	if lastIndex != 0 {
		return errors.New("can't bootstrap a nonempty Storage")
	}
	//
	rn.preHardSt = emptyState
	return nil
}
