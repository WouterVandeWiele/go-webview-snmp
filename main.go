package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gosnmp/gosnmp"
	"github.com/webview/webview"

	"github.com/sleepinggenius2/gosmi"
	"github.com/sleepinggenius2/gosmi/types"
)

//go:embed view/*
var content embed.FS

var w webview.WebView
var SNMPHandler gosnmp.Handler

var connected bool = false
var lastOID string

var mibLocationsHC = []string{"/usr/share/snmp/mibs", "/usr/share/snmp/mibs/iana", "/usr/share/snmp/mibs/ietf"}

type MibMeta struct {
	Path    string
	MibFile string
}

var modules map[string]MibMeta
var settings Settings

type SettingsV3SecurityParameters struct {
	UserName                 string `json:"user_name"`
	AuthenticationProtocol   string `json:"authentication_protocol"`
	AuthenticationPassphrase string `json:"authentication_passphrase"`
	PrivacyProtocol          string `json:"privacy_protocol"`
	PrivacyPassphrase        string `json:"privacy_passphrase"`
}

type SettingSnmpConnection struct {
	Name      string `json:"name"`
	Target    string `json:"target,omitempty"`
	Port      uint16 `json:"port,omitempty"`
	Community string `json:"community,omitempty"`
	Version   string `json:"version"`
	Timeout   uint16 `json:"timeout,omitempty"`
	Transport string `json:"transport,omitempty"`
	Retries   uint16 `json:"retries,omitempty"`

	// v3
	SecurityModel      string                       `json:"security_model,omitempty"`
	SecurityParameters SettingsV3SecurityParameters `json:"security_parameters,omitempty"`
}

type Settings struct {
	MibLocations []string                `json:"mib_locations,omitempty"`
	Connections  []SettingSnmpConnection `json:"connections,omitempty"`
}

func settingLoad() Settings {
	defaultSettings := Settings{
		MibLocations: mibLocationsHC,
	}

	rawData, err := os.ReadFile("Settings.json")
	if err != nil {
		fmt.Println("Could not read settings file:", err)
	}

	err = json.Unmarshal(rawData, &defaultSettings)

	if err != nil {
		fmt.Println("Could not parse settings file:", err)
	}

	return defaultSettings
}

func settingsNew(rawJson string) {
	defaultSettings := SettingSnmpConnection{
		SecurityParameters: SettingsV3SecurityParameters{},
	}

	fmt.Println(rawJson)

	err := json.Unmarshal([]byte(rawJson), &defaultSettings)

	if err != nil {
		fmt.Println("Could not parse new settings:", err)
	}

	settings.Connections = append(settings.Connections, defaultSettings)

	viewSettingsAddConnection(defaultSettings.Name)
	settingSave()
}

func settingSave() {
	content, err := json.MarshalIndent(settings, "", "\t")
	if err != nil {
		fmt.Println(err)
	}
	err = ioutil.WriteFile("Settings.json", content, 0644)
	if err != nil {
		log.Fatal(err)
	}
}

func settingsGetConnections() []string {
	var connections []string

	for _, connection := range settings.Connections {
		connections = append(connections, connection.Name)
	}

	return connections
}

/////////////////////////////////////

func trimExtension(fileName string) string {
	return strings.TrimSuffix(fileName, filepath.Ext(fileName))
}

func laodMibs() {
	gosmi.Init()

	modules = make(map[string]MibMeta)
	for _, mibPath := range mibLocationsHC {
		gosmi.AppendPath(mibPath)

		dir, err := os.Open(mibPath)
		if err != nil {
			fmt.Println("could not open dir:", mibPath)
		}

		files, err := dir.ReadDir(0)
		if err != nil {
			fmt.Println("could not parse mibs in dir:", mibPath)
		}

		for _, file := range files {
			if file.Type().IsRegular() {
				mibModule := trimExtension(file.Name())

				module, err := gosmi.LoadModule(mibModule)
				// _ = module
				if err != nil {
					fmt.Printf("Error loading mib module [%s]: %s\n", mibModule, err)
				} else {
					modules[module] = MibMeta{Path: mibPath, MibFile: file.Name()}
				}
				// fmt.Println(module)
			}
		}
	}

	loadedModules := gosmi.GetLoadedModules()
	fmt.Println("Loaded modules:")
	for _, loadedModule := range loadedModules {
		fmt.Printf("  %s (%s)\n", loadedModule.Name, loadedModule.Path)
	}

	viewMibTreeShow()
	viewStatusText("Mibs loaded")
}

