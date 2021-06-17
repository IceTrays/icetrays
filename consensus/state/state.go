package state

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/icetrays/icetrays/consensus/pb"
	"github.com/icetrays/icetrays/datastore"
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	ds "github.com/ipfs/go-datastore"
	badger "github.com/ipfs/go-ds-badger"
	format "github.com/ipfs/go-ipld-format"
	"github.com/ipfs/go-merkledag"
	"github.com/ipfs/go-mfs"
	"github.com/ipfs/go-unixfs"
	iface "github.com/ipfs/interface-go-ipfs-core"
	"github.com/ipfs/interface-go-ipfs-core/path"
	"io"
	"io/ioutil"
	"os"
	gopath "path"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var ErrParamsNum = errors.New("params num error")

const PinCid = "pinCid"
const PinPeer = "pinPeer"

type FileTreeState struct {
	ID          string
	dag         format.DAGService
	pin         iface.PinAPI
	fs          iface.UnixfsAPI
	root        *mfs.Root
	store       datastore.StateDB
	ctx         context.Context
	once        sync.Once
	index       uint64
	mtx         sync.Mutex
	PreExecuted bool
	ipfsDb      *badger.Datastore
}

func (fts *FileTreeState) Execute(ins *pb.Instruction) error {
	switch ins.GetCode() {
	case pb.Instruction_CP:
		return fts.cp(ins.GetNode(), ins.GetParams()...)
	case pb.Instruction_MV:
		return fts.Mv(ins.GetParams()...)
	case pb.Instruction_RM:
		return fts.Rm(ins.GetParams()...)
	case pb.Instruction_MKDIR:
		return fts.Mkdir(ins.GetParams()...)
	case pb.Instruction_Pin:
		return fts.PinCidFile(ins.GetParams()...)
	default:
		return errors.New("unrecognized operation")
	}
}

func (fts *FileTreeState) Ls(ctx context.Context, path string) ([]mfs.NodeListing, error) {
	fsn, err := mfs.Lookup(fts.root, path)
	if err != nil {
		return nil, err
	}
	switch fsn := fsn.(type) {
	case *mfs.Directory:
		return fsn.List(ctx)
	case *mfs.File:
		_, name := gopath.Split(path)
		node := make([]mfs.NodeListing, 1)
		node[0] = mfs.NodeListing{
			Name: name,
		}
		if size, err := fsn.Size(); err == nil {
			node[0].Size = size
		}
		if nd, err := fsn.GetNode(); err == nil {
			node[0].Hash = nd.Cid().String()
		}
		return node, nil
	default:
		return nil, errors.New("unrecognized type")
	}
}

func (fts *FileTreeState) resolvePath(path string, nodeData []byte) (format.Node, error) {
	if len(path) > 0 && path[0] == '/' {
		fsNode, err := mfs.Lookup(fts.root, path)
		if err != nil {
			return nil, err
		}
		return fsNode.GetNode()
	}
	c, err := cid.Decode(path)
	if err != nil {
		return nil, err
	}
	blk, err := blocks.NewBlockWithCid(nodeData, c)
	if err != nil {
		return nil, err
	}

	return format.DefaultBlockDecoder.Decode(blk)
}

func (fts *FileTreeState) cp(nodeData []byte, params ...string) error {
	if len(params) != 2 {
		return ErrParamsNum
	}
	node, err := fts.resolvePath(params[1], nodeData)
	if err != nil {
		return err
	}
	return mfs.PutNode(fts.root, params[0], node)
}

func (fts *FileTreeState) Mv(params ...string) error {
	if len(params) != 2 {
		return ErrParamsNum
	}
	src, err := checkPath(params[0])
	if err != nil {
		return err
	}
	dst, err := checkPath(params[1])
	if err != nil {
		return err
	}
	return mfs.Mv(fts.root, src, dst)
}

func (fts *FileTreeState) Mkdir(params ...string) error {
	if len(params) != 1 {
		return ErrParamsNum
	}
	src, err := checkPath(params[0])
	if err != nil {
		return err
	}
	return mfs.Mkdir(fts.root, src, mfs.MkdirOpts{
		Mkparents:  true,
		Flush:      false,
		CidBuilder: fts.root.GetDirectory().GetCidBuilder(),
	})
}

