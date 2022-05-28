### 1. 为什么redis不使用C语言内置的char*字符串
- c语言内置的char *字符串是非二进制安全的，无法存储包含\0的字符串
- 获取长度复杂度为O(N),需要遍历
- 拷贝等操作需要先判断字符串长度，防止内存溢出
### 2. sds(simple dynamic string)数据结构
``` C
len; /* used */
alloc; /* excluding the header and null terminator */
flags; /* 3 lsb of type, 5 unused bits */
buf[];
```
- 其中包含了长度，已经分配的长度，和当前sds的类型，字符数组
### 3. sds优化
- 每次获取长度复杂度为O(1)
- 采用内存紧凑型布局，内部将类型分为 sdshdr5, sdshdr8, sdshdr16, sdshd32, sdshdr64
每一种类型都对应的不同的长度，举例sdshdr8
```C
struct __attribute__ ((__packed__)) sdshdr8 {
    uint8_t len; /* used */
    uint8_t alloc; /* excluding the header and null terminator */
    unsigned char flags; /* 3 lsb of type, 5 unused bits */
    char buf[];
};
```
头占用的长度为3个字节，最多可以存在256个字符，实现方式struct __attribute__ ((__packed__))不使用内存对其的方式

### 4. 总结
- sds相比于C语言的char * 字符串获取长度复杂度降低至O(1)
- 二进制安全，存储的字符串可以包含\0
- 字符串长度分配时会依据长度做内部处理，通过__attribute__ ((__packed__))节省内存
- 存储当前字符串长度和已经分配的字符串长度，方便字符串的追加，复制，比较等操作

###  5. 参考
- sds.c[https://github.com/redis/redis/blob/unstable/src/sds.h]
- sds.h[https://github.com/redis/redis/blob/unstable/src/sds.c]
- redis源码剖析与实战[https://time.geekbang.org/column/article/400314]