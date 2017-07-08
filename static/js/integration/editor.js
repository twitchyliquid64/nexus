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
      detail: 'Cause of the execution. Typically "manual", "CRON" etc',
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
      heading: 'web.get()',
      kind: 'method(url,cb,cb)',
      detail: 'Does a GET web request. The first callback is called with the response as a string on success, and the second callback is called on failure.',
    },
  },
  {
    prefix: 'web.',
    name: 'post',
    value: 'post',
    meta: 'method',
    score: 110,
    reference: {
      heading: 'web.post()',
      kind: 'method(url,[data],cb,cb)',
      detail: 'This method is fucked and will be fixed.',
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
]

function startsWith(s, prefix) {
  if (prefix === '' || !prefix){
    return true;
  }
  return s.substring(0, prefix.length) === prefix;
}

app.controller('EditorController', ["$scope", "$rootScope", "$http", function ($scope, $rootScope, $http) {
  $scope.editorObj = null;
  $scope.langTools = null;
  $scope.codeSuggestions = [];
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
      $scope.$digest();
      return callback(null, codeGlobals);
    }

    if (lastWord.indexOf('.') === -1) { //no dots
      $scope.codeSuggestions = codeGlobals.filter(function(g){return startsWith(g.name, lastWord)});
      $scope.$digest();
      return callback(null, $scope.codeSuggestions);
    } else { //try and resolve subs
      $scope.codeSuggestions = codeSubs.filter(function(g){return startsWith(lastWord, g.prefix)});
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