// bits to filereader (https://stackoverflow.com/a/57583377)
type MyReader struct {
	src []byte
	pos int
}

func (r *MyReader) Read(dst []byte) (n int, err error) {
	n = copy(dst, r.src[r.pos:])
	r.pos += n
	if r.pos == len(r.src) {
		return n, io.EOF
	}
	return
}

func NewMyReader(b []byte) *MyReader { return &MyReader{b, 0} }

func expandEmbed(eFS embed.FS) (string, error) {
	// expand embedded dir into temp fs
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		return "", err
	}

	// fmt.Println("expanding to temp dir:", dir)

	err = fs.WalkDir(eFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		fileName := filepath.Join(dir, path)
		if d.IsDir() {
			// fmt.Println("dir", fileName)
			os.MkdirAll(fileName, os.ModePerm)
		} else {

			// fmt.Println("file", fileName)
			destination, err := os.Create(fileName)
			if err != nil {
				return err
			}
			defer destination.Close()
			file, err := content.ReadFile(path)
			nBytes, err := io.Copy(destination, NewMyReader(file))
			_ = nBytes
			return err
		}

		return nil
	})

	return dir, err
}

func main() {
	w = webview.New(true)
	defer snmpDisconnect()
	defer w.Destroy()

	go laodMibs()
	settings = settingLoad()

	// setup GUI
	w.SetTitle("GO Webview SNMP")
	w.SetSize(700, 600, webview.HintNone)

	d, err := expandEmbed(content)
	if err != nil {
		viewStatusText("Error expanding FS")
	}
	defer os.RemoveAll(d)

	// main_page, err := content.ReadFile("view/index.html")

	// if err != nil {
	// 	log.Panic("can't find index.html")
	// }

	index := filepath.Join(d, "view", "index.html")
	fmt.Println(index)

	w.Navigate("file://" + index)
	// w.SetHtml(string(main_page))

	// expose GO functions
	w.Bind("goSettingsNew", settingsNew)
	w.Bind("goSettingsGetConnections", settingsGetConnections)
	w.Bind("goSnmpConnect", snmpConnect)
	w.Bind("goSnmpGet", snmpGet)
	w.Bind("goSnmpGetNext", snmpGetNext)
	w.Bind("goSnmpDisconnect", snmpDisconnect)
	w.Bind("goSnmpBulkWalk", snmpBulkWalk)
	w.Bind("goSnmpCheckMib", snmpCheckMib)
	w.Bind("goViewMibTreeShow", viewMibTreeShow)
	// w.Bind("goSmiGetModules", smiGetModules)
	w.Bind("goSmiModuleTrees", smiModuleTrees)

	// testing
	go func() {
		for {
			viewStatusSpinner(false)
			time.Sleep(time.Second * 2)

			viewStatusSpinner(true)
			time.Sleep(time.Second * 2)
		}
	}()

	w.Run()
}

// view interface
func viewStatusSpinner(status bool) {
	// show or hide the status spinner
	w.Eval(fmt.Sprintf("viewSpinnerVisibility(%t)", status))
}

type snmpPduOidTable struct {
	Time  string `json:"Time"`
	OID   string `json:"OID"`
	Name  string `json:"Name"`
	Type  string `json:"Type"`
	Value string `json:"Value"`
}

func viewAppendOidTable(oidNr, oidName, oidType, oidValue string) {
	row := snmpPduOidTable{
		Time:  time.Now().String(),
		OID:   oidNr,
		Name:  oidName,
		Type:  oidType,
		Value: oidValue,
	}

	jsonRow, err := json.Marshal(row)

	if err != nil {
		viewStatusText("Error creating OID Table row")
	}

	w.Eval(fmt.Sprintf("viewAppendOidTable('%s')", string(jsonRow)))
}

func viewStatusText(text string) {
	w.Eval(fmt.Sprintf("viewStatusText('%s')", text))
}

func viewConnect(status bool) {
	w.Eval(fmt.Sprintf("viewStatusConnected(%t)", status))
}

func viewSettingsAddConnection(name string) {
	w.Eval(fmt.Sprintf("settingsAddConnection('%s')", name))
}

