### 1. hash的数据结构
```C
struct dict {
    dictType *type;

    dictEntry **ht_table[2];
    unsigned long ht_used[2];

    long rehashidx; /* rehashing not in progress if rehashidx == -1 */

    /* Keep small vars at end for optimal (minimal) struct padding */
    int16_t pauserehash; /* If >0 rehashing is paused (<0 indicates coding error) */
    signed char ht_size_exp[2]; /* exponent of size. (size = 1<<exp) */
};

typedef struct dictEntry {
    void *key;
    union {
        void *val;
        uint64_t u64;
        int64_t s64;
        double d;
    } v;
    struct dictEntry *next;     /* Next entry in the same hash bucket. */
    void *metadata[];           /* An arbitrary number of bytes (starting at a
                                 * pointer-aligned address) of size as returned
                                 * by dictType's dictEntryMetadataBytes(). */
} dictEntry;
```
### 2. hash键值对的插入操作之链表法解决冲突
hash的键值插入函数为[dictAddRaw](https://github.com/redis/redis/blob/unstable/src/dict.c),在hash插入操作中可以看到进行了rehash，以及hash冲突时采用的解决办法（即链表法）。流程图如下
![hashAdd](/redis/img/hashAdd.jpeg)
主要代码
```C
dictEntry *dictAddRaw(dict *d, void *key, dictEntry **existing)
{
    long index;
    dictEntry *entry;
    int htidx;

    if (dictIsRehashing(d)) _dictRehashStep(d);

    /* Get the index of the new element, or -1 if
     * the element already exists. */
    if ((index = _dictKeyIndex(d, key, dictHashKey(d,key), existing)) == -1)
        return NULL;
    ...
    entry->next = d->ht_table[htidx][index]; //链接地址法解决hash冲突，刚刚插入的被放在链表头
    d->ht_table[htidx][index] = entry;
    d->ht_used[htidx]++;

    /* Set the hash entry fields. */
    dictSetKey(d, entry, key);
    return entry;
}
```
### 3. rehash操作
rehash是在hash表数据量过大，造成查询缓慢时进行的hash表扩容操作，因此需要考虑如下几个问题
- rehash操作何时触发
- rehash扩容扩多大
- rehash怎么执行

#### 3.1 rehash操作何时触发
rehash扩容判断发生在函数_dictExpandIfNeeded中，该函数中判断元素数量已经达到了1:1的比例且此时允许进行rehash操作
或者元素数量已经达到了5:1的比例，则会进行rehash操作，具体代码如下
```C 
/* Expand the hash table if needed */
static int _dictExpandIfNeeded(dict *d)
{
    /* Incremental rehashing already in progress. Return. */
    if (dictIsRehashing(d)) return DICT_OK;

    /* If the hash table is empty expand it to the initial size. */
    if (DICTHT_SIZE(d->ht_size_exp[0]) == 0) return dictExpand(d, DICT_HT_INITIAL_SIZE);

    /* If we reached the 1:1 ratio, and we are allowed to resize the hash
     * table (global setting) or we should avoid it but the ratio between
     * elements/buckets is over the "safe" threshold, we resize doubling
     * the number of buckets. */
    if (d->ht_used[0] >= DICTHT_SIZE(d->ht_size_exp[0]) &&
        (dict_can_resize ||
         d->ht_used[0]/ DICTHT_SIZE(d->ht_size_exp[0]) > dict_force_resize_ratio) &&
        dictTypeExpandAllowed(d))
    {
        return dictExpand(d, d->ht_used[0] + 1);
    }
    return DICT_OK;
}
```
其中dict_can_resize代表是否能够进行rehash操作，在代码中对该变量的修改来自于如下两个函数
```C
void dictEnableResize(void) {
    dict_can_resize = 1;
}

void dictDisableResize(void) {
    dict_can_resize = 0;
}
```
调用这两个函数的地方为updateDictResizePolicy函数，在进行AOF或者RDB过程中有子进程时则会将dict_can_resize设置为0
具体原因正如代码注释中所描述的在进行RDB或者AOF重写时，redis采用的是写时拷贝（copy on write)[<sup>1</sup>](#refer-anchor-1), 如果在这个时候进行rehash操作则会产生大量的内存页拷贝，因此在进行fork子进程时会避免进行rehash操作

rehash的触发条件弄清楚之后，我们还需要一下函数_dictExpandIfNeeded的调用时间点，_dictKeyIndex计算哈希索引调用了
_dictExpandIfNeeded,而_dictKeyIndex又被dictAddRaw函数调用，调用dictAddRaw有如下几个函数
- dictAdd 往哈希表中插入元素
- dictReplace 往哈希表中插入或者替换元素
- dictAddOrFind 在哈希表表中查找（不存在时插入）元素
具体关系如下图
![dictExpandIfNeeded](/redis/img/dictExpandIfNeed.jpeg)
#### 3.2 rehash扩容扩多大
在最新的redis代码rehash扩容是将当前的size进行加一操作dictExpand(d, d->ht_used[0] + 1)，采用逐渐找到最接近当前size的数量的容量进行扩容
#### 3.3 rehash怎么执行
redis为什么需要进行渐进式rehash操作而非直接一次性迁移，则是因为迁移操作发生在redis主进程中，在迁移过程中redis是无法
进行客户端请求处理的。关于redis执行过程有两个重要的函数_dictRehashStep和dictRehash
- _dictRehashStep 每次迁移的单位，目前为固定值1，即每次迁移一个桶的元素
- dictRehash 为真正进行哈希元素迁移的函数，主要分为两部分
    - 查找待迁移的桶,并进行迁移，其中设置了一个empty_visited该值初始化为10，如果连续10个桶都为空值，则会停止本次rehash操作。否则进行hash元素迁移操作。迁移时将原来桶内元素减1，新桶内元素加1
    ```C
     while(n-- && d->ht_used[0] != 0) {
        dictEntry *de, *nextde;

        /* Note that rehashidx can't overflow as we are sure there are more
         * elements because ht[0].used != 0 */
        assert(DICTHT_SIZE(d->ht_size_exp[0]) > (unsigned long)d->rehashidx);
        while(d->ht_table[0][d->rehashidx] == NULL) {
            d->rehashidx++;
            if (--empty_visits == 0) return 1;
        }
        de = d->ht_table[0][d->rehashidx];
        /* Move all the keys in this bucket from the old to the new hash HT */
        while(de) {
            uint64_t h;

            nextde = de->next;
            /* Get the index in the new hash table */
            h = dictHashKey(d, de->key) & DICTHT_SIZE_MASK(d->ht_size_exp[1]);
            de->next = d->ht_table[1][h];
            d->ht_table[1][h] = de;
            d->ht_used[0]--;
            d->ht_used[1]++;
            de = nextde;
        }
        d->ht_table[0][d->rehashidx] = NULL;
        d->rehashidx++;
    }
    ```
    - 判断是否迁移完成，完成则将进行扫尾工作,主要为hash
    ```C
    if (d->ht_used[0] == 0) {
        zfree(d->ht_table[0]);
        /* Copy the new ht onto the old one */
        d->ht_table[0] = d->ht_table[1];
        d->ht_used[0] = d->ht_used[1];
        d->ht_size_exp[0] = d->ht_size_exp[1];
        _dictReset(d, 1);
        d->rehashidx = -1;
        return 0;
    }
    ```
### 4. 总结
本小节主要内容为redis 哈希数据结构，通过数据结构可以了解到每个dictEntry中包含一个next指针用来指向落在相同哈希槽中的元素，即采用链地址解决哈希冲突。 哈希dict的结构中可以发现定义了两个哈希表，用来做哈希表的扩容和迁移。此外hash表的迁移触发时机为元素数量达到1:1且此时无任何子进程在运行，或者达到了5:1的比例，则会触发rehash操作，为了尽可能减少对客户端响应的影响，rehash操作采用渐进式迁移方式，每次迁移一定的数量key,在rehash期间通过rehashidx标示是否进行rehash操作。
### 5. 参考
<div id="refer-anchor-1"></div>

- [1] [copy on wirte](https://stackoverflow.com/questions/628938/what-is-copy-on-write)
- [2] [极客时间如何实现一个性能优异的Hash表](https://time.geekbang.org/column/article/400379)
