//与外部交互，控制缓存存储和获取的主流程

package mycache

import (
	"fmt"
	"log"
	"sync"
)

// 缓存命名空间，负责与用户的交互，控制缓存值存储和获取的流程
/*
接收 key --> 检查是否被缓存 -----> 返回缓存值 ⑴
|  否                         是
|-----> 是否应当从远程节点获取 -----> 与远程节点交互 --> 返回缓存值 ⑵
			  |  否
			  |-----> 调用`回调函数`，获取值并添加到缓存 --> 返回缓存值 ⑶
*/
type Group struct {
	name      string     //缓存名称
	getter    Getter     //缓存未命中时获取源数据的回掉
	mainCache cache      //实现的并发缓存
	peers     PeerPicker //定位节点对应特定key的HTTP客户端
}

// 不支持多数据源配置，由用户决定数据获取，设计回掉函数，缓存不存在时，调用回掉得到源数据
// 接口式函数或函数式接口，http包中的HandlerFunc,实现ServeHTTP方法，然后用作普通函数包装
// 参数既可以是接口，又可以是函数（因为函数本身也实现了这个接口）。这种写法有一个前提：即接口内部只定义了一个方法。
type Getter interface {
	Get(key string) ([]byte, error)
}

// GetterFunc函数类型，
// 定义一个函数类型 F，并且实现接口 A 的方法，然后在这个方法中调用自己。这是 Go 语言中将其他函数
// （参数返回值定义与 F 一致）转换为接口 A 的常用技巧。
type GetterFunc func(key string) ([]byte, error)

// 回掉函数，函数类型实现Getter接口Get方法，方便调用时既能传入函数作为参数，也能传入实现该接口的结构体作为参数
func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}

var (
	mu     sync.RWMutex              //全局读写锁
	groups = make(map[string]*Group) //全局groups
)

// 创建group实例，并存储在全局变量groups
func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	if getter == nil {
		panic("nil Getter")
	}
	mu.Lock()
	defer mu.Unlock()
	g := &Group{
		name:      name,
		getter:    getter,
		mainCache: cache{cacheBytes: cacheBytes},
	}
	groups[name] = g //同样严格来说只需对map进行加解锁
	return g
}

// 获取对应名的group，使用只读锁，不涉及冲突变量的写操作
func GetGroup(name string) *Group {
	mu.RLock()
	g := groups[name]
	mu.RUnlock()
	return g
}

// 从group中获取key对应的缓存
func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key is required")
	}

	if v, ok := g.mainCache.get(key); ok {
		log.Printf("[mycache] hit")
		return v, nil
	}
	return g.load(key)
}

// 注册节点，用于选择远程对等节点
func (g *Group) RegisterPeers(peers PeerPicker) {
	if g.peers != nil {
		panic("RegisterPeerPicker called more than once")
	}
	g.peers = peers
}

// 分布式场景下，load会从远程节点获取getFromPeer,失败了再会推到getLocally。
func (g *Group) load(key string) (value ByteView, err error) {
	if g.peers != nil {
		if peer, ok := g.peers.PickPeer(key); ok {
			if value, err = g.getFromPeer(peer, key); err == nil {
				return value, nil
			}
			log.Println("[MyCache] Failed to get from peer", err)
		}
	}

	return g.getLocally(key)
}

// PeerGetter接口的httpGetter从访问远程节点获取对应group和key的缓存
func (g *Group) getFromPeer(peer PeerGetter, key string) (ByteView, error) {
	bytes, err := peer.Get(g.name, key)
	if err != nil {
		return ByteView{}, err
	}
	return ByteView{b: bytes}, nil
}

// 从本地获取缓存，注意cloneBytes这里，切片不会被深拷贝，bytes是返回的切片
func (g *Group) getLocally(key string) (ByteView, error) {
	bytes, err := g.getter.Get(key)
	if err != nil {
		return ByteView{}, err
	}
	value := ByteView{b: cloneBytes(bytes)}
	g.populateCache(key, value)
	return value, nil
}

// 将源数据添加到缓存mainCache中
func (g *Group) populateCache(key string, value ByteView) {
	g.mainCache.add(key, value)
}