func viewMibTreeShow() {
	w.Eval(fmt.Sprintf("mibTreeShow()"))
}

// smi
func smiGetModules() string {
	modules := gosmi.GetLoadedModules()

	jsonBytes, _ := json.Marshal(modules)

	return string(jsonBytes)
}

func smiModuleTrees() string {

	type moduleInfo struct {
		Module gosmi.SmiModule
		Nodes  []gosmi.SmiNode
		Types  []gosmi.SmiType
	}

	returnValue := make(map[string]moduleInfo)

	_ = os.MkdirAll("mibModules", os.ModePerm)

	for _, module := range gosmi.GetLoadedModules() {
		m, err := gosmi.GetModule(module.Name)
		if err != nil {
			fmt.Printf("ModuleTrees Error: %s\n", err)
			continue
		}

		nodes := m.GetNodes()
		types := m.GetTypes()

		returnValue[module.Name] = moduleInfo{
			Module: m,
			Nodes:  nodes,
			Types:  types,
		}

		content, err := json.MarshalIndent(returnValue[module.Name], "", "\t")

		if err != nil {
			fmt.Println(err)
		}
		err = ioutil.WriteFile(fmt.Sprintf("mibModules/%s.json", module.Name), content, 0644)
		if err != nil {
			log.Fatal(err)
		}

		// os.Stdout.Write(jsonBytes)
	}
	jsonBytes, _ := json.Marshal(returnValue)

	return string(jsonBytes)
}

func smiModuleTree(module string) string {
	m, err := gosmi.GetModule(module)
	if err != nil {
		fmt.Printf("ModuleTrees Error: %s\n", err)
	}

	nodes := m.GetNodes()
	types := m.GetTypes()

	jsonBytes, _ := json.Marshal(struct {
		Module gosmi.SmiModule
		Nodes  []gosmi.SmiNode
		Types  []gosmi.SmiType
	}{
		Module: m,
		Nodes:  nodes,
		Types:  types,
	})

	content, err := json.MarshalIndent(struct {
		Module gosmi.SmiModule
		Nodes  []gosmi.SmiNode
		Types  []gosmi.SmiType
	}{
		Module: m,
		Nodes:  nodes,
		Types:  types,
	}, "", "\t")

	if err != nil {
		fmt.Println(err)
	}
	err = ioutil.WriteFile(fmt.Sprintf("mibModules/%s.json", module), content, 0644)
	if err != nil {
		log.Fatal(err)
	}

	return string(jsonBytes)
}

