package snowflake

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/kyren223/eko/pkg/assert"
)

const (
	// Epoch is set to the twitter snowflake epoch of Nov 04 2010 01:42:54 UTC in milliseconds
	// TODO: change this to eko epoch when eko is production ready
	Epoch     int64 = 1288834974657
	nodeBits        = 10
	stepBits        = 12
	NodeMax         = 1<<nodeBits - 1
	nodeMask        = NodeMax << stepBits
	stepMask        = 1<<stepBits - 1
	timeShift       = nodeBits + stepBits
	nodeShift       = stepBits
)

type ID int64

func (id ID) String() string {
	return strconv.FormatInt(int64(id), 10)
}

func (id ID) Time() (msSinceEpoch int64) {
	return (int64(id) >> timeShift) + Epoch
}

func (id ID) Node() int64 {
	return int64(id) & nodeMask >> nodeShift
}

func (id ID) Step() int64 {
	return int64(id) & stepMask
}

type Node struct {
	mu    sync.Mutex
	epoch time.Time
	time  int64
	node  int64
	step  int64
}

func (n *Node) String() string {
	return fmt.Sprintf("Node%v(epoch: %v, step: %v, time: %v)", n.node, n.epoch, n.step, n.time)
}

// Creates a new node for generating unique IDs.
// To gurantee uniqueness, the given node ID must be unique.
func NewNode(node int64) *Node {
	assert.Assert(nodeBits+stepBits == 22, "node and step bits must add up to 22")
	assert.Assert(0 <= node && node <= NodeMax, "node must be within 0 and NodeMax", "node", node)

	// Credit to https://github.com/bwmarrin/snowflake
	currentTime := time.Now()
	// add time.Duration to curTime to make sure we use the monotonic clock if available
	epoch := currentTime.Add(time.Unix(Epoch/1000, (Epoch%1000)*1000000).Sub(currentTime))

	return &Node{
		node:  node,
		epoch: epoch,
	}
}

// Generates a unique snowflake ID.
func (n *Node) Generate() ID {
	n.mu.Lock()
	defer n.mu.Unlock()

	now := time.Since(n.epoch).Milliseconds()

	if now == n.time {
		n.step = (n.step + 1) & stepMask
		for n.step == 0 && now <= n.time {
			now = time.Since(n.epoch).Milliseconds()
		}
	} else {
		n.step = 0
	}

	n.time = now

	return ID((now << timeShift) | (n.node << nodeShift) | (n.step))
}

// Json marshling to avoid inprecision of json number (float64)

func (id ID) MarshalJSON() ([]byte, error) {
	buffer := make([]byte, 0, 22) // 2 quotes, 19 digits, 1 sign
	buffer = append(buffer, '"')
	buffer = strconv.AppendInt(buffer, int64(id), 10)
	buffer = append(buffer, '"')
	return buffer, nil
}

func (id *ID) UnmarshalJSON(data []byte) error {
	if len(data) < 3 || data[0] != '"' || data[len(data)-1] != '"' {
		return fmt.Errorf("invalid snowflake ID %q", string(data))
	}

	num, err := strconv.ParseInt(string(data[1:len(data)-1]), 10, 64)
	if err != nil {
		return err
	}

	*id = ID(num)
	return nil
}
