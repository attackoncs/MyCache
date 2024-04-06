package lru

import "container/list"

// 缓存结构体定义,非并发安全
type Cache struct {
	maxBytes  int64                         //最大内存
	nbytes    int64                         //已使用内存
	ll        *list.List                    //双向链表
	cache     map[string]*list.Element      //字符串和双向链表节点的键值对缓存
	onEvicted func(key string, value Value) //记录被移除时的回调函数
}

// 键值对entry是双向链表节点的数据类型
type entry struct {
	key   string
	value Value
}

// 通用性，只要实现Len()的类型都可作为值
type Value interface {
	Len() int
}

// 构造实例
func New(maxBytes int64, onEvicted func(key string, value Value)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		nbytes:    0,
		ll:        list.New(),
		cache:     make(map[string]*list.Element),
		onEvicted: onEvicted,
	}
}

// 查找，从字典中找到对应双向链表节点，并且将该节点移动到队尾
func (c *Cache) Get(key string) (value Value, ok bool) {
	if element, ok := c.cache[key]; ok {
		c.ll.MoveToFront(element)
		//将接口类型list.Element转换为*entry类型
		pair := element.Value.(*entry)
		return pair.value, true
	}
	return
}

// 删除，删除链表队首，并且删除节点对应的键值对，修改nbytes并判断是否执行回掉
func (c *Cache) RemoveOldest() {
	element := c.ll.Back()
	if element != nil {
		c.ll.Remove(element)
		kv := element.Value.(*entry)
		delete(c.cache, kv.key)
		c.nbytes -= int64(len(kv.key)) + int64(kv.value.Len())
		if c.onEvicted != nil {
			c.onEvicted(kv.key, kv.value)
		}
	}
}

// 新增或修改
func (c *Cache) Add(key string, value Value) {
	if element, ok := c.cache[key]; ok { //修改
		c.ll.MoveToFront(element)
		kv := element.Value.(*entry)
		c.nbytes += int64(value.Len()) - int64(kv.value.Len())
		kv.value = value
	} else { //新增
		ele := c.ll.PushFront(&entry{key: key, value: value})
		c.cache[key] = ele
		c.nbytes += int64(len(key)) + int64(value.Len())
	}
	//这里要不断遍历直到不超过最大内存
	for c.maxBytes != 0 && c.maxBytes < c.nbytes {
		c.RemoveOldest()
	}
}

// cache的数目
func (c *Cache) Len() int {
	return c.ll.Len()
}
