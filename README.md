# GO Webview SNMP browser

Simple SNMP browser written in GO.

Backend:
- https://github.com/gosnmp/gosnmp
- https://github.com/sleepinggenius2/gosmi
Frontend: 
- https://github.com/webview/webview
- https://getbootstrap.com/docs/5.2/getting-started/introduction/
- https://datatables.net/

Features:
- SNMP v1 v2c and v3 
- common SNMP operations GET, GETNEXT, SET, BULKWALK
- save connection info for quick access
- parsing of MIB files (provide the location in the settings file)
  - gosmi makes a json represantation of all MIB files found and exports them to `mibModules` folder next to the executable.
- OID Table view:
  - export reults to clipboard, csv*, excel* and pdf* (*check the downloads folder)
  - sorting and filtering of results
- MIB Explorer view:
  - explore the MIBs that are parsed by gosmi
  (currently only the meta inforamtion)


## example settings

```json
{
	"mib_locations": [
		"/usr/share/snmp/mibs",
		"/usr/share/snmp/mibs/iana",
		"/usr/share/snmp/mibs/ietf"
	],
	"connections": [
		{
			"name": "Default v2",
			"target": "127.0.0.1",
			"community": "public",
			"version": "2",
			"transport": "UDP",
			"security_parameters": {
				"user_name": "",
				"authentication_protocol": "",
				"authentication_passphrase": "",
				"privacy_protocol": "",
				"privacy_passphrase": ""
			}
		}
	]
}
```