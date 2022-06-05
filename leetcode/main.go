package main

import (
	"fmt"
	"leetcode/link"
)

func main() {
	test1 := link.ListConstruct(1)
	test1.Insert([]int{2, 3, 4, 5, 6})
	cur := test1
	for cur != nil {
		fmt.Printf("%d,", cur.Val)
		cur = cur.Next
	}
	fmt.Printf("\n")
	midTest1 := test1.FindMidLeft()
	fmt.Println(midTest1.Val)
	midTest2 := test1.FindMidRight()
	fmt.Println(midTest2.Val)
	head := test1.ReverseList()
	cur = head
	for cur != nil {
		fmt.Printf("%d,", cur.Val)
		cur = cur.Next
	}
	fmt.Printf("\n")
	findk := head.FindReverseK(2)
	fmt.Println(findk.Val)

}
