## 关于go语言的使用规范或者易踩坑

### 1. 采用defer释放锁，可能造成加锁时间变长，临界区的扩大
``` Go
临界区边界扩大，导致的加锁时间变长
func doDemo() {
    lock.Lock()
    defer lock.Unlock()
    //step1:临界区内的操作

    //step2:临界区外的操作
}

修改为
func doDemo() {
    func(){
        lock.Lock()
        defer lock.Unlock()
        //step1:临界区内的caozuo
    }（）
    //step2:临界区外的操作
}
```
### 2. 规范类操作
#### 2.1 命名规范
- 除测试文件_test包外，包名不应该包含下划线。应该为全部大写或者全部小写
- 