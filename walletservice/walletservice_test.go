// Copyright (c) 2018 IoTeX
// This is an alpha (internal) release and is not suitable for production. This source code is provided 'as is' and no
// warranties are given as to title or non-infringement, merchantability or fitness for purpose and, to the extent
// permitted by law, all liability for your use of the code is disclaimed. This source code is governed by Apache
// License 2.0 that can be found in the LICENSE file.

package walletservice

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/iotexproject/iotex-core/iotxaddress"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/freitx-project/freitx-network-blockchain/config"
	"github.com/freitx-project/freitx-network-blockchain/crypto"
	"github.com/freitx-project/freitx-network-blockchain/pkg/enc"
	"github.com/freitx-project/freitx-network-blockchain/pkg/keypair"
	"github.com/freitx-project/freitx-network-blockchain/server/itx"
	"github.com/freitx-project/freitx-network-blockchain/testutil"
	"github.com/freitx-project/freitx-network-wallet/pb"
)

const (
	creatorPubKey = "1da55277895a7c9ce5ef7b591a7ff0f2ad1985ef58a6e65cb1823b36e7b0960ba1c40402517be6025f257daf4402b38240a657147760d8272465ae48076c88367358b58e4d756701"
	creatorPriKey = "720de16059fa942f415ad57599ab8869235682945118503bc00702a45747cee1c0077a01"
	pubKey1       = "bdfed45c709c8f4694644e4cb9a32ba7266e0827ff9d526fcd817d721ed16abe22ce3d00a6b7bba482088dc7f963aeb00c9881c31d6fd11ff05a0908ec3a479840f45fc992971e00"
	priKey1       = "635ff4444d1fe7c0121d71a51e9854b2e0d0670e628ccdeaa7fac049b6138bd320703f00"
	rawAddr1      = "1x1qyqsqqqqmv8myg82q06d528r5w82r7xlajvkakl5k9gzdh"
	pubKey2       = "b9c6b0dd0abc59731216544a1711643909f0038b49d91ff0f052726efe823b742f89fc069ce22728b8f4f4d41d1c90577543d2caed8bbb54db7cc1f17437b8c4b499b617b5688804"
	priKey2       = "79687a02a9269e8e77858a247ea4937e4d01de321d7aec24d9a03eca5befa90269208501"
	rawAddr2      = "1x1qyqsqqqq8569qmwseyv0nfk7rydw9xr467jtpvtx02nyjn"

	testChainPath = "./chain.db"
	testTriePath  = "./trie.db"
)

func TestWalletServer_NewWallet(t *testing.T) {
	require := require.New(t)

	testutil.CleanupPath(t, testChainPath)
	testutil.CleanupPath(t, testTriePath)

	cfg, err := newConfig()
	require.NoError(err)
	ctx := context.Background()
	chainID := cfg.Chain.ID

	// create server
	svr, err := itx.NewServer(*cfg)
	require.NoError(err)
	require.Nil(svr.Start(ctx))
	defer func() {
		require.NoError(svr.Stop(ctx))
		testutil.CleanupPath(t, testChainPath)
		testutil.CleanupPath(t, testTriePath)
	}()
	explorerAddr := fmt.Sprintf("127.0.0.1:%d", svr.ChainService(chainID).Explorer().Port())
	s := NewWalletServer(":42124", explorerAddr, 5, 10, 5, 1, creatorPubKey, creatorPriKey)
	s.Start()
	defer s.Stop()

	conn, err := grpc.Dial(":42124", grpc.WithInsecure())
	require.NoError(err)
	defer conn.Close()

	client := pb.NewWalletServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	r, err := client.NewWallet(ctx, &pb.NewWalletRequest{ChainID: 1})
	require.NoError(err)

	publicKey, err := keypair.DecodePublicKey(r.Address.PublicKey)
	require.NoError(err)
	require.Equal(72, len(publicKey))
	privateKey, err := keypair.DecodePrivateKey(r.Address.PrivateKey)
	require.NoError(err)
	require.Equal(36, len(privateKey))

	// Wait until the injected transfer for the new address gets into the action pool
	require.NoError(testutil.WaitUntil(100*time.Millisecond, 2*time.Second, func() (bool, error) {
		actions := svr.ChainService(chainID).ActionPool().PickActs()
		return len(actions) == 1, nil
	}))
}

