var browserDocs = "\
<h4>Browser object</h4>\
<h5>Methods</h5>\
<ul>\
<li>open('https://google.com') - navigate the browser.</li>\
<li>title() - Returns the title of the current page.</li>\
<li>body() - Returns the body of the current page.</li>\
<li>bodyRaw() - Returns the raw body of the current page.</li>\
<li>cookies() - Returns the cookies set on the current session.</li>\
<li>setUserAgent('some user agent') - Sets the browsers user agent.</li>\
<li>setChromeAgent() - Sets the user agent to Chrome.</li>\
<li>setFirefoxAgent() - Sets the user agent to Firefox.</li>\
<li>form('#form') - Returns a form object.\
  <ul style='margin-left: 10px;'>\
    <li>set('selector', 'value') - Sets an input.</li>\
    <li>submit() - Submits the form.</li>\
  </ul>\
</li>\
<li>find('#form') - Returns a selector object.\
  <ul style='margin-left: 10px;'>\
    <li>text() - Returns the text in the selector.</li>\
    <li>html() - Returns the HTML in the selector.</li>\
  </ul>\
</li>\
<li>cookies() - Returns the cookies set on the current session.</li>\
</ul>\
";

var codeGlobals = [
  {
    name: 'console',
    value: 'console',
    meta: 'logger',
    score: 110,
    reference: {
      heading: 'console',
      kind: 'global object',
      detail: 'Contains methods to log debugging information.',
    },
  },
  {
    name: 'owner',
    value: 'owner',
    meta: 'account',
    score: 110,
    reference: {
      heading: 'owner',
      kind: 'global object',
      detail: 'Information pertaining to the account owning the integration.',
    },
  },
  {
    name: 'context',
    value: 'context',
    meta: 'execution context',
    score: 110,
    reference: {
      heading: 'context',
      kind: 'global object',
      detail: 'Information pertaining to the context/reason this integration was executed.',
    },
  },
  {
    name: 'cronspec',
    value: 'cronspec',
    meta: 'CRON trigger only',
    score: 105,
    reference: {
      heading: 'cronspec',
      kind: 'global object',
      detail: 'Only available when triggered by a CRON. Set to the cronspec which triggered the current execution',
    },
  },
  {
    name: 'request',
    value: 'request',
    meta: 'HTTP trigger only',
    score: 110,
    reference: {
      heading: 'request',
      kind: 'global object',
      detail: 'Only available when triggered by a HTTP request. Information/methods pertaining to the HTTP request which triggered the run, such as the URL or Host.',
    },
  },
  {
    name: 'browser',
    value: 'browser',
    meta: 'method',
    score: 110,
    reference: {
      heading: 'browser()',
      kind: 'method',
      detail: 'Creates a \'fake browser\'. Press Control-B to see a reference of methods for this object.',
    },
  },
  {
    name: 'email',
    value: 'email',
    meta: 'emailer',
    score: 110,
    reference: {
      heading: 'email',
      kind: 'global object',
      detail: 'Has methods and constants to send email.',
    },
  },
  {
    name: 'kv',
    value: 'kv',
    meta: 'storage',
    score: 110,
    reference: {
      heading: 'kv',
      kind: 'global object',
      detail: 'Key value store, where you can get/set objects with a string key.',
    },
  },
  {
    name: 'fs',
    value: 'fs',
    meta: 'filesystem',
    score: 110,
    reference: {
      heading: 'fs',
      kind: 'global object',
      detail: 'Contains methods to read and write to the virtual filesystem.',
    },
  },
  {
    name: 'datastore',
    value: 'datastore',
    meta: 'databases',
    score: 110,
    reference: {
      heading: 'datastore',
      kind: 'global object',
      detail: 'Contains methods to insert, delete, and query from datastores you have access to.',
    },
  },
  {
    name: 'web',
    value: 'web',
    meta: 'www',
    score: 110,
    reference: {
      heading: 'web',
      kind: 'global object',
      detail: 'Contains methods to query HTTP(S) servers.',
    },
  },
  {
    name: 't',
    value: 't',
    meta: 'time',
    score: 110,
    reference: {
      heading: 't',
      kind: 'global object',
      detail: 'Contains methods to manipulate time.',
    },
  },
  {
    name: 'gcp',
    value: 'gcp',
    meta: 'cloud',
    score: 110,
    reference: {
      heading: 'gcp',
      kind: 'global object',
      detail: 'Contains methods to interact with Google Cloud Platform.',
    },
  },
];

