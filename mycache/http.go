//提供其他节点访问的能力（基于http）

package mycache

import (
	"fmt"
	"io/ioutil"
	"log"
	"mycache/consistenthash"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

// 默认路径，这里并没提供自定义路径，原因是http服务还能提供其他api服务
const (
	defaultBasePath = "/mycache/"
	defaultReplicas = 50
)

// HTTPPool实现结点间通信
type HTTPPool struct {
	//peer基础url，如https://example.net:8000
	self        string                 //包括主机名或IP和端口
	basePath    string                 //节点间通讯地址的前缀，默认是/mycache/
	mu          sync.Mutex             //互斥锁
	peers       *consistenthash.Map    //根据key选择节点
	httpGetters map[string]*httpGetter //映射远程节点与对应的httpGetter，因为httpGetter与远程节点的地址baseURL有关
}

// 构造实例
func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{
		self:     self,
		basePath: defaultBasePath,
	}
}

// 服务器日志打印
func (p *HTTPPool) Log(format string, v ...interface{}) {
	log.Printf("[Server %s] %s", p.self, fmt.Sprintf(format, v...))
}

// 处理所有的http请求。分割url得到groupName和key，之后查找对应key的值，返回缓存拷贝
func (p *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, p.basePath) {
		panic("HTTPPool serving unexpected path:" + r.URL.Path)
	}
	p.Log("%s %s", r.Method, r.URL.Path)
	parts := strings.SplitN(r.URL.Path[len(p.basePath):], "/", 2)
	if len(parts) != 2 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	groupName := parts[0]
	key := parts[1]

	group := GetGroup(groupName)
	if group == nil {
		http.Error(w, "no such group: "+groupName, http.StatusNotFound)
		return
	}

	view, err := group.Get(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(view.ByteSlice()) //等价于w.Write(view.b)因为写入http body不影响cache
}

// 实例化一致性哈希算法，并添加传入节点，并为每个节点创建一个HTTP客户端httpGetter
func (p *HTTPPool) Set(peers ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.peers = consistenthash.New(defaultReplicas, nil)
	p.peers.Add(peers...)
	p.httpGetters = make(map[string]*httpGetter, len(peers))
	for _, peer := range peers {
		p.httpGetters[peer] = &httpGetter{baseURL: peer + p.basePath}
	}
}

// 包装一致性哈希算法的Get方法，根据具体key,选择节点，返回节点对应的HTTP客户端
func (p *HTTPPool) PickPeer(key string) (PeerGetter, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if peer := p.peers.Get(key); peer != "" && peer != p.self {
		p.Log("Pick peer %s", peer)
		return p.httpGetters[peer], true
	}
	return nil, false
}

var _ PeerPicker = (*HTTPPool)(nil)

// http客户端类
type httpGetter struct {
	baseURL string //要访问的远程节点的地址，如http://example.com/mycache/
}

// 通过http.Get获取group对应key的哈希值，转为[]byte类型
func (h *httpGetter) Get(group string, key string) ([]byte, error) {
	u := fmt.Sprintf("%v%v%v", h.baseURL, url.QueryEscape(group), url.QueryEscape(key))
	res, err := http.Get(u)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned: %v", res.Status)
	}

	bytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %v", err)
	}

	return bytes, nil
}

var _ PeerGetter = (*httpGetter)(nil)