func TestWalletServer_Unlock(t *testing.T) {
	require := require.New(t)

	s := NewWalletServer(":42124", "127.0.0.1:14004", 5, 10, 5, 1, creatorPubKey, creatorPriKey)
	s.Start()
	defer s.Stop()

	conn, err := grpc.Dial(":42124", grpc.WithInsecure())
	require.NoError(err)
	defer conn.Close()

	client := pb.NewWalletServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	r, err := client.Unlock(ctx, &pb.UnlockRequest{PrivateKey: priKey1, ChainID: 1})
	require.NoError(err)

	require.Equal(pubKey1, r.Address.PublicKey)
	require.Equal(rawAddr1, r.Address.RawAddress)

	r, err = client.Unlock(ctx, &pb.UnlockRequest{PrivateKey: priKey2, ChainID: 2})
	require.NoError(err)

	require.Equal(pubKey2, r.Address.PublicKey)
	require.NotEqual(rawAddr2, r.Address.RawAddress)
}

func TestWalletServer_SignTransfer(t *testing.T) {
	require := require.New(t)

	s := NewWalletServer(":42124", "127.0.0.1:14004", 5, 10, 5, 1, creatorPubKey, creatorPriKey)
	s.Start()
	defer s.Stop()

	conn, err := grpc.Dial(":42124", grpc.WithInsecure())
	require.NoError(err)
	defer conn.Close()

	client := pb.NewWalletServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	addressPb := &pb.Address{
		PublicKey:  pubKey1,
		PrivateKey: priKey1,
		RawAddress: rawAddr1,
	}

	rawTransferPb := &pb.Transfer{
		Nonce:     1,
		Amount:    "1",
		Sender:    rawAddr1,
		Recipient: rawAddr2,
		GasLimit:  1000000,
		GasPrice:  "10",
	}

	request := &pb.SignTransferRequest{
		Address:  addressPb,
		Transfer: rawTransferPb,
	}

	response, err := client.SignTransfer(ctx, request)
	require.NoError(err)

	require.NotNil(response.Transfer.Signature)
}

func TestWalletServer_SignVote(t *testing.T) {
	require := require.New(t)

	s := NewWalletServer(":42124", "127.0.0.1:14004", 5, 10, 5, 1, creatorPubKey, creatorPriKey)
	s.Start()
	defer s.Stop()

	conn, err := grpc.Dial(":42124", grpc.WithInsecure())
	require.NoError(err)
	defer conn.Close()

	client := pb.NewWalletServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	addressPb := &pb.Address{
		PublicKey:  pubKey2,
		PrivateKey: priKey2,
		RawAddress: rawAddr2,
	}

	rawVotePb := &pb.Vote{
		Nonce:        2,
		VoterAddress: rawAddr2,
		VoteeAddress: rawAddr2,
		GasLimit:     1000000,
		GasPrice:     "10",
	}

	request := &pb.SignVoteRequest{
		Address: addressPb,
		Vote:    rawVotePb,
	}

	response, err := client.SignVote(ctx, request)
	require.NoError(err)

	require.NotNil(response.Vote.Signature)
}

func TestWalletServer_SignExecution(t *testing.T) {
	require := require.New(t)

	s := NewWalletServer(":42124", "127.0.0.1:14004", 5, 10, 5, 1, creatorPubKey, creatorPriKey)
	s.Start()
	defer s.Stop()

	conn, err := grpc.Dial(":42124", grpc.WithInsecure())
	require.NoError(err)
	defer conn.Close()

	client := pb.NewWalletServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	addressPb := &pb.Address{
		PublicKey:  pubKey1,
		PrivateKey: priKey1,
		RawAddress: rawAddr1,
	}

	rawExecutionPb := &pb.Execution{
		Nonce:    3,
		Amount:   "3",
		Executor: rawAddr1,
		Contract: "",
		GasLimit: 1000000,
		GasPrice: "10",
	}

	request := &pb.SignExecutionRequest{
		Address:   addressPb,
		Execution: rawExecutionPb,
	}

	response, err := client.SignExecution(ctx, request)
	require.NoError(err)

	require.NotNil(response.Execution.Signature)
}

