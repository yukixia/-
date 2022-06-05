## reactor模型在redis中的使用
### 1. 介绍
reactor模型为网络服务器端用来处理高并发网络IO的一种编程模型
reactor包含三类事件和三个角色
- 三类事件: 连接事件，读事件，写事件
连接事件：客户端和服务器进行交互，客户端会向服务器发起一个连接请求，此时对应一个连接事件
写事件：连接事件建立以后，客户端会向服务器发送读请求，以便读取数据，服务器向客户端写回数据，对应一个写请求
读事件：无论是客户端发送读请求还是写请求，服务器端都需要从客户端读取请求内容，对应服务器端的读请求
- 三个角色： acceptor, reactor, handler
客户端向服务器端发起的连接请求，是由acceptor进行处理
产生的读写请求，交由handler处理
连接事件，读写事件同时发生时，需要一个角色专门监听和分配事件，此角色为reactor角色， 负责三类事件的监听和分配，将连接
事件交由acceptor处理，将读写事件交由handler处理
这三类角色围绕着事件的监听，分发和处理来进行交互，三者之前的交互流程离不开事件驱动框架
### 2. 事件驱动框架
事件驱动框架包含两部分，第一部分为事件的初始化，第二部分为事件的捕获，分发和处理
reactor模型的基本工作流程为：客户端的不同请求会在服务器端触发连接，读，写三类事件。这三类事件的捕获，分发和处理交由reactor,
acceptor, handler进行完成。这三类角色会通过事件驱动框架进行交互和事件处理
### 3 redis对事件驱动框架的实现
redis对事件驱动框架的代码在ae.c, 和ae.h两个文件中
#### 3.1 事件的定义， redis定义了两类事件，IO事件和时间事件
- rfileProc, wfileProc函数对应了事件处理框架中的handler函数，用于处理读写事件请求
```C
/* File event structure */
typedef struct aeFileEvent {
    int mask; /* one of AE_(READABLE|WRITABLE|BARRIER) */
    aeFileProc *rfileProc;
    aeFileProc *wfileProc;
    void *clientData;
} aeFileEvent;
```
#### 3.2 事件处理主循环，aeMain, 它在redis main函数中，当完成了服务器初始化之后被调用
```C
void aeMain(aeEventLoop *eventLoop) {
    eventLoop->stop = 0;
    while (!eventLoop->stop) {
        aeProcessEvents(eventLoop, AE_ALL_EVENTS|
                                   AE_CALL_BEFORE_SLEEP|
                                   AE_CALL_AFTER_SLEEP);
    }
}
```
#### 3.3 事件的捕获和分发
在aeMain中我们可以看到调用了aeProcessEvents函数，此函数负责处理事件的捕获和分发，相当于reactor的角色
从代码中可以看到该函数主体由三个if分支组成，最终返回处理的事件数量
- 既没有IO事件也没有事件事件，则直接返回
- 存在IO事件或者紧急的事件事件
- 普通事件事件，交由processTimeEvents
```C
int aeProcessEvents(aeEventLoop *eventLoop, int flags)
{
    int processed = 0, numevents;

    /* Nothing to do? return ASAP */
    if (!(flags & AE_TIME_EVENTS) && !(flags & AE_FILE_EVENTS)) return 0;

    /* Note that we want to call select() even if there are no
     * file events to process as long as we want to process time
     * events, in order to sleep until the next time event is ready
     * to fire. */
    if (eventLoop->maxfd != -1 ||
        ((flags & AE_TIME_EVENTS) && !(flags & AE_DONT_WAIT))) {
        ...

        /* Call the multiplexing API, will return only on timeout or when
         * some event fires. */
        numevents = aeApiPoll(eventLoop, tvp);
        ....

    }
    /* Check time events */
    if (flags & AE_TIME_EVENTS)
        processed += processTimeEvents(eventLoop);

    return processed; /* return the number of processed file/time events */
}
```

其中第二类情况，存在IO事件或者紧急的事件事件，调用aeApiPoll进行事件的捕获。aeApiPoll函数底层依赖于操作系统的IO多路复用
技术。redis针对于不同的操作系统采用不同的多路复用，具体为
``` C
/* Include the best multiplexing layer supported by this system.
 * The following should be ordered by performances, descending. */
#ifdef HAVE_EVPORT
#include "ae_evport.c" //Solaris
#else
    #ifdef HAVE_EPOLL
    #include "ae_epoll.c" //linux
    #else
        #ifdef HAVE_KQUEUE 
        #include "ae_kqueue.c" //macos
        #else
        #include "ae_select.c" //windows
        #endif
    #endif
#endif
```
#### 3.4 事件和处理函数的注册绑定
当redis启动后，服务器main函数会调用initServer来进行初始化，在初始化过程中调用aeCreateFileEvent进行事件的注册和对应的处理函数,其中AE_READABLE为注册的事件，而对应的处理函数为accept_handler（即acceptTcpHandler）
```C
/* Create an event handler for accepting new connections in TCP or TLS domain sockets.
 * This works atomically for all socket fds */
int createSocketAcceptHandler(socketFds *sfd, aeFileProc *accept_handler) {
    int j;

    for (j = 0; j < sfd->count; j++) {
        if (aeCreateFileEvent(server.el, sfd->fd[j], AE_READABLE, accept_handler,NULL) == AE_ERR) {
            /* Rollback */
            for (j = j-1; j >= 0; j--) aeDeleteFileEvent(server.el, sfd->fd[j], AE_READABLE);
            return C_ERR;
        }
    }
    return C_OK;
}
```
那么aeCreateFileEvent是如何实现事件和处理函数的注册的呢，则仍然依赖于操作系统，以linux为例，epoll_ctl函数用于新增注册事件。而redis对此函数进行了封装，即函数aeApiAddEvent

### 4.总结
redis实现了事件驱动框架，能够同时处理多个客户端请求，实现了高性能的网络框架，通过事件驱动框架，redis可以通过一个循环不断捕获请求，处理来自客户端的连接事件，读写请求。具体关键函数如下
- aeMain函数负责主循环，在server启动时被调用
- aeProcessEvent负责不同事件的ch，在u 里aeMain中调用
- aeApiPoll调用操作系统的IO多路复用，进行事件的捕获。在aeProcessEvent中被调用
- epoll_wait检测并返回内核中发生的事件，在aeApiPoll中被调用
至此我们了解了redis高效处理客户端请求的具体实现，借助reactor的编程模型，依赖IO多路复用技术实现了事件驱动框架。由此实现了高并发处理客户端请求