package webui

import (
	"fmt"
	"time"
	"strings"
	"net/http"
	"encoding/json"
	"github.com/piotrnar/gocoin/lib/btc"
	"github.com/piotrnar/gocoin/lib/qdb"
	"github.com/piotrnar/gocoin/lib/others/sys"
	"github.com/piotrnar/gocoin/client/usif"
	"github.com/piotrnar/gocoin/client/common"
	"github.com/piotrnar/gocoin/client/wallet"
	"github.com/piotrnar/gocoin/client/network"
	"github.com/piotrnar/gocoin/lib/others/peersdb"
)


func p_home(w http.ResponseWriter, r *http.Request) {
	if !ipchecker(r) {
		return
	}

	// The handler also gets called for /favicon.ico
	if r.URL.Path!="/" {
		http.NotFound(w, r)
	}

	s := load_template("home.html")

	wallet.BalanceMutex.Lock()
	if len(wallet.MyBalance)>0 {
		wal := load_template("home_wal.html")
		wal = strings.Replace(wal, "{TOTAL_BTC}", fmt.Sprintf("%.8f", float64(wallet.LastBalance)/1e8), 1)
		wal = strings.Replace(wal, "{UNSPENT_OUTS}", fmt.Sprint(len(wallet.MyBalance)), 1)
		s = strings.Replace(s, "<!--WALLET-->", wal, 1)
	} else {
		if wallet.MyWallet==nil {
			s = strings.Replace(s, "<!--WALLET-->", "You have no wallet", 1)
		} else {
			s = strings.Replace(s, "<!--WALLET-->", "Your balance is <b>zero</b>", 1)
		}
	}
	wallet.BalanceMutex.Unlock()

	s = strings.Replace(s, "<--NETWORK_HASHRATE-->", usif.GetNetworkHashRate(), 1)

	network.Mutex_net.Lock()
	s = strings.Replace(s, "{OPEN_CONNS_TOTAL}", fmt.Sprint(len(network.OpenCons)), 1)
	s = strings.Replace(s, "{OPEN_CONNS_OUT}", fmt.Sprint(network.OutConsActive), 1)
	s = strings.Replace(s, "{OPEN_CONNS_IN}", fmt.Sprint(network.InConsActive), 1)
	network.Mutex_net.Unlock()

	common.LockBw()
	common.TickRecv()
	common.TickSent()
	s = strings.Replace(s, "{DL_SPEED_NOW}", fmt.Sprint(common.DlBytesPrevSec>>10), 1)
	s = strings.Replace(s, "{DL_SPEED_MAX}", fmt.Sprint(common.DownloadLimit>>10), 1)
	s = strings.Replace(s, "{DL_TOTAL}", common.BytesToString(common.DlBytesTotal), 1)
	s = strings.Replace(s, "{UL_SPEED_NOW}", fmt.Sprint(common.UlBytesPrevSec>>10), 1)
	s = strings.Replace(s, "{UL_SPEED_MAX}", fmt.Sprint(common.UploadLimit>>10), 1)
	s = strings.Replace(s, "{UL_TOTAL}", common.BytesToString(common.UlBytesTotal), 1)
	common.UnlockBw()


	network.ExternalIpMutex.Lock()
	for ip, rec := range network.ExternalIp4 {
		ips := fmt.Sprintf("<b title=\"%d times. Last seen %d min ago\">%d.%d.%d.%d</b> ",
				rec[0], (uint(time.Now().Unix())-rec[1])/60,
				byte(ip>>24), byte(ip>>16), byte(ip>>8), byte(ip))
		s = templ_add(s, "<!--ONE_EXTERNAL_IP-->", ips)
	}
	network.ExternalIpMutex.Unlock()

	s = strings.Replace(s, "<!--NEW_BLOCK_BEEP-->", fmt.Sprint(common.CFG.Beeps.NewBlock), 1)

	common.LockCfg()
	dat, _ := json.Marshal(&common.CFG)
	common.UnlockCfg()
	s = strings.Replace(s, "{CONFIG_FILE}", strings.Replace(string(dat), ",\"", ", \"", -1), 1)

	write_html_head(w, r)
	w.Write([]byte(s))
	write_html_tail(w)
}


func json_status(w http.ResponseWriter, r *http.Request) {
	if !ipchecker(r) {
		return
	}

	w.Header()["Content-Type"] = []string{"application/json"}

	w.Write([]byte("{"))
	common.Last.Mutex.Lock()
	w.Write([]byte(fmt.Sprint("\"height\":", common.Last.Block.Height, ",")))
	w.Write([]byte(fmt.Sprint("\"hash\":\"", common.Last.Block.BlockHash.String(), "\",")))
	w.Write([]byte(fmt.Sprint("\"timestamp\":", common.Last.Block.Timestamp(), ",")))
	w.Write([]byte(fmt.Sprint("\"received\":", common.Last.Time.Unix(), ",")))
	w.Write([]byte(fmt.Sprint("\"time_now\":", time.Now().Unix(), ",")))
	w.Write([]byte(fmt.Sprint("\"diff\":", btc.GetDifficulty(common.Last.Block.Bits()))))
	common.Last.Mutex.Unlock()
	w.Write([]byte("}"))
}


func json_system(w http.ResponseWriter, r *http.Request) {
	if !ipchecker(r) {
		return
	}

	w.Header()["Content-Type"] = []string{"application/json"}
	w.Write([]byte("{"))

	al, sy := sys.MemUsed()
	w.Write([]byte(fmt.Sprint("\"blocks_cached\":", len(network.CachedBlocks), ",")))
	w.Write([]byte(fmt.Sprint("\"known_peers\":", peersdb.PeerDB.Count(), ",")))
	w.Write([]byte(fmt.Sprint("\"node_uptime\":", time.Now().Sub(common.StartTime).Seconds(), ",")))
	w.Write([]byte(fmt.Sprint("\"net_block_qsize\":\"", len(network.NetBlocks), "\",")))
	w.Write([]byte(fmt.Sprint("\"net_tx_qsize\":\"", len(network.NetTxs), "\",")))
	w.Write([]byte(fmt.Sprint("\"heap_size\":", al, ",")))
	w.Write([]byte(fmt.Sprint("\"heap_sysmem\":", sy, ",")))
	w.Write([]byte(fmt.Sprint("\"qdb_extramem\":", qdb.ExtraMemoryConsumed, ",")))
	w.Write([]byte(fmt.Sprint("\"ecdsa_verify_cnt\":", btc.EcdsaVerifyCnt, "")))

	w.Write([]byte("}\n"))
}
