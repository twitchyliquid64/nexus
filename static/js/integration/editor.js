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
      detail: 'Prints arguments to the console.',
    },
  },
  {
    prefix: 'console.',
    name: 'assert',
    value: 'assert',
    meta: 'trace on failure',
    score: 100,
    reference: {
      heading: 'console.assert()',
      kind: 'method',
      detail: 'Prints result + stack trace to the console if the assertion was false.',
    },
  }
]

function startsWith(s, prefix) {
  if (prefix === '' || !prefix){
    return true;
  }
  return s.substring(0, prefix.length) === prefix;
}

app.controller('EditorController', ["$scope", "$rootScope", function ($scope, $rootScope) {
  $scope.editorObj = null;
  $scope.langTools = null;
  $scope.codeSuggestions = [];

  $scope.doEditorAutocomplete = function(editor, session, pos, prefix, callback) {
    console.log(session, pos, prefix, pos.column, pos.line);
    fullLine = session.getLine(pos.row);
    console.log(fullLine);

    spl = fullLine.split(" ");
    lastWord = spl[spl.length-1];

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

  $rootScope.$on('page-change', function(event, args) {
    if (args.page == 'integration-editor'){
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
      }
      $scope.editorObj.resize();
    }
  });

}]);
