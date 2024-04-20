package consistenthash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

// Hash函数
type Hash func(data []byte) uint32

// 一致性哈希的关键数据结构
type Map struct {
	hash     Hash           //hash函数
	replicas int            //虚拟节点倍数
	keys     []int          //哈希环
	hashMap  map[int]string //虚拟节点和真实节点映射表，键是虚拟节点哈希值，值是真实节点名称
}

// 自定义虚拟节点倍数和Hash函数
func New(replicas int, fn Hash) *Map {
	m := &Map{
		replicas: replicas,
		hash:     fn,
		hashMap:  make(map[int]string),
	}
	if m.hash == nil {
		m.hash = crc32.ChecksumIEEE //默认crc32哈希函数
	}
	return m
}

// 添加真实节点/机器
func (m *Map) Add(keys ...string) {
	for _, key := range keys {
		for i := 0; i < m.replicas; i++ {
			hash := int(m.hash([]byte(strconv.Itoa(i) + key))) //添加编号方式区分不同虚拟节点
			m.keys = append(m.keys, hash)                      //添加到环上
			m.hashMap[hash] = key                              //虚拟节点和真实节点映射
		}
	}
	sort.Ints(m.keys) //环上哈希值排序
}

// 选择节点
func (m *Map) Get(key string) string {
	if len(m.keys) == 0 {
		return ""
	}

	hash := int(m.hash([]byte(key))) //计算key的哈希值
	//从虚拟环中顺时针二分找到不小于哈希值的虚拟节点下标idx
	idx := sort.Search(len(m.keys), func(i int) bool {
		return m.keys[i] >= hash
	})
	//经映射找到真实节点，这里用到取模就是环状结构
	return m.hashMap[m.keys[idx%len(m.keys)]]
}