func TestWalletServer_SignCreateDeposit(t *testing.T) {
	require := require.New(t)

	s := NewWalletServer(":42124", "127.0.0.1:14004", 5, 10, 5, 1, creatorPubKey, creatorPriKey)
	s.Start()
	defer s.Stop()

	conn, err := grpc.Dial(":42124", grpc.WithInsecure())
	require.NoError(err)
	defer conn.Close()

	client := pb.NewWalletServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	addressPb := &pb.Address{
		PublicKey:  pubKey2,
		PrivateKey: priKey2,
		RawAddress: rawAddr2,
	}

	var chainIDBytes [4]byte
	enc.MachineEndian.PutUint32(chainIDBytes[:], uint32(2))
	pubkey, err := keypair.DecodePublicKey(pubKey2)
	require.NoError(err)
	addr, err := iotxaddress.GetAddressByPubkey(iotxaddress.IsTestnet, chainIDBytes[:], pubkey)

	rawCreateDepositPb := &pb.CreateDeposit{
		Nonce:     4,
		Amount:    "4",
		Sender:    rawAddr2,
		Recipient: addr.RawAddress,
		GasLimit:  1000000,
		GasPrice:  "10",
	}

	request := &pb.SignCreateDepositRequest{
		Address:       addressPb,
		CreateDeposit: rawCreateDepositPb,
	}

	response, err := client.SignCreateDeposit(ctx, request)
	require.NoError(err)

	require.NotNil(response.CreateDeposit.Signature)
}

func TestWalletServer_SignSettleDeposit(t *testing.T) {
	require := require.New(t)

	s := NewWalletServer(":42124", "127.0.0.1:14004", 5, 10, 5, 1, creatorPubKey, creatorPriKey)
	s.Start()
	defer s.Stop()

	conn, err := grpc.Dial(":42124", grpc.WithInsecure())
	require.NoError(err)
	defer conn.Close()

	client := pb.NewWalletServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	addressPb := &pb.Address{
		PublicKey:  pubKey1,
		PrivateKey: priKey1,
		RawAddress: rawAddr1,
	}

	var chainIDBytes [4]byte
	enc.MachineEndian.PutUint32(chainIDBytes[:], uint32(2))
	pubkey, err := keypair.DecodePublicKey(pubKey1)
	require.NoError(err)
	addr, err := iotxaddress.GetAddressByPubkey(iotxaddress.IsTestnet, chainIDBytes[:], pubkey)

	rawSettleDepositPb := &pb.SettleDeposit{
		Nonce:     5,
		Amount:    "5",
		Index:     0,
		Sender:    rawAddr1,
		Recipient: addr.RawAddress,
		GasLimit:  1000000,
		GasPrice:  "10",
	}

	request := &pb.SignSettleDepositRequest{
		Address:       addressPb,
		SettleDeposit: rawSettleDepositPb,
	}

	response, err := client.SignSettleDeposit(ctx, request)
	require.NoError(err)

	require.NotNil(response.SettleDeposit.Signature)
}

func newConfig() (*config.Config, error) {
	cfg := config.Default
	cfg.NodeType = config.DelegateType
	cfg.Consensus.Scheme = config.NOOPScheme
	cfg.Chain.ChainDBPath = testChainPath
	cfg.Chain.TrieDBPath = testTriePath
	cfg.Chain.GenesisActionsPath = "./testnet_actions.yaml"

	pk, sk, err := crypto.EC283.NewKeyPair()
	if err != nil {
		return nil, err
	}
	cfg.Chain.ProducerPubKey = keypair.EncodePublicKey(pk)
	cfg.Chain.ProducerPrivKey = keypair.EncodePrivateKey(sk)
	cfg.Network.Port = 0
	cfg.Network.PeerMaintainerInterval = 100 * time.Millisecond
	cfg.Explorer.Enabled = true
	cfg.Explorer.Port = 0
	return &cfg, nil
}
