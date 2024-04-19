//缓存值的抽象和封装

package mycache

// 以便支持任意类似能够，如字符串、图片等
type ByteView struct {
	b []byte
}

// lru.Cache中要求缓存对象必须实现Value接口，即Len() int 方法，返回所占内存大小
func (v ByteView) Len() int {
	return len(v.b)
}

// b只读，用ByteSlice返回拷贝，以免缓存值被外部程序修改
func (v ByteView) ByteSlice() []byte {
	return cloneBytes(v.b)
}

// 拷贝数据
func cloneBytes(b []byte) []byte {
	c := make([]byte, len(b))
	copy(c, b)
	return c
}

// 转换为string，如果有必要则返回拷贝
func (v ByteView) String() string {
	return string(v.b)
}
