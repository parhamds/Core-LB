package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"

	log "github.com/sirupsen/logrus"
)

type RuleReq struct {
	GwIP string   `json:"gwip"`
	Teid []string `json:"teid"`
	Ip   []string `json:"ip"`
}

type RegisterReq struct {
	GwIP    string `json:"gwip"`
	CoreMac string `json:"coremac"`
}

var addedRule map[string]struct{}
var registeredUPFs map[string]string // [gwip] coremac

func main() {
	log.SetLevel(log.TraceLevel)
	log.Traceln("application started")
	addedRule = make(map[string]struct{})
	registeredUPFs = make(map[string]string)
	http.HandleFunc("/addrule", addRuleHandler)
	http.HandleFunc("/register", registerHandler)
	server := http.Server{Addr: ":8080"}

	server.ListenAndServe()

}

func addRuleHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "PUT":
		fallthrough
	case "POST":
		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Errorln("http req read body failed.")
			sendHTTPResp(http.StatusBadRequest, w)
		}

		log.Traceln(string(body))

		//var nwSlice NetworkSlice
		var rulereq RuleReq
		//fmt.Println("parham log : http body = ", body)
		err = json.Unmarshal(body, &rulereq)
		if err != nil {
			log.Errorln("Json unmarshal failed for http request")
			sendHTTPResp(http.StatusBadRequest, w)
		}
		for _, i := range rulereq.Ip {
			added := false
			_, added = addedRule[i]
			if !added {
				err = execAddRule(rulereq.GwIP, i)
				err = execAddArp(rulereq.GwIP, i)
				if err != nil {
					sendHTTPResp(http.StatusInternalServerError, w)
					return
				}
				addedRule[i] = struct{}{}
			}
		}
		if err != nil {
			sendHTTPResp(http.StatusInternalServerError, w)
			return
		}
		sendHTTPResp(http.StatusCreated, w)
	default:
		log.Traceln(w, "Sorry, only PUT and POST methods are supported.")
		sendHTTPResp(http.StatusMethodNotAllowed, w)
	}
}

func markFromIP(ip string) string {
	octets := strings.Split(ip, ".")
	if len(octets) == 4 {
		mark := octets[3]
		return mark
	} else {
		log.Errorln("invalind gateway ip format. GwIP = ", ip)
		return ""
	}
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "PUT":
		fallthrough
	case "POST":
		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Errorln("http req read body failed.")
			sendHTTPResp(http.StatusBadRequest, w)
		}

		log.Traceln(string(body))

		//var nwSlice NetworkSlice
		var regReq RegisterReq
		//fmt.Println("parham log : http body = ", body)
		err = json.Unmarshal(body, &regReq)
		if err != nil {
			log.Errorln("Json unmarshal failed for http request")
			sendHTTPResp(http.StatusBadRequest, w)
		}
		registeredUPFs[regReq.GwIP] = regReq.CoreMac
		if err != nil {
			sendHTTPResp(http.StatusInternalServerError, w)
			return
		}
		sendHTTPResp(http.StatusCreated, w)
	default:
		log.Traceln(w, "Sorry, only PUT and POST methods are supported.")
		sendHTTPResp(http.StatusMethodNotAllowed, w)
	}
}

func execAddRule(gwip, ueip string) error {
	mark := markFromIP(gwip)
	cmd := exec.Command("iptables", "-t", "mangle", "-A", "PREROUTING", "-d", ueip, "-j", "MARK", "--set-mark", mark)
	err := cmd.Run()
	if err != nil {
		log.Errorf("Error executing command: %v", err)
		return err
	}
	log.Traceln("iptables rule applied successfully for ip : ", ueip)
	return nil
}

func execAddArp(gwip, ueip string) error {
	mark := markFromIP(gwip)
	iface := fmt.Sprint("upf", mark)
	upfmac, ok := registeredUPFs[gwip]
	if !ok {
		errtxt := fmt.Sprint("upf connected to ", gwip, " is not registered in exitlb")
		return errors.New(errtxt)
	}
	cmd := exec.Command("arp", "-s", ueip, upfmac, "-i", iface)
	err := cmd.Run()
	if err != nil {
		log.Errorf("Error executing command: %v", err)
		return err
	}
	log.Traceln("iptables rule applied successfully for ip : ", ueip)
	return nil
}

func sendHTTPResp(status int, w http.ResponseWriter) {
	w.WriteHeader(status)
	w.Header().Set("Content-Type", "application/json")

	resp := make(map[string]string)

	switch status {
	case http.StatusCreated:
		resp["message"] = "Status Created"
	default:
		resp["message"] = "Failed to add slice"
	}

	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Errorln("Error happened in JSON marshal. Err: ", err)
	}

	_, err = w.Write(jsonResp)
	if err != nil {
		log.Errorln("http response write failed : ", err)
	}
}
