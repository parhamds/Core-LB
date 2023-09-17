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
	GwIP string `json:"gwip"`
	//Teid []string `json:"teid"`
	Ip []string `json:"ip"`
}

type RegisterReq struct {
	GwIP    string `json:"gwip"`
	CoreMac string `json:"coremac"`
}
type operation int

const (
	ruleAdd operation = iota
	ruleDel
	arpAdd
	arpDel
)

var addedRule map[string]string
var registeredUPFs map[string]string // [gwip] coremac

func main() {
	log.SetLevel(log.TraceLevel)
	log.Traceln("application started")
	addedRule = make(map[string]string)
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
		if err != nil || rulereq.GwIP == "" || len(rulereq.Ip) == 0 {
			log.Errorln("Json unmarshal failed for http request")
			sendHTTPResp(http.StatusBadRequest, w)
		}
		for _, i := range rulereq.Ip {
			added := false
			gwip, added := addedRule[i]
			if !added {
				err = execRule(rulereq.GwIP, i, ruleAdd)
				if err != nil {
					sendHTTPResp(http.StatusInternalServerError, w)
					return
				}
				err = execArp(rulereq.GwIP, i, arpAdd)
				if err != nil {
					sendHTTPResp(http.StatusInternalServerError, w)
					return
				}
				addedRule[i] = rulereq.GwIP
				continue
			}
			if rulereq.GwIP != gwip {
				err = execRule(gwip, i, ruleDel)
				if err != nil {
					sendHTTPResp(http.StatusInternalServerError, w)
					return
				}
				err = execArp(gwip, i, arpDel)
				if err != nil {
					sendHTTPResp(http.StatusInternalServerError, w)
					return
				}
				err = execRule(rulereq.GwIP, i, ruleAdd)
				if err != nil {
					sendHTTPResp(http.StatusInternalServerError, w)
					return
				}
				err = execArp(rulereq.GwIP, i, arpAdd)
				if err != nil {
					sendHTTPResp(http.StatusInternalServerError, w)
					return
				}
				addedRule[i] = rulereq.GwIP
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
		if err != nil || regReq.CoreMac == "" || regReq.GwIP == "" {
			log.Errorln("Json unmarshal failed for http request")
			sendHTTPResp(http.StatusBadRequest, w)
		}
		if regUPFCore, ok := registeredUPFs[regReq.GwIP]; ok && regUPFCore == regReq.CoreMac {
			sendHTTPResp(http.StatusCreated, w)
			return
		}
		registeredUPFs[regReq.GwIP] = regReq.CoreMac
		sendHTTPResp(http.StatusCreated, w)
		return

	default:
		log.Traceln(w, "Sorry, only PUT and POST methods are supported.")
		sendHTTPResp(http.StatusMethodNotAllowed, w)
	}
}

func execRule(gwip, ueip string, op operation) error {
	mark := markFromIP(gwip)
	var oper string
	switch op {
	case ruleAdd:
		oper = "-A"
	case ruleDel:
		oper = "-D"
	}
	cmd := exec.Command("iptables", "-t", "mangle", oper, "PREROUTING", "-d", ueip, "-j", "MARK", "--set-mark", mark)
	log.Traceln("executing command : ", cmd.String())
	combinedOutput, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Error executing command: %v\nCombined Output: %s", cmd.String(), combinedOutput)
		return err
	}
	log.Traceln("iptables rule applied successfully for ip : ", ueip)
	return nil
}

func execArp(gwip, ueip string, op operation) error {
	mark := markFromIP(gwip)
	iface := fmt.Sprint("upf", mark)
	var cmd *exec.Cmd

	switch op {
	case arpAdd:
		upfmac, ok := registeredUPFs[gwip]
		if !ok {
			errtxt := fmt.Sprint("upf connected to ", gwip, " is not registered in exitlb")
			return errors.New(errtxt)
		}
		cmd = exec.Command("arp", "-s", ueip, upfmac, "-i", iface)
	case arpDel:
		cmd = exec.Command("arp", "-d", ueip, "-i", iface)
	}
	log.Traceln("executing command : ", cmd.String())
	combinedOutput, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Error executing command: %v\nCombined Output: %s", cmd.String(), combinedOutput)
		return err
	}
	log.Traceln("static arp applied successfully for ip : ", ueip)
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
