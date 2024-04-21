package mycache

import pb "mycache/mycachepb"

// 必须实现用来定位特定key对应的节点PeerGetter
type PeerPicker interface {
	//根据传入的key选择相应节点PeerGetter
	PickPeer(key string) (peer PeerGetter, ok bool)
}

// 必须实现用来查找缓存值的HTTP客户端
type PeerGetter interface {
	//从对应group查找缓存值，PeerGetter对应HTTP客户端
	Get(in *pb.Request, out *pb.Response) error
}