// snmp
func snmpConnect(name string) {

	nameFound := false
	for _, availale_name := range settingsGetConnections() {
		if availale_name == name {
			nameFound = true
		}
	}

	if !nameFound {
		viewStatusText("no valid connection")
		return
	}

	var conParams SettingSnmpConnection

	for _, settings := range settings.Connections {
		if settings.Name == name {
			conParams = settings
		}

	}

	fmt.Printf("ver: %s, co: %s, ip: %s\n", conParams.Version, conParams.Community, conParams.Target)

	SNMPHandler = gosnmp.NewHandler()

	ipAddress := "127.0.0.1"
	if conParams.Target != "" {
		ipAddress = conParams.Target
	}
	SNMPHandler.SetTarget(ipAddress)

	community := SNMPHandler.Community()
	if conParams.Community != "" {
		community = conParams.Community
	}
	SNMPHandler.SetCommunity(community)

	timeout := time.Second * 10
	if conParams.Timeout != 0 {
		timeout = time.Second * time.Duration(conParams.Timeout)
	}
	SNMPHandler.SetTimeout(timeout)

	// TODO: no SetTransport on SNMPHandler ????
	// transport := "udp"
	// if conParams.Transport != "" {
	// 	transport = conParams.Transport
	// }
	// SNMPHandler.

	var retries uint16 = 3
	if conParams.Retries != 0 {
		retries = conParams.Retries
	}
	SNMPHandler.SetRetries(int(retries))

	var port uint16 = 161
	if conParams.Port != 0 {
		port = conParams.Port
	}
	SNMPHandler.SetPort(port)

	// SNMPv3
	if conParams.SecurityModel != "" {
		if conParams.SecurityModel == "SnmpV3SecurityModel" {
			SNMPHandler.SetSecurityModel(gosnmp.UserSecurityModel)
		}
	}

	mAuthProto := make(map[string]gosnmp.SnmpV3AuthProtocol)
	mAuthProto["No Auth"] = gosnmp.NoAuth
	mAuthProto["MD5"] = gosnmp.MD5
	mAuthProto["SHA"] = gosnmp.SHA
	mAuthProto["SHA224"] = gosnmp.SHA224
	mAuthProto["SHA256"] = gosnmp.SHA256
	mAuthProto["SHA384"] = gosnmp.SHA384
	mAuthProto["SHA512"] = gosnmp.SHA512

	mPrivProto := make(map[string]gosnmp.SnmpV3PrivProtocol)
	mPrivProto["No Priv"] = gosnmp.NoPriv
	mPrivProto["DES"] = gosnmp.DES
	mPrivProto["AES"] = gosnmp.AES
	mPrivProto["AES192"] = gosnmp.AES192
	mPrivProto["AES256"] = gosnmp.AES256
	mPrivProto["AES192C"] = gosnmp.AES192C
	mPrivProto["AES256C"] = gosnmp.AES256C

	if conParams.SecurityParameters.AuthenticationProtocol != "" && conParams.SecurityParameters.PrivacyProtocol != "" {
		security := gosnmp.UsmSecurityParameters{
			UserName:                 conParams.SecurityParameters.UserName,
			AuthenticationProtocol:   mAuthProto[conParams.SecurityParameters.AuthenticationProtocol],
			AuthenticationPassphrase: conParams.SecurityParameters.AuthenticationPassphrase,
			PrivacyProtocol:          mPrivProto[conParams.SecurityParameters.PrivacyProtocol],
			PrivacyPassphrase:        conParams.SecurityParameters.PrivacyPassphrase,
		}
		SNMPHandler.SetSecurityParameters(&security)
	}

	var err error
	if conParams.Version == "1" {
		SNMPHandler.SetVersion(gosnmp.Version1)
	} else if conParams.Version == "2" {
		SNMPHandler.SetVersion(gosnmp.Version2c)
	} else if conParams.Version == "3" {
		SNMPHandler.SetVersion(gosnmp.Version3)
	}

	err = SNMPHandler.Connect()
	connected = false
	if err != nil {
		log.Fatalf("Connect() err: %v", err)
	} else {
		connected = true
		lastOID = "1.1.0"
	}
	viewStatusText(fmt.Sprintf("Connected to %s", ipAddress))
	viewConnect(true)
}

func snmpGet(oid string) {
	oids := []string{oid}
	// oids := []string{"1.3.6.1.2.1.1.4.0", "1.3.6.1.2.1.1.7.0"}

	result, err := SNMPHandler.Get(oids) // Get() accepts up to g.MAX_OIDS
	if err != nil {
		viewStatusText("SNMP get called, but no connection available")
		// log.Fatalf("Get() err: %v", err2)
	} else {
		for i, variable := range result.Variables {
			fmt.Printf("%d: oid: %s ", i, variable.Name)

			snmpPrintWalkValue(variable)
		}
	}
}

func snmpGetNext() {
	oids := []string{lastOID}

	result, err := SNMPHandler.GetNext(oids)
	if err != nil {
		viewStatusText("SNMP getNext called, but no connection available")
		fmt.Println("GetNext", err)
		// log.Fatalf("Get() err: %v", err2)
	} else {
		for i, variable := range result.Variables {
			fmt.Printf("%d: oid: %s ", i, variable.Name)

			snmpPrintWalkValue(variable)
		}
	}
}

func snmpBulkWalk(oid string) {
	err := SNMPHandler.BulkWalk(oid, snmpPrintWalkValue)
	if err != nil {
		fmt.Printf("Walk Error: %v\n", err)
		viewStatusText("SNMP bulk walk, error during get")
	}
}

