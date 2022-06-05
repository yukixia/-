// 填充每个节点的下一个右侧节点指针 II
//Definition for a Node.

package link

type Node struct {
	Val   int
	Left  *Node
	Right *Node
	Next  *Node
}

func connect(root *Node) *Node {
	if root == nil {
		return root
	}
	start := root
	for start != nil {
		var nextStart, lastNode *Node
		handle := func(cur *Node) {
			if cur == nil {
				return
			}
			if nextStart == nil {
				nextStart = cur
			}
			if lastNode != nil {
				lastNode.Next = cur
			}
			lastNode = cur
		}
		for node := start; node != nil; node = node.Next {
			handle(node.Left)
			handle(node.Right)
		}
		start = nextStart
	}
	return root
}