func (fts *FileTreeState) PinCidFile(params ...string) error {
	if len(params) != 3 {
		return ErrParamsNum
	}
	cidBytes := []byte(params[0])
	k := ds.KeyWithNamespaces([]string{PinCid, params[1]})
	err := fts.ipfsDb.Put(k, cidBytes)
	if err != nil {
		return err
	}
	peers := make([]string, 0)
	_ = json.Unmarshal(cidBytes, &peers)
	f := false
	for _, peer := range peers {
		if peer == fts.ID {
			f = true
		}
		k = ds.KeyWithNamespaces([]string{PinPeer, peer})
		err := fts.ipfsDb.Put(k, cidBytes)
		if err != nil {
			return err
		}
	}
	if !f {
		return nil
	}
	c, err := cid.Decode(params[1])
	if err != nil {
		return err
	}
	err = fts.pin.Add(fts.ctx, path.IpfsPath(c))
	if err != nil {
		return err
	}
	cctx, cancel := context.WithTimeout(context.Background(), time.Second*20)
	defer cancel()
	ipldNode, err := fts.dag.Get(cctx, c)
	if err != nil {
		return err
	}
	return fts.cp(ipldNode.RawData(), params[2], params[1])
}

func (fts *FileTreeState) Rm(params ...string) error {
	if len(params) != 1 {
		return ErrParamsNum
	}
	dir, name := gopath.Split(params[0])

	pdir, err := getParentDir(fts.root, dir)
	if err != nil {
		if err == os.ErrNotExist {
			return nil
		}
		return fmt.Errorf("parent lookup: %s", err)
	}
	err = pdir.Unlink(name)
	if err != nil {
		if err == os.ErrNotExist {
			return nil
		}
		return err
	}
	return pdir.Flush()
}

func (fts *FileTreeState) Flush() error {
	_, err := mfs.FlushPath(context.Background(), fts.root, "/")
	if err != nil {
		return err
	}
	return fts.store.StoreState(fts.String())
}

func (fts *FileTreeState) Root() (string, error) {
	n, err := fts.root.GetDirectory().GetNode()
	if err != nil {
		return "", err
	}
	return n.Cid().String(), nil
}

func (fts *FileTreeState) MustGetRoot() string {
	for {
		n, err := fts.root.GetDirectory().GetNode()
		if err != nil {
			time.Sleep(time.Millisecond * 20)
			continue
		}
		return n.Cid().String()
	}
}

func (fts *FileTreeState) Marshal(writer io.Writer) error {
	_, err := writer.Write([]byte(fts.String()))
	return err
}

func (fts *FileTreeState) String() string {
	d := SnapShot{
		Index: fts.Index(),
		Root:  fts.MustGetRoot(),
	}
	data, _ := json.Marshal(d)
	return string(data)
}

func (fts *FileTreeState) Lock() SnapShot {
	fts.mtx.Lock()
	ss := SnapShot{
		Index: fts.Index(),
		Root:  fts.MustGetRoot(),
	}
	return ss
}

func (fts *FileTreeState) UnLock() SnapShot {
	ss := SnapShot{
		Index: fts.Index(),
		Root:  fts.MustGetRoot(),
	}
	fts.mtx.Unlock()
	return ss
}

func (fts *FileTreeState) SnapShot() SnapShot {
	ss := SnapShot{
		Index: fts.Index(),
		Root:  fts.MustGetRoot(),
	}
	return ss
}

func (fts *FileTreeState) RollBack(ss SnapShot) error {
	fts.mtx.Lock()
	defer fts.mtx.Unlock()
	if fts.Index() > ss.Index {
		return nil
	}
	return fts.Unmarshal(strings.NewReader(ss.String()))
}

func (fts *FileTreeState) MustRollBack(ss SnapShot) {
	fts.mtx.Lock()
	defer fts.mtx.Unlock()
	for {
		if fts.Index() > ss.Index {
			return
		}
		if err := fts.Unmarshal(strings.NewReader(ss.String())); err != nil {
			time.Sleep(time.Millisecond * 20)
			continue
		}
	}
}