func snmpPrintWalkValue(variable gosnmp.SnmpPDU) error {
	fmt.Printf("%s = ", variable.Name)

	var varType string
	var value string = fmt.Sprintf("%d", variable.Value)

	switch variable.Type {
	case gosnmp.OctetString:
		varType = "OctetString"
		value = string(variable.Value.([]byte))
	case gosnmp.EndOfContents:
		varType = "EndOfContents/UnknownType"
	case gosnmp.Boolean:
		varType = "Boolean"
	case gosnmp.Integer:
		varType = "Integer"
	case gosnmp.BitString:
		varType = "BitString"
	case gosnmp.Null:
		varType = "Null"
	case gosnmp.ObjectIdentifier:
		varType = "ObjectIdentifier"
		value = variable.Value.(string)

	// case gosnmp.ObjectDescriptionIPAddress:
	// 	varType = "ObjectDescriptionIPAddress"
	case gosnmp.Counter32:
		varType = "Counter32"
	case gosnmp.Gauge32:
		varType = "Gauge32"
	case gosnmp.TimeTicks:
		varType = "TimeTicks"
	case gosnmp.Opaque:
		varType = "Opaque"
	case gosnmp.NsapAddress:
		varType = "NsapAddress"
	case gosnmp.Counter64:
		varType = "Counter64"
	case gosnmp.Uinteger32:
		varType = "Uinteger32"
	case gosnmp.OpaqueFloat:
		varType = "OpaqueFloat"
	case gosnmp.OpaqueDouble:
		varType = "OpaqueDouble"
	case gosnmp.NoSuchObject:
		varType = "NoSuchObject"
	case gosnmp.NoSuchInstance:
		varType = "NoSuchInstance"
	case gosnmp.EndOfMibView:
		varType = "EndOfMibView"
	default:
		// value = gosnmp.ToBigInt(variable.Value)
		// fmt.Printf("number: %d\n", value)
		// viewAppendOidTable(variable.Name, "number", fmt.Sprintf("%d", value))
	}

	lastOID = variable.Name
	node, err := gosmi.GetNodeByOID(types.OidMustFromString(variable.Name))
	if err != nil {
		fmt.Printf("Subtree Error: %s\n", err)
	}

	viewAppendOidTable(variable.Name, node.Name, varType, value)
	return nil
}

func snmpSet(oid, _type, value string) {
	pdu := gosnmp.SnmpPDU{
		Value: value,
		Name:  oid,
	}

	switch _type {
	case "OctetString":
		pdu.Type = gosnmp.OctetString
	case "EndOfContents":
		pdu.Type = gosnmp.EndOfContents
	case "Boolean":
		pdu.Type = gosnmp.Boolean
	case "Integer":
		pdu.Type = gosnmp.Integer
	case "BitString":
		pdu.Type = gosnmp.BitString
	case "Null":
		pdu.Type = gosnmp.Null
	case "ObjectIdentifier":
		pdu.Type = gosnmp.ObjectIdentifier
	// case "ObjectDescriptionIPAddress":
	// 	pdu.Type = gosnmp.ObjectDescriptionIPAddress
	case "Counter32":
		pdu.Type = gosnmp.Counter32
	case "Gauge32":
		pdu.Type = gosnmp.Gauge32
	case "TimeTicks":
		pdu.Type = gosnmp.TimeTicks
	case "Opaque":
		pdu.Type = gosnmp.Opaque
	case "NsapAddress":
		pdu.Type = gosnmp.NsapAddress
	case "Counter64":
		pdu.Type = gosnmp.Counter64
	case "Uinteger32":
		pdu.Type = gosnmp.Uinteger32
	case "OpaqueFloat":
		pdu.Type = gosnmp.OpaqueFloat
	case "OpaqueDouble":
		pdu.Type = gosnmp.OpaqueDouble
		// case "NoSuchObject":
		// 	pdu.Type = gosnmp.NoSuchObject
		// case "NoSuchInstance":
		// 	pdu.Type = gosnmp.NoSuchInstance
		// case "EndOfMibView":
		// 	pdu.Type = gosnmp.EndOfMibView
	}
	pdus := []gosnmp.SnmpPDU{pdu}
	result, err := SNMPHandler.Set(pdus)

	if err != nil {
		viewStatusText("Error setting value")
	}

	for i, variable := range result.Variables {
		fmt.Printf("%d: oid: %s ", i, variable.Name)

		snmpPrintWalkValue(variable)
	}
}

func snmpCheckMib(oid string) string {
	node, err := gosmi.GetNodeByOID(types.OidMustFromString(oid))
	if err != nil {
		fmt.Printf("Subtree Error: %s\n", err)
	}

	jsonBytes, _ := json.Marshal(node)

	return string(jsonBytes)
}

func snmpDisconnect() {
	if connected == true {
		SNMPHandler.Close()
		connected = false
		viewConnect(false)

		viewStatusText("Disconnected")
	}
}
