package snowflake

import (
	"errors"
	"sync"
	"time"
)

const (
	epoch       = int64(1704067200000) // 2024-01-01 00:00:00 UTC
	sequenceBit = 12
	workerBit   = 10
	maxSequence = int64(1<<sequenceBit - 1)
	maxWorker   = int64(1<<workerBit - 1)
)

// Node 是一个进程内唯一的 ID 发生器，对外暴露的业务实体 ID 必须由此生成。
type Node struct {
	mu        sync.Mutex
	workerID  int64
	sequence  int64
	lastStamp int64
}

var defaultNode = New(1)

func New(workerID int64) *Node {
	if workerID < 0 || workerID > maxWorker {
		workerID = workerID & maxWorker
	}
	return &Node{workerID: workerID, lastStamp: -1}
}

// NextID 生成下一个全局 ID；时钟回拨时返回错误而非产出重复 ID。
func (n *Node) NextID() (int64, error) {
	n.mu.Lock()
	defer n.mu.Unlock()

	stamp := time.Now().UnixMilli()
	if stamp < n.lastStamp {
		return 0, errors.New("snowflake: clock moved backwards")
	}
	if stamp == n.lastStamp {
		n.sequence = (n.sequence + 1) & maxSequence
		if n.sequence == 0 {
			stamp = n.waitNextMillis(stamp)
		}
	} else {
		n.sequence = 0
	}
	n.lastStamp = stamp

	return (stamp-epoch)<<(sequenceBit+workerBit) | n.workerID<<sequenceBit | n.sequence, nil
}

func (n *Node) waitNextMillis(stamp int64) int64 {
	for stamp <= n.lastStamp {
		time.Sleep(time.Millisecond)
		stamp = time.Now().UnixMilli()
	}
	return stamp
}

// NextID 使用默认节点生成全局 ID。
func NextID() (int64, error) {
	return defaultNode.NextID()
}

// MustID 生成 ID，出错时回退为时间戳拼接，避免调用方因 ID 生成失败中断业务流程。
func MustID() int64 {
	id, err := defaultNode.NextID()
	if err != nil {
		return time.Now().UnixNano()
	}
	return id
}
