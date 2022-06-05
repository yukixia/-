/*
* 链表相关内容，主要包含链表基础操作
* 链表反转
* 链表找中间节点
* 链表
 */
package link

type ListNode struct {
	Val  int
	Next *ListNode
}

//构造链表头
func ListConstruct(val int) *ListNode {
	return &ListNode{Val: val}
}

//插入
func (head *ListNode) Insert(vals []int) *ListNode {
	cur := head
	for _, val := range vals {
		cur.Next = &ListNode{Val: val}
		cur = cur.Next
	}
	return head
}

//找中间节点,为偶数时偏左的节点
func (head *ListNode) FindMidLeft() *ListNode {
	if head == nil {
		return head
	}
	slow, fast := head, head
	for fast.Next != nil && fast.Next.Next != nil {
		slow = slow.Next
		fast = fast.Next.Next
	}
	return slow
}

//找中间节点，为偶数时偏右的节点
func (head *ListNode) FindMidRight() *ListNode {
	if head == nil {
		return head
	}
	slow, fast := head, head
	for fast.Next != nil {
		slow = slow.Next
		fast = fast.Next
		if fast.Next != nil {
			fast = fast.Next
		}
	}
	return slow

}

//反转链表
func (head *ListNode) ReverseList() *ListNode {
	var pre, cur *ListNode = nil, head
	for cur != nil {
		tmp := cur.Next
		cur.Next = pre
		pre = cur
		cur = tmp
	}
	return pre
}

//找到倒数第K个节点，其中k是从1开始计算
func (head *ListNode) FindReverseK(k int) *ListNode {
	//采用哑节点可以不用考虑空或者步数不对的情况
	dummyNode := &ListNode{}
	dummyNode.Next = head
	first, second := dummyNode, dummyNode
	for i := 0; i < k; i++ {
		first = first.Next
	}
	for first != nil {
		first = first.Next
		second = second.Next
	}
	return second
}
