package mobile

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/mosaicnetworks/babble/src/babble"
	"github.com/mosaicnetworks/babble/src/config"
	"github.com/mosaicnetworks/babble/src/node"
	"github.com/mosaicnetworks/babble/src/proxy"
	"github.com/mosaicnetworks/babble/src/proxy/inmem"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// Node ...
type Node struct {
	nodeID uint32
	node   *node.Node
	proxy  proxy.AppProxy
	logger *logrus.Entry
}

// New initializes Node struct
func New(
	//	privKey string,   // Loaded in InitKey
	//	nodeAddr string,
	//	jsonPeers string,
	//	jsonGenesisPeers string,
	commitHandler CommitHandler,
	exceptionHandler ExceptionHandler,
	configDir string,
	//config *MobileConfig
) *Node {

	babbleConfig := config.NewDefaultConfig()

	babbleConfig.Logger().WithFields(logrus.Fields{
		"config": fmt.Sprintf("%v", babbleConfig),
	}).Debug("New Mobile Node")

	viper.SetConfigName("babble")  // name of config file (without extension)
	viper.AddConfigPath(configDir) // search root directory

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		babbleConfig.Logger().Debugf("Using config file: %s", viper.ConfigFileUsed())
	} else if _, ok := err.(viper.ConfigFileNotFoundError); ok {
		babbleConfig.Logger().Debugf("No config file found in: %s", filepath.Join(configDir, "babble.toml"))
	} else {
		babbleConfig.Logger().Errorf("Error loading config file: %v", err)
		return nil
	}

	if err := viper.Unmarshal(babbleConfig); err != nil {
		babbleConfig.Logger().Errorf("Error marshalling config file: %v", err)
		return nil
	}

	//mobileApp implements the ProxyHandler interface, and we use it to
	//instantiate an InmemProxy
	mobileApp := newMobileApp(commitHandler, exceptionHandler, babbleConfig.Logger())
	babbleConfig.Proxy = inmem.NewInmemProxy(mobileApp, babbleConfig.Logger())

	engine := babble.NewBabble(babbleConfig)

	//	engine.Peers = peerSet
	//	engine.GenesisPeers = genesisPeerSet

	if err := engine.Init(); err != nil {
		exceptionHandler.OnException(fmt.Sprintf("Cannot initialize engine: %s", err))
		return nil
	}

	return &Node{
		node:   engine.Node,
		proxy:  babbleConfig.Proxy,
		nodeID: engine.Node.GetID(),
		logger: babbleConfig.Logger(),
	}
}

/*
// OldNew initializes Node struct
func OldNew(privKey string,
	nodeAddr string,
	jsonPeers string,
	jsonGenesisPeers string,
	commitHandler CommitHandler,
	exceptionHandler ExceptionHandler,
	config *MobileConfig) *Node {

	babbleConfig := config.toBabbleConfig()

	babbleConfig.Logger().WithFields(logrus.Fields{
		"nodeAddr": nodeAddr,
		"peers":    jsonPeers,
		"config":   fmt.Sprintf("%v", config),
	}).Debug("New Mobile Node")

	babbleConfig.BindAddr = nodeAddr

	//Check private key
	keyBytes, err := hex.DecodeString(privKey)
	if err != nil {
		exceptionHandler.OnException(fmt.Sprintf("Failed to decode private key bytes: %s", err))
		return nil
	}

	key, err := keys.ParsePrivateKey(keyBytes)
	if err != nil {
		exceptionHandler.OnException(fmt.Sprintf("Failed to parse private key: %s", err))
		return nil
	}

	babbleConfig.Key = key

	// Decode the peers
	var ps []*peers.Peer
	dec := json.NewDecoder(strings.NewReader(jsonPeers))
	if err := dec.Decode(&ps); err != nil {
		exceptionHandler.OnException(fmt.Sprintf("Failed to parse PeerSet: %s", err))
		return nil
	}

	peerSet := peers.NewPeerSet(ps)

	// Decode the genesis peers
	var gps []*peers.Peer
	decoder := json.NewDecoder(strings.NewReader(jsonGenesisPeers))
	if err := decoder.Decode(&gps); err != nil {
		exceptionHandler.OnException(fmt.Sprintf("Failed to parse PeerSet: %s", err))
		return nil
	}

	genesisPeerSet := peers.NewPeerSet(gps)

	babbleConfig.LoadPeers = false

	// mobileApp implements the ProxyHandler interface, and we use it to
	// instantiate an InmemProxy
	mobileApp := newMobileApp(commitHandler, exceptionHandler, babbleConfig.Logger())
	babbleConfig.Proxy = inmem.NewInmemProxy(mobileApp, babbleConfig.Logger())

	engine := babble.NewBabble(babbleConfig)

	engine.Peers = peerSet
	engine.GenesisPeers = genesisPeerSet

	if err := engine.Init(); err != nil {
		exceptionHandler.OnException(fmt.Sprintf("Cannot initialize engine: %s", err))
		return nil
	}

	return &Node{
		node:   engine.Node,
		proxy:  babbleConfig.Proxy,
		nodeID: engine.Node.GetID(),
		logger: babbleConfig.Logger(),
	}
}

*/

// Run ...
func (n *Node) Run(async bool) {
	if async {
		n.node.RunAsync(true)
	} else {
		n.node.Run(true)
	}
}

// Leave ...
func (n *Node) Leave() {
	n.node.Leave()
}

// Shutdown ...
func (n *Node) Shutdown() {
	n.node.Shutdown()
}

// SubmitTx ...
func (n *Node) SubmitTx(tx []byte) {
	//have to make a copy or the tx will be garbage collected and weird stuff
	//happens in transaction pool
	t := make([]byte, len(tx), len(tx))
	copy(t, tx)
	n.proxy.SubmitCh() <- t
}

// GetPeers ...
func (n *Node) GetPeers() string {
	peers := n.node.GetPeers()

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	if err := enc.Encode(peers); err != nil {
		return ""
	}

	return buf.String()
}

// GetGenesisPeers ...
func (n *Node) GetGenesisPeers() string {
	peers, err := n.node.GetValidatorSet(0)

	if err != nil {
		return ""
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	if err := enc.Encode(peers); err != nil {
		return ""
	}

	return buf.String()
}

// GetStats ...
func (n *Node) GetStats() string {
	stats := n.node.GetStats()

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	if err := enc.Encode(stats); err != nil {
		return ""
	}

	return buf.String()
}
