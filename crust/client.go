package main

import (
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/ipfs/go-cid"
	"sync"

	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v3"
	"github.com/centrifuge/go-substrate-rpc-client/v3/signature"
	"github.com/centrifuge/go-substrate-rpc-client/v3/types"
)

var fileInfoKeyPrefix, _ = hex.DecodeString("5ebf094108ead4fefa73f7a3b13cb4a7b3b78f30e9b952d60249b22fcdaaa76dac896fb8ba4ecabfb8")
var CrustNetworkID uint8 = 42

type Client struct {
	keyPair     signature.KeyringPair
	api         *gsrpc.SubstrateAPI
	recordNonce uint64
	genesisHash types.Hash
	mtx         sync.Mutex
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

func (c *Client) PlaceStorageOrder(fileCid cid.Cid, fileSize uint64, tip uint64) (*types.Hash, error) {
	meta, err := c.api.RPC.State.GetMetadataLatest()
	if err != nil {
		return nil, err
	}
	call, err := types.NewCall(meta, "Market.place_storage_order", fileCid.String(), fileSize, types.NewUCompactFromUInt(tip))
	if err != nil {
		return nil, err
	}
	ext := types.NewExtrinsic(call)
	rv, err := c.api.RPC.State.GetRuntimeVersionLatest()
	if err != nil {
		return nil, err
	}

	key, err := types.CreateStorageKey(meta, "System", "Account", c.keyPair.PublicKey, nil)
	if err != nil {
		return nil, err
	}

	var accountInfo types.AccountInfo
	ok, err := c.api.RPC.State.GetStorageLatest(key, &accountInfo)

	if err != nil || !ok {
		return nil, err
	}
	c.mtx.Lock()
	nonce := max(uint64(accountInfo.Nonce), c.recordNonce)
	c.recordNonce = nonce + 1
	c.mtx.Unlock()

	o := types.SignatureOptions{
		BlockHash:          c.genesisHash,
		Era:                types.ExtrinsicEra{IsMortalEra: false},
		GenesisHash:        c.genesisHash,
		Nonce:              types.NewUCompactFromUInt(nonce),
		SpecVersion:        rv.SpecVersion,
		Tip:                types.NewUCompactFromUInt(0),
		TransactionVersion: rv.TransactionVersion,
	}
	err = ext.Sign(c.keyPair, o)
	if err != nil {
		return nil, err
	}
	hash, err := c.api.RPC.Author.SubmitExtrinsic(ext)
	if err != nil {
		return nil, err
	}
	return &hash, nil
}

func (c *Client) GetFileInfo(fileCid cid.Cid) (*FileInfo, error) {
	var fileInfo = FileInfo{}
	key := append(fileInfoKeyPrefix, []byte(fileCid.String())...)
	ok, err := c.api.RPC.State.GetStorageLatest(key, &fileInfo)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New(fmt.Sprintf("order cid: %s not found", fileCid.String()))
	}
	return &fileInfo, nil
}

func max(a, b uint64) uint64 {
	if a > b {
		return a
	}
	return b
}

func NewClient(wsUrl, secret string) (*Client, error) {
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
	var client = &Client{
		keyPair:     keyPair,
		api:         api,
		recordNonce: 0,
		genesisHash: genesisHash,
	}
	return client, nil
}

