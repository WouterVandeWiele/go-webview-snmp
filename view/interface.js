var datatable = null;

$(document).ready( function () {
    datatable = $('#oidTable').DataTable({
        "searching": true,
        fixedHeader: true,
        colReorder: true,
        dom: 'Bfrtip',
        columns: [
            {data: 'Time'},
            {data: 'OID'},
            {data: 'Name'},
            {data: 'Type'},
            {data: 'Value'},
        ],
        buttons: [
            {
                extend: 'copy',
                split: ['csv', 'excel', 'pdf']
            },
            'colvis',
            {
                extend: 'searchBuilder',
                config: {
                    depthLimit: 2
                }
            },
            'searchPanes',
            // {
            //     extend: 'collection',
            //     text: 'Info',
            //     buttons: [
            //         { text: 'MibInfo',   action: function () { console.log("mibinfo") } }
            //     ],
            //     fade: true
            // }
        ],
        paging: false,
        scrollY: "100px",
        scrollCollapse: true,
        pageResize: true
    });

    $('#oidTable tbody').on('click', 'tr', function(){
        var data = datatable.row( $(this) ).data();
        document.getElementById('actionOID').value = data.OID;
        document.getElementById('actionType').value = data.Type;
        goSnmpCheckMib(data.OID).then((res) => console.log(res));
    });
});

$(document).ready( function () {
    setTableHeight();
});

$(document).ready( function () {
    fillConnectionsDropdown();
});

$('#datatablesSearch').keyup(function(){
    datatable.search($(this).val()).draw() ;
});


const tableOffset = 430;
function setTableHeight() {
    windowHeight = window.innerHeight;
    totalExplorer = (windowHeight - tableOffset) + "px";
    
    d = document.getElementsByClassName("dataTables_scroll")[0];
    d.children[1].style.maxHeight = totalExplorer;
}

addEventListener("resize", (event) => {setTableHeight()});

///////////////////////////////////////////////////////////////////

function clearOidTableTable() {
    datatable.clear();
    datatable.draw();
}

function viewStatusText(text) {
    document.getElementById("statusText").innerHTML = text;

    console.log('viewStatusText', text);
}

function viewStatusConnected(status) {
    badge = document.getElementById("badgeStatusConnected");
    button = document.getElementById("serverConnect");
    actions = document.getElementById("actionsFieldset");

    if (status == true) {
        badge.classList.add("bg-success");
        badge.classList.remove("bg-danger");
        button.innerHTML = button.innerHTML.replace(/Connect/, 'Disconnect');
        button.onclick = callDisconnect;
        actions.disabled = false;
    } else {
        badge.classList.add("bg-danger");
        badge.classList.remove("bg-success");
        button.innerHTML = button.innerHTML.replace(/Disconnect/, 'Connect');
        button.onclick = callConnect;
        actions.disabled = true;
    }
}

function viewSpinnerVisibility(status) {
    spinner = document.getElementById("spinnerStatus");
    
    if (status == true) {
        spinner.classList.remove("invisible");
    } else {
        spinner.classList.add("invisible");
    }
}

function callConnect() {
    connection = document.getElementById("serverConnections").value;
    goSnmpConnect(connection);

}

function callDisconnect() {
    goSnmpDisconnect();
}

function snmpGet() {
    oid = document.getElementById("actionOID").value;

    console.log("get>", oid);
    goSnmpGet(oid);
}

function snmpBulkWalk() {
    oid = document.getElementById("actionOID").value;

    console.log("bulk walk>", oid);
    goSnmpBulkWalk(oid);
}

function viewAppendOidTable(jsonRaw) {
    datatable.row.add(JSON.parse(jsonRaw)).draw();
}

function settingsAddConnection(name) {
    if (name != "") {
        console.log(name);

        select = document.getElementById("serverConnections");

        var opt = document.createElement('option');
        opt.innerHTML = name;
        select.appendChild(opt);
    }
}

function fillConnectionsDropdown() {
    goSettingsGetConnections().then((connectionList) => {
        connectionList.forEach(element => {
            settingsAddConnection(element);
        });
    });
}

function settingsNewConnection() {
    console.log("settingsNewConnection");
    
    auth = {}
    auth.name = document.getElementById("serverConnectionName").value;
    auth.target = document.getElementById("serverIP").value;
    auth.port = parseInt(document.getElementById("serverPort").value);
    auth.timeout = parseInt(document.getElementById("serverTimeout").value);
    auth.transport = document.getElementById("serverTransport").value;
    auth.retries = parseInt(document.getElementById("serverRetries").value);

    if (document.getElementById("listGroupRadiosSnmpVersion1").checked) {
        auth.community = document.getElementById("serverCv1").value;
        auth.version = "1";
    } else if (document.getElementById("listGroupRadiosSnmpVersion2").checked) {
        auth.community = document.getElementById("serverCv2").value;
        auth.version = "2"
    } else if (document.getElementById("listGroupRadiosSnmpVersion3").checked) {
        auth.security_model = document.getElementById("serverSecurityModel").value;
        auth.security_parameters = {}
        auth.security_parameters.user_name = document.getElementById("serverUserName").value;
        auth.security_parameters.authentication_protocol = document.getElementById("serverAuthProtocol").value;
        auth.security_parameters.authentication_passphrase = document.getElementById("serverAuthPass").value;
        auth.security_parameters.privacy_protocol = document.getElementById("serverPrivProtocol").value;
        auth.security_parameters.privacy_passphrase = document.getElementById("serverPrivPass").value;
        auth.version = "3"
    }
    goSettingsNew(JSON.stringify(auth));

    modal = document.getElementById("newConnectionModal");
    bootstrap.Modal.getInstance(modal).hide();
}

