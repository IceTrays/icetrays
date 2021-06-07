package crust

import (
	"errors"
	"github.com/ipfs/go-cid"
	"math/big"
	"sync"
	"time"

	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v3"
	"github.com/centrifuge/go-substrate-rpc-client/v3/signature"
	"github.com/centrifuge/go-substrate-rpc-client/v3/types"
)

var (
	ErrCrustUpgraded          = errors.New("may be crust upgraded")
	ErrBalanceNotEnough       = errors.New("balance not enough")
	ErrSubmitExtrinsicTimeout = errors.New("submit extrinsic timeout")
	ErrCidNotFound            = errors.New("cid not found in cust")
)

var CrustNetworkID uint8 = 42

type Client struct {
	keyPair       signature.KeyringPair
	api           *gsrpc.SubstrateAPI
	genesisHash   types.Hash
	mtx           sync.Mutex
	meta          *types.Metadata
	submitTimeout time.Duration
}

/*
	file_size: 'u64',
	expired_on: 'BlockNumber',
	calculated_at: 'BlockNumber',
	amount: 'Balance',
	prepaid: 'Balance',
	reported_replica_count: 'u32',
	replicas: 'Vec<Replica<AccountId>>',
*/
type FileInfo struct {
	FileSize             types.U64
	ExpiredOn            types.BlockNumber
	CalculatedAt         types.BlockNumber
	Amount               types.U128
	Prepaid              types.U128
	ReportedReplicaCount types.U32
}

func (c *Client) PlaceStorageOrder(fileCid cid.Cid, fileSize uint64, tip uint64) error {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	meta, err := c.api.RPC.State.GetMetadataLatest()
	if err != nil {
		return err
	}
	call, err := types.NewCall(meta, "Market.place_storage_order", fileCid.String(), fileSize, types.NewUCompactFromUInt(tip))
	if err != nil {
		return err
	}
	ext := types.NewExtrinsic(call)
	rv, err := c.api.RPC.State.GetRuntimeVersionLatest()
	if err != nil {
		return err
	}

	accountInfo, err := c.getAccountInfo()
	if err != nil {
		return err
	}

	orderPrice, err := c.getOrderPrice(fileSize)
	if err != nil {
		return err
	}
	if orderPrice.Cmp(accountInfo.Data.Free.Int) > 0 {
		return ErrBalanceNotEnough
	}

	o := types.SignatureOptions{
		BlockHash:          c.genesisHash,
		Era:                types.ExtrinsicEra{IsMortalEra: false},
		GenesisHash:        c.genesisHash,
		Nonce:              types.NewUCompactFromUInt(uint64(accountInfo.Nonce)),
		SpecVersion:        rv.SpecVersion,
		Tip:                types.NewUCompactFromUInt(0),
		TransactionVersion: rv.TransactionVersion,
	}
	err = ext.Sign(c.keyPair, o)
	if err != nil {
		return err
	}

	sub, err := c.api.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		panic(err)
	}
	defer sub.Unsubscribe()
	timeOut := time.NewTimer(c.submitTimeout)
	for {
		select {
		case status := <-sub.Chan():
			// todo log
			if status.IsInBlock {
				return nil
			}
		case <-timeOut.C:
			return ErrSubmitExtrinsicTimeout
		}
	}
}

func (c *Client) GetFileInfo(fileCid cid.Cid) (*FileInfo, error) {
	cidbs := []byte(fileCid.String())
	bs := make([]byte, len(cidbs)+1)
	// TODO no reason
	bs[0] = byte(len(cidbs) * 4)
	copy(bs[1:], cidbs)
	bs = append(bs)

	key, err := types.CreateStorageKey(c.meta, "Market", "Files", bs, nil)
	if err != nil {
		return nil, err
	}

	var fileInfo = FileInfo{}

	ok, err := c.api.RPC.State.GetStorageLatest(key, &fileInfo)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrCidNotFound
	}
	return &fileInfo, nil
}

func (c *Client) getAccountInfo() (*types.AccountInfo, error) {
	key, err := types.CreateStorageKey(c.meta, "System", "Account", c.keyPair.PublicKey, nil)
	if err != nil {
		return nil, err
	}

	var accountInfo types.AccountInfo
	ok, err := c.api.RPC.State.GetStorageLatest(key, &accountInfo)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrCrustUpgraded
	}
	return &accountInfo, nil
}

func (c *Client) getOrderPrice(fileSize uint64) (*types.U128, error) {
	fileSizeWithMB := fileSize/1024/1024 + 1
	price, err := c.getFilePricePerMB()
	if err != nil {
		return nil, err
	}
	shouldPay := price.Mul(price.Int, big.NewInt(int64(fileSizeWithMB)))
	fileBase, err := c.getFileBase()
	if err != nil {
		return nil, err
	}
	if shouldPay.Cmp(fileBase.Int) >= 0 {
		return &types.U128{Int: shouldPay}, nil
	} else {
		return fileBase, nil
	}
}

func (c *Client) getFileBase() (*types.U128, error) {
	key, err := types.CreateStorageKey(c.meta, "Market", "FileBaseFee", nil, nil)
	var fileBase types.U128
	ok, err := c.api.RPC.State.GetStorageLatest(key, &fileBase)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrCrustUpgraded
	}
	return &fileBase, nil
}

func (c *Client) getFilePricePerMB() (*types.U128, error) {
	key, err := types.CreateStorageKey(c.meta, "Market", "FilePrice", nil, nil)
	var filePrice types.U128
	ok, err := c.api.RPC.State.GetStorageLatest(key, &filePrice)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrCrustUpgraded
	}
	return &filePrice, nil
}

func max(a, b uint64) uint64 {
	if a > b {
		return a
	}
	return b
}

func NewClient(wsUrl, secret string, timeout time.Duration) (*Client, error) {
	api, err := gsrpc.NewSubstrateAPI(wsUrl)
	if err != nil {
		return nil, err
	}
	keyPair, err := signature.KeyringPairFromSecret(secret, CrustNetworkID)
	if err != nil {
		return nil, err
	}
	genesisHash, err := api.RPC.Chain.GetBlockHash(0)
	if err != nil {
		return nil, err
	}
	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return nil, err
	}
	var client = &Client{
		keyPair:       keyPair,
		api:           api,
		genesisHash:   genesisHash,
		meta:          meta,
		submitTimeout: timeout,
	}
	return client, nil
}