func main() {
	// Display the events that occur during a transfer by sending a value to bob

	// Instantiate the API
	api, err := gsrpc.NewSubstrateAPI("wss://rocky-api.crust.network/")
	if err != nil {
		panic(err)
	}

	keypair, err := signature.KeyringPairFromSecret("tomorrow gun unfair damp crisp pet basket zone matrix kidney together april", 42)
	if err != nil {
		panic(err)
	}
	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		panic(err)
	}
	for _, sector := range meta.AsMetadataV12.Modules {
		fmt.Println(sector.Name)

		//for _, f := range sector.Calls {
		//	fmt.Println("	method: ", f.Name, f.Args)
		//}

		for _, item := range sector.Storage.Items {

			fmt.Println(item.Type.AsMap.Key, "->", item.Type.AsMap.Value)
			fmt.Printf("	key:    %#v, %#v, %#v \n", item.Name, item.Type.AsMap.Value, item.Documentation)

		}
	}

	// Create a call, transferring 12345 units to Bob
	//bob, err := types.NewAddressFromHexAccountID("0x8eaf04151687736326c9fea17e25fc5287613693c912909cb226aa4794f26a48")
	//if err != nil {
	//	panic(err)
	//}

	//ccid, _ := cid.Decode("QmVCN9cXCjnuG9DUmY1wRt9E8unwtmMZzLLycb6WBEPjdZ")
	key1, err := types.CreateStorageKey(meta, "Market", "Files", []byte("QmVCN9cXCjnuG9DUmY1wRt9E8unwtmMZzLLycb6WBEPjdZ"), nil)
	fmt.Println("hexx", hex.EncodeToString([]byte("QmVCN9cXCjnuG9DUmY1wRt9E8unwtmMZzLLycb6WBEPjdZ")))
	if err != nil {
		panic(err)
	}
	fmt.Println("hex", key1.Hex())

	key1, _ = hex.DecodeString("5ebf094108ead4fefa73f7a3b13cb4a7b3b78f30e9b952d60249b22fcdaaa76dac896fb8ba4ecabfb8")
	key1 = append(key1, []byte("QmVCN9cXCjnuG9DUmY1wRt9E8unwtmMZzLLycb6WBEPjdZ")...)
	kk, err := api.RPC.State.GetStorageRawLatest(key1)
	if err != nil {
		panic(err)
	}
	fmt.Println("????", kk.Hex())
	var fileInfo FileInfo
	ok1, err := api.RPC.State.GetStorageLatest(key1, &fileInfo)
	if err != nil || !ok1 {
		panic(err)
	}
	fmt.Println(fileInfo)

	key3, err := types.CreateStorageKey(meta, "System", "Account", keypair.PublicKey, nil)
	if err != nil {
		panic(err)
	}
	kk2, err := api.RPC.State.GetStorageRawLatest(key3)
	if err != nil {
		panic(err)
	}
	fmt.Println("????", kk2.Hex())

	amount := types.NewUCompactFromUInt(0)

	c, err := types.NewCall(meta, "Market.place_storage_order", "QmVCN9cXCjnuG9DUmY1wRt9E8unwtmMZzLLycb6WBEPjdZ", uint64(17), amount)
	if err != nil {
		panic(err)
	}

	// Create the extrinsic
	ext := types.NewExtrinsic(c)

	genesisHash, err := api.RPC.Chain.GetBlockHash(0)
	if err != nil {
		panic(err)
	}

	rv, err := api.RPC.State.GetRuntimeVersionLatest()
	if err != nil {
		panic(err)
	}

	// Get the nonce for Alice
	key, err := types.CreateStorageKey(meta, "System", "Account", keypair.PublicKey, nil)
	if err != nil {
		panic(err)
	}

	var accountInfo types.AccountInfo
	ok, err := api.RPC.State.GetStorageLatest(key, &accountInfo)
	if err != nil || !ok {
		panic(err)
	}

	nonce := uint32(accountInfo.Nonce)

	o := types.SignatureOptions{
		BlockHash:          genesisHash,
		Era:                types.ExtrinsicEra{IsMortalEra: false},
		GenesisHash:        genesisHash,
		Nonce:              types.NewUCompactFromUInt(uint64(nonce)),
		SpecVersion:        rv.SpecVersion,
		Tip:                types.NewUCompactFromUInt(0),
		TransactionVersion: rv.TransactionVersion,
	}

	fmt.Printf("Sending %v from %#x with nonce %v\n", amount, keypair.PublicKey, nonce)

	// Sign the transaction using Alice's default account
	err = ext.Sign(keypair, o)
	if err != nil {
		panic(err)
	}
	fmt.Println(types.EncodeToHexString(ext))

	// Do the transfer and track the actual status
	sub, err := api.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		panic(err)
	}
	defer sub.Unsubscribe()

	for {
		status := <-sub.Chan()
		fmt.Printf("Transaction status: %#v\n", status)
		bs, _ := status.MarshalJSON()
		fmt.Println(string(bs))
		if status.IsInBlock {
			fmt.Printf("Completed at block hash: %#x\n", status.AsInBlock)
		}
	}
}

//0x02000000000000000100000001837423f91a07000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000

//0x5ebf094108ead4fefa73f7a3b13cb4a7b3b78f30e9b952d60249b22fcdaaa76d682705db4aa834e900
//0x5ebf094108ead4fefa73f7a3b13cb4a7b3b78f30e9b952d60249b22fcdaaa76d7051a818f93b1367 516d56434e396358436a6e75473944556d5931775274394538756e77746d4d5a7a4c4c79636236574245506a645a
//0x5ebf094108ead4fefa73f7a3b13cb4a7b3b78f30e9b952d60249b22fcdaaa76dac896fb8ba4ecabfb8 516d56434e396358436a6e75473944556d5931775274394538756e77746d4d5a7a4c4c79636236574245506a645a