function settingsGetName() {
    
    goSettingsGetConnections().then((connectionList) => {
        connectionName = document.getElementById("serverConnectionName");
        saveButton = document.getElementById("serverConnectionSave");

        nameFeedback = document.getElementById("serverConnectionNameFeedback");

        if (connectionName.value == ""){
            nameFeedback.classList.remove("valid-feedback");
            connectionName.classList.remove("is-valid");
            nameFeedback.classList.add("invalid-feedback");
            connectionName.classList.add("is-invalid");
            saveButton.disabled = true;
            nameFeedback.innerHTML = "name can't be empty";
        } else if (connectionList.includes(connectionName.value)) {
            nameFeedback.classList.remove("valid-feedback");
            connectionName.classList.remove("is-valid");
            nameFeedback.classList.add("invalid-feedback");
            connectionName.classList.add("is-invalid");
            saveButton.disabled = true;
            nameFeedback.innerHTML = "name already used";
        } else {
            nameFeedback.classList.remove("invalid-feedback");
            connectionName.classList.remove("is-invalid");
            nameFeedback.classList.add("valid-feedback");
            connectionName.classList.add("is-valid");
            saveButton.disabled = false;
            nameFeedback.innerHTML = "";
        }
    });
}

function mibTreeSearchFocus() {
    var value = document.getElementById("mibTreeSearch").value;
    var element = document.getElementById("MIB-" + value);
    element.scrollIntoView({behavior: "smooth", block: "start", inline: "nearest"});     
}

function mibTreeScrollTop() {
    document.documentElement.scrollTop = 0; 
}

function mibTreeShow() {
    goSmiModuleTrees().then((modules_raw) => {
        modules = JSON.parse(modules_raw);

        var parentNode = document.getElementById("mibtree");
        var selectNode = document.getElementById("mibTreeSearchList");
        Object.keys(modules).forEach(function(key) {
            // DOM element

            var value = modules[key];

            var name = value.Module.Name.replace(/[^a-z0-9\-]/gi, '');
            var card = document.createElement("div");
            card.classList.add("card", "my-2");
            card.id = `MIB-${name}`;

            var card_header = document.createElement("h5");
            card_header.classList.add("card-header");
            card_header.innerHTML = name;

            var card_body = document.createElement("div");
            card_body.classList.add("card-body")

            var card_list = document.createElement("ul");
            // card_list.appendChild(document.createElement("li").innerHTML = `Name - ${element.Name}`);
            var node = document.createElement("li");
            node.innerHTML = `Language - ${value.Module.Language}`;
            card_list.appendChild(node);

            var node = document.createElement("li");
            node.innerHTML = `Organization - ${value.Module.Organization}`.replace("/\n/g", "<br>");
            card_list.appendChild(node);

            var node = document.createElement("li");
            node.innerHTML = `Path - ${value.Module.Path}`;
            card_list.appendChild(node);

            var node = document.createElement("li");
            node.innerHTML = `Reference - ${value.Module.Reference}`;
            card_list.appendChild(node);

            var node = document.createElement("li");
            node.innerHTML = `Description - ${value.Module.Description}`;
            card_list.appendChild(node);

            var node = document.createElement("li");
            node.innerHTML = `ContactInfo - ${value.Module.ContactInfo}`;
            card_list.appendChild(node);

            // accordion node & types
            var accordion = `<div class="accordion accordion-flush">
<div class="accordion-item">
    <h2 class="accordion-header" id="NODES-HEADING-${name}">
        <button class="accordion-button collapsed" type="button" data-bs-toggle="collapse" data-bs-target="#NODES-${name}" aria-expanded="false" aria-controls="NODES-${name}">
            Node Elements ${name}
        </button>
    </h2>
    <div id="NODES-${name}" class="accordion-collapse collapse" aria-labelledby="NODES-HEADING-${name}" data-bs-parent="#accordionFlushExample">
        <div class="accordion-body">
            Placeholder content for this accordion, which is intended to demonstrate the <code>.accordion-flush</code> class. This is the first item's accordion body.
        </div>
    </div>
</div>
<div class="accordion-item">
    <h2 class="accordion-header" id="TYPES-HEADING-${name}">
        <button class="accordion-button collapsed" type="button" data-bs-toggle="collapse" data-bs-target="#TYPES-${name}" aria-expanded="false" aria-controls="TYPES-${name}">
            Type Elements ${name}
        </button>
    </h2>
    <div id="TYPES-${name}" class="accordion-collapse collapse" aria-labelledby="TYPES-HEADING-${name}" data-bs-parent="#accordionFlushExample">
        <div class="accordion-body">
            Placeholder content for this accordion, which is intended to demonstrate the <code>.accordion-flush</code> class. This is the second item's accordion body. Let's imagine this being filled with some actual content.
        </div>
    </div>
</div>
</div>
`;
            
            card_body.appendChild(card_list);
            card.appendChild(card_header);
            card.appendChild(card_body);
            card.insertAdjacentHTML("beforeend", accordion);
            parentNode.appendChild(card);

            // select -> option search
            var option = document.createElement("option");
            option.setAttribute("data-tokens", name);
            option.innerHTML = name;
            selectNode.appendChild(option);

        });
    });

    document.getElementById("mibTreeLoading").setAttribute("hidden", true);
}