func (fts *FileTreeState) Index() uint64 {
	return atomic.LoadUint64(&fts.index)
}

func (fts *FileTreeState) SetIndex(idx uint64) {
	atomic.StoreUint64(&fts.index, idx)
}

func walkDirectory(ctx context.Context, dir *mfs.Directory, visited map[string]bool) error {
	ls, err := dir.List(ctx)
	if err != nil {
		return err
	}
	for _, node := range ls {
		node, err := dir.Child(node.Name)
		if err != nil {
			return err
		}
		if fnode, err := node.GetNode(); err != nil {
			return err
		} else {
			if _, ok := visited[fnode.Cid().String()]; ok {
				return nil
			} else {
				visited[fnode.Cid().String()] = true
			}
		}
		if node.Type() == mfs.TDir {
			err = walkDirectory(ctx, node.(*mfs.Directory), visited)
			if err != nil {
				return err
			}
		}
		return nil
	}
	return nil
}

func (fts *FileTreeState) EnsureStored() error {
	visited := make(map[string]bool)
	return walkDirectory(fts.ctx, fts.root.GetDirectory(), visited)
}

func (fts *FileTreeState) Unmarshal(reader io.Reader) error {
	bs, err := ioutil.ReadAll(reader)
	if err != nil {
		return err
	}
	state := struct {
		Index uint64 `json:"index"`
		Root  string `json:"root"`
	}{}
	if err = json.Unmarshal(bs, &state); err != nil {
		return err
	}
	c, err := cid.Decode(state.Root)
	if err != nil {
		return err
	}
	raw, err := fts.dag.Get(fts.ctx, c)
	if err != nil {
		return err
	}

	rootNode, ok := raw.(*merkledag.ProtoNode)
	if !ok {
		return errors.New("invalid root node")
	}
	r, err := mfs.NewRoot(fts.ctx, fts.dag, rootNode, func(ctx context.Context, cid cid.Cid) error {
		return nil
	})
	if err != nil {
		return err
	}
	fts.root = r
	fts.SetIndex(state.Index)
	return nil
}

func NewFileTreeState(store datastore.StateDB, id string, dag format.DAGService, pin iface.PinAPI, fs iface.UnixfsAPI, d *badger.Datastore) (*FileTreeState, error) {
	s, err := store.LoadState()
	state := &FileTreeState{
		ID:     id,
		dag:    dag,
		pin:    pin,
		fs:     fs,
		store:  store,
		ctx:    context.Background(),
		ipfsDb: d,
	}
	if err != nil {
		if err != datastore.ErrKeyNotFound {
			return nil, err
		} else {
			r, _ := mfs.NewRoot(context.Background(), dag, unixfs.EmptyDirNode(), func(ctx context.Context, cid cid.Cid) error {
				return nil
			})
			state.root = r
		}
	} else {
		err := state.Unmarshal(strings.NewReader(s))
		if err != nil {
			return nil, err
		}
		_ = state.EnsureStored()
	}
	return state, nil
}

func checkPath(p string) (string, error) {
	if len(p) == 0 {
		return "", fmt.Errorf("paths must not be empty")
	}

	if p[0] != '/' {
		return "", fmt.Errorf("paths must start with a leading slash")
	}

	cleaned := gopath.Clean(p)
	if p[len(p)-1] == '/' && p != "/" {
		cleaned += "/"
	}
	return cleaned, nil
}

func getParentDir(root *mfs.Root, dir string) (*mfs.Directory, error) {
	parent, err := mfs.Lookup(root, dir)
	if err != nil {
		return nil, err
	}

	pdir, ok := parent.(*mfs.Directory)
	if !ok {
		return nil, errors.New("expected *mfs.Directory, didn't get it. This is likely a race condition")
	}
	return pdir, nil
}

type SnapShot struct {
	Index uint64 `json:"index"`
	Root  string `json:"root"`
}

func (ss SnapShot) String() string {
	data, _ := json.Marshal(ss)
	return string(data)
}
