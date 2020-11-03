package trace

import "github.com/bwmarrin/snowflake"

var snowNode *snowflake.Node

// InitSnowNode
func InitSnowNode(n int64) (*snowflake.Node, error) {
	if snowNode == nil {
		node, err := snowflake.NewNode(n)
		if err != nil {
			return nil, err
		}
		snowNode = node
	}
	return snowNode, nil
}

// GenerateID
// n: workerId
func GenerateID(n ...int64) string {
	var workerId int64 = 0
	if len(n) > 0 {
		workerId = n[0]
	}
	node, err := InitSnowNode(workerId)
	if err != nil {
		return ""
	}

	// Generate a snowflake ID.
	idSnow := node.Generate()
	return idSnow.String()
}
