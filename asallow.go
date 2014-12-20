package main

import (
	"code.google.com/p/gcfg"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"strings"
)

const PREFIX_URI="https://stat.ripe.net/data/announced-prefixes/data.json?resource="

var ipset_count int = 0
var ipset_string string = ""

type cfg struct {
    Main struct {
        Allow []string
        ASN []string
    }
}

func readconfig(cfgfile string) cfg {
	data, err := ioutil.ReadFile(cfgfile)
    if err != nil {
        log.Fatal(err)
    }
	cfgStr := string(data)
	cfg := struct {
		Main struct {
			Allow []string
            ASN []string
		}
	}{}
	err = gcfg.ReadStringInto(&cfg, cfgStr)
	if err != nil {
		log.Fatal("Failed to parse "+ cfgfile +":",err)
	}
	return cfg
}

func getAS(ASnumber string) []byte  {
	resp, err := http.Get(PREFIX_URI + ASnumber)
	if err != nil {
		log.Fatal("site not available")
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("can not read body")
	}
    return body
}

func doipset() {
	cmd := exec.Command("ipset", "-!", "create", "AS_allow", "hash:net", "comment")
	err := cmd.Run()
    if err != nil {
        log.Fatal(err)
    }
    cmd = exec.Command("ipset","-!","create","AS_allow_swap","hash:net", "comment")
    cmd.Run()
    if err != nil {
        log.Fatal(err)
    }
	cmd = exec.Command("ipset", "-!", "restore")
	cmd.Stdin = strings.NewReader(ipset_string)
	err = cmd.Run()
	if err != nil {
		log.Fatal("ip addresses could not be added", err)
	}
    cmd = exec.Command("ipset","swap","AS_allow","AS_allow_swap")
    cmd.Run()
    if err != nil {
        log.Fatal(err)
    }
    cmd = exec.Command("ipset","destroy","AS_allow_swap")
    cmd.Run()
}

func parseBody(body []byte, ASnumber string) {
	dec := json.NewDecoder(strings.NewReader(string(body)))
	var mapstring map[string]interface{}
	if err := dec.Decode(&mapstring); err != nil {
		log.Fatal(err)
	}
	datamap := mapstring["data"]
	mapstring = datamap.(map[string]interface{})
	prefixes := mapstring["prefixes"]
	prefixes_array := prefixes.([]interface{})
	for _, prefix_element := range prefixes_array {
		mapstring = prefix_element.(map[string]interface{})
		if strings.Contains(mapstring["prefix"].(string), "::") != true {
			ipset_string += "add AS_allow_swap " + mapstring["prefix"].(string) + " comment AS" + ASnumber + "\n"
			ipset_count += 1
		}
	}
}

func addAllowed(allowed []string) {
	for _, el := range allowed {
		ipset_string += "add AS_allow_swap " + el + " comment \"read from asallow.conf\"\n"
	}
}

func main() {
    cfgfile := flag.String("conf","asallow.conf","a valid config file")
	flag.Parse()

	cfg := readconfig(*cfgfile)
    for _,ASN := range cfg.Main.ASN {
        body := getAS(ASN)
        parseBody(body,ASN)
    }
    addAllowed(cfg.Main.Allow)
    doipset()

	fmt.Printf("%v ip addresses added\n", ipset_count)
}