var codeSubs = [
  {
    prefix: 'console.',
    name: 'log',
    value: 'log',
    meta: 'prints to console',
    score: 100,
    reference: {
      heading: 'console.log()',
      kind: 'method',
      detail: 'Logs arguments to the console.',
    },
  },
  {
    prefix: 'console.',
    name: 'warn',
    value: 'warn',
    meta: 'warn to console',
    score: 100,
    reference: {
      heading: 'console.warn()',
      kind: 'method',
      detail: 'Logs arguments to the console as a warning.',
    },
  },
  {
    prefix: 'console.',
    name: 'error',
    value: 'error',
    meta: 'error to console',
    score: 100,
    reference: {
      heading: 'console.error()',
      kind: 'method',
      detail: 'Logs arguments to the console as an error.',
    },
  },
  {
    prefix: 'console.',
    name: 'data',
    value: 'data',
    meta: 'object debug',
    score: 100,
    reference: {
      heading: 'console.data(<object>)',
      kind: 'method',
      detail: 'Logs a single object to the console.',
    },
  },
  {
    prefix: 'context.',
    name: 'run_id',
    value: 'run_id',
    meta: 'string',
    score: 110,
    reference: {
      heading: 'context.run_id',
      kind: 'string',
      detail: '8 character identifier, uniquely identifying the current run.',
    },
  },
  {
    prefix: 'context.',
    name: 'run_reason',
    value: 'run_reason',
    meta: 'string',
    score: 110,
    reference: {
      heading: 'context.run_reason',
      kind: 'string',
      detail: 'Cause of the execution. Typically "manual", "CRON", "HTTP" etc',
    },
  },
  {
    prefix: 'context.',
    name: 'trigger_id',
    value: 'trigger_id',
    meta: 'string',
    score: 110,
    reference: {
      heading: 'context.trigger_id',
      kind: 'int',
      detail: 'If a trigger caused the current execution, this will be the ID of the trigger.',
    },
  },
  {
    prefix: 'context.',
    name: 'start_time',
    value: 'start_time',
    meta: 'time.Time',
    score: 110,
    reference: {
      heading: 'context.start_time',
      kind: 'golang time.Time',
      detail: 'Time at which execution started.',
    },
  },
  {
    prefix: 'web.',
    name: 'get',
    value: 'get',
    meta: 'method',
    score: 110,
    reference: {
      heading: 'web.get(url[,opts])',
      kind: 'method',
      detail: 'Does a GET web request, returning either an error or an object describing the result.',
      more: '<h4>web.get() documentation</h4><br><label>Example success result:</label><div style=\'white-space: pre-wrap;\'>' + jsonPrettyPrint.toHtml({
        "Code":200,
        "CodeStr":"200 OK",
        "Cookies":[],
        "Data":"{\"ip\":\"REDACTED\"}",
        "Header":{
          "Access-Control-Allow-Origin":["*"],
          "Connection":["keep-alive"],
          "Content-Length":["23"],
          "Content-Type":["application/json"],
          "Date":["Sun, 09 Jul 2017 04:45:36 GMT"],
          "Server":["Cowboy"],
          "Via":["1.1 vegur"]
        },
        "URL":"https://api.ipify.org/?format=json"}) + '</div>' + '<label>Permitted options fields:</label><div style=\'white-space: pre-wrap;\'>' + jsonPrettyPrint.toHtml({
          content_type: 'string (eg: application/json)',
          headers: {key: 'value'},
        }) + '</div>',
    },
  },
  {
    prefix: 'web.',
    name: 'post',
    value: 'post',
    meta: 'method',
    score: 110,
    reference: {
      heading: 'web.post(url[,opts])',
      kind: 'method',
      detail: 'Does a POST web request, returning either an error or an object describing the result.',
      more: '<h4>web.get() documentation</h4><br><label>Example success result:</label><div style=\'white-space: pre-wrap;\'>' + jsonPrettyPrint.toHtml({
        "Code":200,
        "CodeStr":"200 OK",
        "Cookies":[],
        "Data":"{\"ip\":\"REDACTED\"}",
        "Header":{
          "Access-Control-Allow-Origin":["*"],
          "Connection":["keep-alive"],
          "Content-Length":["23"],
          "Content-Type":["application/json"],
          "Date":["Sun, 09 Jul 2017 04:45:36 GMT"],
          "Server":["Cowboy"],
          "Via":["1.1 vegur"]
        },
        "URL":"https://api.ipify.org/?format=json"}) + '</div>' + '<label>Permitted options fields:</label><div style=\'white-space: pre-wrap;\'>' + jsonPrettyPrint.toHtml({
          content_type: 'string (eg: application/json)',
          headers: {key: 'value'},
          body: 'string',
        }) + '</div>',
    },
  },
  {
    prefix: 'web.',
    name: 'values',
    value: 'values',
    meta: 'method',
    score: 110,
    reference: {
      heading: 'web.values(<obj>)',
      kind: 'method',
      detail: 'Returns a form-encoded string, which represents the key-value pairs in the provided object.',
    },
  },
  {
    prefix: 'owner.',
    name: 'id',
    value: 'id',
    meta: 'int',
    score: 110,
    reference: {
      heading: 'owner.id',
      kind: 'int',
      detail: 'User ID of the account which owns this integration.',
    },
  },
  {
    prefix: 'owner.',
    name: 'get',
    value: 'get()',
    meta: 'User',
    score: 110,
    reference: {
      heading: 'owner.get()',
      kind: 'method',
      detail: 'Information about the account which owns this integration.',
    },
  },
  {
    prefix: 'request.',
    name: 'matched_pattern',
    value: 'matched_pattern',
    meta: 'string',
    score: 110,
    reference: {
      heading: 'request.matched_pattern',
      kind: 'string',
      detail: 'Regex which was matched to the URL of the request, and triggered the current run.',
    },
  },
  {
    prefix: 'request.',
    name: 'matched_name',
    value: 'matched_name',
    meta: 'string',
    score: 110,
    reference: {
      heading: 'request.matched_name',
      kind: 'string',
      detail: 'Name of the trigger which triggered the current run.',
    },
  },
  {
    prefix: 'request.',
    name: 'body',
    value: 'body',
    meta: 'string',
    score: 110,
    reference: {
      heading: 'request.body',
      kind: 'string',
      detail: 'Content of the web request.',
    },
  },
  {
    prefix: 'request.',
    name: 'user_agent',
    value: 'user_agent',
    meta: 'string',
    score: 110,
    reference: {
      heading: 'request.user_agent',
      kind: 'string',
      detail: 'User Agent of the HTTP request.',
    },
  },
  {
    prefix: 'request.',
    name: 'url',
    value: 'url',
    meta: 'string',
    score: 110,
    reference: {
      heading: 'request.url',
      kind: 'object',
      detail: 'Parsed URL of the request.',
    },
  },
  {
    prefix: 'request.',
    name: 'referer',
    value: 'referer',
    meta: 'string',
    score: 110,
    reference: {
      heading: 'request.referer',
      kind: 'string',
      detail: 'Referer header of the request.',
    },
  },
  {
    prefix: 'request.',
    name: 'method',
    value: 'method',
    meta: 'string',
    score: 110,
    reference: {
      heading: 'request.method',
      kind: 'string',
      detail: 'Method (GET/POST etc) of the request.',
    },
  },
  {
    prefix: 'request.',
    name: 'write',
    value: 'write()',
    meta: 'method',
    score: 111,
    reference: {
      heading: 'request.write(<data>)',
      kind: 'method',
      detail: 'Writes response data to handle the request.',
    },
  },
  {
    prefix: 'request.',
    name: 'done',
    value: 'done()',
    meta: 'method',
    score: 111,
    reference: {
      heading: 'request.done()',
      kind: 'method',
      detail: 'Finalizes the HTTP response.',
    },
  },
  {
    prefix: 'request.',
    name: 'host',
    value: 'host',
    meta: 'string',
    score: 110,
    reference: {
      heading: 'request.host',
      kind: 'string',
      detail: 'Host header of the HTTP request.',
    },
  },
  {
    prefix: 'request.',
    name: 'uri',
    value: 'uri',
    meta: 'string',
    score: 110,
    reference: {
      heading: 'request.uri',
      kind: 'string',
      detail: 'Unparsed URI of the HTTP request.',
    },
  },
  {
    prefix: 'request.',
    name: 'remote_addr',
    value: 'remote_addr',
    meta: 'string',
    score: 110,
    reference: {
      heading: 'request.remote_addr',
      kind: 'string',
      detail: 'Remote address (ip:port) of the client making the request.',
    },
  },
  {
    prefix: 'request.',
    name: 'auth',
    value: 'auth()',
    meta: 'method',
    score: 110,
    reference: {
      heading: 'request.auth()',
      kind: 'object',
      detail: 'Returns an object describing the currently authenticated Nexus user if any. If authed, \'authenticated\' is set to true.',
      more: '<h4>request.auth() object</h4><br><label>Example object for authenticated user:</label><div style=\'white-space: pre-wrap;\'>' + jsonPrettyPrint.toHtml({
        authenticated: true,
        session: {
          SessionUID: 1325,
          UID: 28,
          SID: "<session ID>",
          Created: "2017-07-08T21:29:00.417359564+10:00",
          AccessWeb: true,
          AccessAPI: true,
          AuthedVia: "PASS",
          Revoked: false
        },
        user: {
          UID: 28,
          DisplayName: "Tom",
          Username: "jsonp",
          CreatedAt: "2017-05-13T17:59:45.769387303+10:00",
          IsRobot: false,
          AdminPerms: {
             Accounts: true,
             Data: true,
             Integrations: true
          },
          Grants: null
        }
      }) + '</div>',
    },
  },
  {
    prefix: 'email.',
    name: 'gmail_addr',
    value: 'gmail_addr',
    meta: 'string',
    score: 110,
    reference: {
      heading: 'email.gmail_addr',
      kind: 'string',
      detail: 'Domain and port to communicate with Gmail.',
    },
  },
  {
    prefix: 'email.',
    name: 'send()',
    value: 'send()',
    meta: 'string',
    score: 110,
    reference: {
      heading: 'email.send()',
      kind: 'method(addr,pass,{info})',
      detail: 'Sends an email using password authentication. The info object should contain to, from, subject, body.',
    },
  },
  {
    prefix: 'kv.',
    name: 'get()',
    value: 'get()',
    meta: 'obj',
    score: 110,
    reference: {
      heading: 'kv.get(<key>)',
      kind: 'method',
      detail: 'Gets an object from the KV store. Returns null if the specified key does not exist.',
    },
  },
  {
    prefix: 'kv.',
    name: 'set()',
    value: 'set()',
    meta: 'method',
    score: 110,
    reference: {
      heading: 'kv.set(<key>,<obj>)',
      kind: 'method',
      detail: 'Saves an object in the KV store.',
    },
  },
  {
    prefix: 'fs.',
    name: 'read()',
    value: 'read()',
    meta: 'method',
    score: 110,
    reference: {
      heading: 'fs.read(<path>)',
      kind: 'method',
      detail: 'Returns the content of a file by the specified path.',
    },
  },
  {
    prefix: 'fs.',
    name: 'list()',
    value: 'list()',
    meta: 'method',
    score: 110,
    reference: {
      heading: 'fs.list(<path>)',
      kind: 'method',
      detail: 'Returns an array of files/directories in the specified path.',
      more: '<h4>file list object</h4><br><label>Example objects:</label><div style=\'white-space: pre-wrap;\'>' + jsonPrettyPrint.toHtml([
       {
          Name: "test.txt",
          ItemKind: 2,
          Modified: "2017-07-14T20:17:20+10:00",
          SourceDetail: 0
       },
       {
          Name: "test.html",
          ItemKind: 2,
          Modified: "2017-07-14T20:40:16+10:00",
          SourceDetail: 0
       }]) + '</div>',
    },
  },
  {
    prefix: 'fs.',
    name: 'isFile()',
    value: 'isFile()',
    meta: 'method',
    score: 110,
    reference: {
      heading: 'fs.isFile(<int|obj>)',
      kind: 'method',
      detail: 'Returns true/false if the ItemKind represents a file. Expects either an integer or an object returned from fs.list().',
    },
  },
  {
    prefix: 'fs.',
    name: 'isDir()',
    value: 'isDir()',
    meta: 'method',
    score: 110,
    reference: {
      heading: 'fs.isDir(<int|obj>)',
      kind: 'method',
      detail: 'Returns true/false if the ItemKind represents a directory. Expects either an integer or an object returned from fs.list().',
    },
  },
  {
    prefix: 'fs.',
    name: 'delete()',
    value: 'delete()',
    meta: 'method',
    score: 110,
    reference: {
      heading: 'fs.delete(<path>)',
      kind: 'method',
      detail: 'Deletes a file. Returns null if successful, error otherwise.',
    },
  },
  {
    prefix: 'fs.',
    name: 'write()',
    value: 'write()',
    meta: 'method',
    score: 110,
    reference: {
      heading: 'fs.write(<path>, <data>)',
      kind: 'method',
      detail: 'Writes the contents of a file, creating it if it doesnt exist. Containing folder must exist.',
    },
  },
  {
    prefix: 'datastore.',
    name: 'insert',
    value: 'insert()',
    meta: 'method',
    score: 110,
    reference: {
      heading: 'datastore.insert(<name>, <fields>)',
      kind: 'method',
      detail: 'Inserts a row into a datastore. All columns must be specified. Fields should be an object where the key is the name of the column, and the value to be inserted for that row.',
    },
  },
  {
    prefix: 'datastore.',
    name: 'editRow',
    value: 'editRow()',
    meta: 'method',
    score: 110,
    reference: {
      heading: 'datastore.editRow()',
      kind: 'method',
      detail: 'datastore.editRow(<name>, <rowID>, <fields>). Edits a row based on its rowID. Fields should be an object where the key is the name of the column, and the value to be edited for that row. You don\'t need to specify every column.',
    },
  },
  {
    prefix: 'datastore.',
    name: 'query',
    value: 'query()',
    meta: 'method',
    score: 110,
    reference: {
      heading: 'datastore.query()',
      kind: 'method',
      detail: 'Runs a query against the named datastore, returning results or an error. datastore.query(<datastore> [, <filters> [, <limit>, <offset>]])',
      more: '<h4>datastore.query()</h4><br><label>Querying the \'Test\' datastore:</label><div style=\'white-space: pre-wrap;\'>' +
      'datastore.query("Test", ' + jsonPrettyPrint.toHtml([
        {value: "kek", column: "text", condition: "!="}
      ]) + ');</div><br><label>Example response:</label><div style=\'white-space: pre-wrap;\'>' + jsonPrettyPrint.toHtml(
        {
           results: [
              {
                 num: 1503121824,
                 rowid: 1,
                 text: "m8",
                 time: "2017-08-19T16:15:27.121824074+01:00"
              },
           ],
           success: true,
        }
      ) + ');</div><br><br><p>Note the success attribute will always be true if the lookup succeeded. The rowid is a unique identifier for that row, and is always populated.</p>',
    },
  },
  {
    prefix: 't.',
    name: 'now',
    value: 'now()',
    meta: 'method',
    score: 110,
    reference: {
      heading: 't.now()',
      kind: 'method',
      detail: 'Returns the current time as a time object.',
    },
  },
  {
    prefix: 't.',
    name: 'unix',
    value: 'unix()',
    meta: 'method',
    score: 110,
    reference: {
      heading: 't.unix(<seconds>)',
      kind: 'method',
      detail: 'Converts the given epoch into a time object.',
    },
  },
  {
    prefix: 't.',
    name: 'nano',
    value: 'nano()',
    meta: 'method',
    score: 110,
    reference: {
      heading: 't.nano(<nano-seconds>)',
      kind: 'method',
      detail: 'Converts the nanoseconds-since-epoch into a time object.',
    },
  },
  {
    prefix: 't.',
    name: 'addDate',
    value: 'addDate()',
    meta: 'method',
    score: 110,
    reference: {
      heading: 't.addDate()',
      kind: 'method',
      detail: 't.addDate(<timeObj>[, years, [months, [days]]]). Adds days/months/years onto the given time object, returning an updated time object.',
    },
  },
  {
    prefix: 't.',
    name: 'addTime',
    value: 'addTime()',
    meta: 'method',
    score: 110,
    reference: {
      heading: 't.addTime()',
      kind: 'method',
      detail: 't.addTime(<timeObj>[, hours, [minutes, [seconds]]]). Adds hours/minutes/seconds onto the given time object, returning an updated time object.',
    },
  },
  {
    prefix: 'gcp.',
    name: 'load_service_credential',
    value: 'load_service_credential()',
    meta: 'method',
    score: 110,
    reference: {
      heading: 'gcp.load_service_credential()',
      kind: 'method',
      detail: 'gcp.load_service_credential(<path_to_cred_file>). Parses a credential file for a service account, returning a configuration which can be used to call APIs.',
    },
  },
  {
    prefix: 'gcp.',
    name: 'compute_client',
    value: 'compute_client()',
    meta: 'method',
    score: 110,
    reference: {
      heading: 'gcp.compute_client()',
      kind: 'method',
      detail: 'gcp.compute_client(<config>). Consumes a service account configuration (returned by gcp.load_service_credential()), and returns an object who\'s methods perform actions on GCP.',
      more: '<h4>Compute client methods</h4><br><label>.list(&lt;GCP project name&gt;, &lt;GCP zone&gt;)</label><br>Returns an error or a list of instances.'+
      'Instances are structured like <a href="https://godoc.org/google.golang.org/api/compute/v1#Instance">this</a>.',
    },
  },
]

