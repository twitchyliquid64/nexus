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
  }
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
      }
      if ($scope.runnable){
        $scope.editorObj.setValue($scope.runnable.Content);
        $scope.editorObj.gotoLine(0,0)
      }
      $scope.editorObj.resize();
    }
  });

}]);
