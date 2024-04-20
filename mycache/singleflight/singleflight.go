package singleflight

import "sync"

// 正在进行中，或已经结束的请求
type call struct {
	wg  sync.WaitGroup //WaitGroup锁避免重入
	val interface{}
	err error
}

// 管理不同key的请求
type Group struct {
	mu sync.Mutex //保护map，不过这里应该可用sync.Map代替普通map
	m  map[string]*call
}

// 针对相同的key,无论Do被调用多少次，函数fn都只会被调用一次，等待fn调用返回或错误
func (g *Group) Do(key string, fn func() (interface{}, error)) (interface{}, error) {
	g.mu.Lock()
	if g.m == nil {
		g.m = make(map[string]*call)
	}
	if c, ok := g.m[key]; ok {
		g.mu.Unlock()
		c.wg.Wait()         //若请求正在进行中，则等待直到锁被释放
		return c.val, c.err //请求结束，返回结果
	}
	c := new(call)
	c.wg.Add(1)  //发起请求前加锁
	g.m[key] = c //添加到g.m,表明key已经有对应的请求在处理
	g.mu.Unlock()

	c.val, c.err = fn() //调用fn发起前请求
	c.wg.Done()         //锁减1

	g.mu.Lock()
	delete(g.m, key) //更新g.m
	g.mu.Unlock()    //返回结果

	return c.val, c.err
}