function startsWith(s, prefix) {
  if (prefix === '' || !prefix){
    return true;
  }
  return s.substring(0, prefix.length) === prefix;
}

function topSort(word, set) {
  var output = [];
  for (var i = 0; i < set.length; i++) {
    if (startsWith(set[i].prefix + set[i].name, word)){
      output.unshift(set[i]);
    } else {
      output.push(set[i]);
    }
  }
  return output;
}

app.controller('EditorController', ["$scope", "$rootScope", "$http", function ($scope, $rootScope, $http) {
  $scope.editorObj = null;
  $scope.langTools = null;
  $scope.codeSuggestions = [];
  $scope.datastoreSuggestions = null;
  $scope.runnable = null;
  $scope.lastSaved = null;
  $scope.loading = false;
  $scope.error = null;

  $scope.doEditorAutocomplete = function(editor, session, pos, prefix, callback) {
    console.log(session, pos, prefix, pos.column, pos.line);
    fullLine = session.getLine(pos.row);
    console.log(fullLine);

    spl = fullLine.substring(0, pos.column).split(/,| |\(|=/);
    lastWord = spl[spl.length-1];
    //console.log(lastWord);

    if (fullLine == ""){ //nothing typed - show globals
      $scope.codeSuggestions = codeGlobals;
      $scope.datastoreSuggestions = null;
      $scope.$digest();
      return callback(null, codeGlobals);
    }

    // show information about datastores
    if (fullLine.match('.*datastore\\.(insert|query).*')){
      $scope.datastoreSuggestions = {loading: true};
      var datastoreExtract = fullLine.match('.*datastore\\.(insert|query)\\("(.*)".*');
      if (datastoreExtract) { //show information on only the specified datastore
        $http({
          method: 'GET',
          url: '/web/v1/data/list'
        }).then(function successCallback(response) {
          $scope.datastoreSuggestions = {loading: false, datastores: response.data, filter: datastoreExtract[1]};
          for (var i = 0; i < response.data.length; i++) {
            if (response.data[i].Name == datastoreExtract[2]){
              $scope.datastoreSuggestions.cols = response.data[i].Cols;
            }
          }
        }, function errorCallback(response) {
          $scope.datastoreSuggestions = {loading: false};
          console.log(response);
        });
      } else { //show information on all accessible datastores
        $http({
          method: 'GET',
          url: '/web/v1/data/list'
        }).then(function successCallback(response) {
          $scope.datastoreSuggestions = {loading: false, datastores: response.data};
        }, function errorCallback(response) {
          $scope.datastoreSuggestions = {loading: false};
          console.log(response);
        });
      }
    } else {
      $scope.datastoreSuggestions = null;
    }

    if (lastWord.indexOf('.') === -1) { //no dots
      $scope.codeSuggestions = topSort(lastWord, codeGlobals.filter(function(g){return startsWith(g.name, lastWord)}));
      $scope.$digest();
      return callback(null, $scope.codeSuggestions);
    } else { //try and resolve subs
      $scope.codeSuggestions = topSort(lastWord, codeSubs.filter(function(g){return startsWith(lastWord, g.prefix)}));
      $scope.$digest();
      return callback(null, $scope.codeSuggestions);
    }
    callback(null, []);
  };

  $scope.triggerIcon = function(trigger){
    switch (trigger.Kind) {
      case 'CRON':
        return 'schedule';
      case 'HTTP':
        return 'http';
    }
    return '?'
  }

  $scope.save = function(){
    $scope.loading = true;
    $scope.error = null;
    $http({
      method: 'POST',
      url: '/web/v1/integrations/code/save',
      data: {
        UID: $scope.runnable.UID,
        Code: $scope.editorObj.getValue(),
      },
    }).then(function successCallback(response) {
      $scope.loading = false;
      $scope.lastSaved = new Date();
    }, function errorCallback(response) {
      $scope.loading = false;
      $scope.error = response;
    });
  }

  $scope.moreReference = function(moreObj) {
    $rootScope.$broadcast('documentation-modal', {docs: moreObj});
  }

  $scope.datastoreDatatypeToString = function(datatype){
    switch (datatype){
      case 0:
        return "int";
      case 2:
        return "float";
      case 3:
        return "string";
      case 4:
        return "blob";
      case 5:
        return "time";
    }
  }


  $rootScope.$on('integration-code-editor', function(event, args) {
    $scope.runnable = args.runnable;
  });

  $rootScope.$on('page-change', function(event, args) {
    if (args.page == 'integration-editor'){
      $scope.lastSaved = new Date();

      if (!$scope.editorObj) {
        $scope.langTools = ace.require("ace/ext/language_tools");
        $scope.editorObj = ace.edit("codeEditor");
        var JavaScriptMode = ace.require("ace/mode/javascript").Mode;
        $scope.editorObj.session.setMode(new JavaScriptMode());
        $scope.editorObj.setOptions({
            enableBasicAutocompletion: true
        });
        $scope.langTools.addCompleter({getCompletions: $scope.doEditorAutocomplete});
        $scope.editorObj.setTheme("ace/theme/github");
        $scope.editorObj.getSession().on('change', function(){
          if ($scope.codeSuggestions.length > 0){
            console.log('clearing suggestions');
            $scope.codeSuggestions = [];
            $scope.$digest();
          }
        });
        $scope.editorObj.commands.addCommand({
          name: 'saveFile',
          bindKey: {
            win: 'Ctrl-S',
            mac: 'Command-S',
            sender: 'editor|cli'
          },
          exec: function(env, args, request) {
            $scope.save();
          }
        });
        $scope.editorObj.commands.addCommand({
          name: 'browserDocs',
          bindKey: {
            win: 'Ctrl-B',
            mac: 'Command-B',
            sender: 'editor|cli'
          },
          exec: function(env, args, request) {
            $rootScope.$broadcast('documentation-modal', {docs: browserDocs});
            $rootScope.$digest();
          }
        });
      }
      if ($scope.runnable){
        $scope.editorObj.setValue($scope.runnable.Content);
        $scope.editorObj.gotoLine(0,0);
      }
      $scope.editorObj.resize();
    }
  });

}]);
