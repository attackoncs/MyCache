//提供其他节点访问的能力（基于http）

package mycache

import (
	"fmt"
	"log"
	"net/http"
	"strings"
)

// 默认路径，这里并没提供自定义路径，原因是http服务还能提供其他api服务
const defaultBasePath = "/mycache/"

// HTTPPool实现结点间通信
type HTTPPool struct {
	//peer基础url，如https://example.net:8000
	self     string //包括主机名或IP和端口
	basePath string //节点间通讯地址的前缀，默认是/mycache/
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
